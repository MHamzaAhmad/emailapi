package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	internaldns "github.com/emailapi/api/internal/dns"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/ses"
	rediscache "github.com/emailapi/api/internal/repository/redis"
	tbrepo "github.com/emailapi/api/internal/repository/tinybird"
	"github.com/emailapi/api/internal/validation"
)

const (
	// defaultRegion is the default AWS region for SES.
	defaultRegion = "us-east-1"

	// VerifyCooldown is the minimum time between verification calls per domain.
	VerifyCooldown = 30 * time.Second

	// StaleThreshold is how old data can be before auto-refresh on read.
	StaleThreshold = 5 * time.Minute
)

// DomainService handles domain business logic.
type DomainService struct {
	store             Store
	ses               ses.Client
	dns               *internaldns.Validator
	cache             rediscache.DomainCacheInterface
	activity          tbrepo.ActivityRepositoryInterface
	reputationChecker validation.ReputationChecker
	region            string
	configurationSet  string // SES configuration set for notifications
}

// NewDomainService creates a new DomainService.
func NewDomainService(store Store, sesClient ses.Client, cache rediscache.DomainCacheInterface, activity tbrepo.ActivityRepositoryInterface, reputationChecker validation.ReputationChecker, region, configurationSet string) *DomainService {
	if region == "" {
		region = defaultRegion
	}
	return &DomainService{
		store:             store,
		ses:               sesClient,
		dns:               internaldns.NewValidator(),
		cache:             cache,
		activity:          activity,
		reputationChecker: reputationChecker,
		region:            region,
		configurationSet:  configurationSet,
	}
}

// canVerify checks if enough time has passed since the last verification.
func (s *DomainService) canVerify(d *domain.SendingDomain) (bool, time.Time) {
	if d.LastCheckedAt == nil {
		return true, time.Time{}
	}
	nextAllowed := d.LastCheckedAt.Add(VerifyCooldown)
	if time.Now().After(nextAllowed) {
		return true, time.Time{}
	}
	return false, nextAllowed
}

// isStale checks if domain data is outdated.
func (s *DomainService) isStale(d *domain.SendingDomain) bool {
	if d.LastCheckedAt == nil {
		return true
	}
	return time.Now().After(d.LastCheckedAt.Add(StaleThreshold))
}

// Add registers a new sending domain with AWS SES.
// MAIL FROM is auto-configured as mail.{domain}.
func (s *DomainService) Add(ctx context.Context, userID, domainName string) (*domain.DomainWithDetails, error) {
	// Check reputation first - suspended users cannot add domains
	if s.reputationChecker != nil {
		if err := s.reputationChecker.CheckSendPermission(ctx, userID); err != nil {
			return nil, err
		}
	}

	// Validate domain name
	domainName = strings.TrimSpace(strings.ToLower(domainName))
	if domainName == "" {
		return nil, fmt.Errorf("domain name is required")
	}

	// Check if domain already exists
	existing, _ := s.store.Domains().GetByDomainName(ctx, userID, domainName)
	if existing != nil {
		return nil, fmt.Errorf("domain %s already exists", domainName)
	}

	// Create identity in SES
	result, err := s.ses.CreateEmailIdentity(ctx, domainName)
	if err != nil {
		return nil, fmt.Errorf("failed to create email identity: %w", err)
	}

	// Auto-configure MAIL FROM domain
	mailFromDomain := "mail." + domainName
	if err := s.ses.PutEmailIdentityMailFromAttributes(ctx, domainName, mailFromDomain); err != nil {
		// Log warning but don't fail - MAIL FROM is optional for basic sending
		log.Warn().Err(err).Str("domain", domainName).Msg("failed to set MAIL FROM, continuing without it")
		mailFromDomain = ""
	}

	// Apply configuration set for email event notifications (delivery, bounce, complaint)
	if s.configurationSet != "" {
		if err := s.ses.PutEmailIdentityConfigurationSetAttributes(ctx, domainName, s.configurationSet); err != nil {
			log.Warn().Err(err).Str("domain", domainName).Str("configSet", s.configurationSet).
				Msg("failed to apply configuration set, notifications may not work")
		}
	}

	// Create domain entity
	now := time.Now()
	d := &domain.SendingDomain{
		ID:                 uuid.New().String(),
		UserID:             userID,
		Domain:             domainName,
		Status:             domain.DomainStatusPending,
		VerifiedForSending: result.VerifiedForSendingStatus,
		DkimTokens:         result.DkimTokens,
		DkimStatus:         toDomainStatus(result.DkimStatus),
		MailFromDomain:     mailFromDomain,
		MailFromStatus:     domain.DomainStatusPending,
		Region:             s.region,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Store domain
	if err := s.store.Domains().Create(ctx, d); err != nil {
		// Try to clean up the SES identity if DB storage fails
		_ = s.ses.DeleteEmailIdentity(ctx, domainName)
		return nil, fmt.Errorf("failed to store domain: %w", err)
	}

	// Invalidate user's domain list cache
	if s.cache != nil {
		_ = s.cache.InvalidateByUserID(ctx, userID)
	}

	// Log activity
	if s.activity != nil {
		_ = s.activity.LogDomain(ctx, userID, d.ID, "create", "success", fmt.Sprintf("Domain %s created", domainName))
	}

	return s.buildDomainWithDetails(d), nil
}

// Get retrieves a domain by ID with authorization check.
// Auto-refreshes if data is stale (>5 min).
func (s *DomainService) Get(ctx context.Context, userID, domainID string) (*domain.DomainWithDetails, error) {
	// Try cache first
	if s.cache != nil {
		if cached, _ := s.cache.GetByID(ctx, domainID); cached != nil {
			if cached.UserID == userID {
				return s.buildDomainWithDetails(cached), nil
			}
		}
	}

	d, err := s.store.Domains().GetByID(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}

	if d.UserID != userID {
		return nil, fmt.Errorf("domain not found")
	}

	// Auto-refresh if stale and cooldown allows
	if s.isStale(d) {
		if canVerify, _ := s.canVerify(d); canVerify {
			refreshed, err := s.refreshFromSES(ctx, d)
			if err == nil {
				d = refreshed
			}
		}
	}

	// Cache the result
	if s.cache != nil {
		_ = s.cache.SetByID(ctx, d)
	}

	return s.buildDomainWithDetails(d), nil
}

// List retrieves all domains for a user with pagination.
// Uses cache for unpaginated requests (page <= 1, pageSize >= 100) for performance.
func (s *DomainService) List(ctx context.Context, userID string, page, pageSize int) ([]*domain.DomainWithDetails, int, error) {
	// Default pagination if not specified
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 25 // Reasonable default
	}

	// Use cache for "list all" requests (first page with large page size)
	// This optimizes the common case of loading domains in UI
	useCache := page == 1 && pageSize >= 25

	if useCache && s.cache != nil {
		if cached, _ := s.cache.GetByUserID(ctx, userID); cached != nil {
			// Return cached results (limited to pageSize for consistency)
			limit := len(cached)
			if pageSize < limit {
				limit = pageSize
			}

			result := make([]*domain.DomainWithDetails, limit)
			var wg sync.WaitGroup
			for i := 0; i < limit; i++ {
				wg.Add(1)
				go func(idx int, dom *domain.SendingDomain) {
					defer wg.Done()
					result[idx] = s.buildDomainWithDetails(dom)
				}(i, cached[i])
			}
			wg.Wait()
			return result, len(cached), nil
		}
	}

	offset := (page - 1) * pageSize

	domains, err := s.store.Domains().GetByUserID(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list domains: %w", err)
	}

	// Get total count for pagination
	total, err := s.store.Domains().CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count domains: %w", err)
	}

	// Cache for first page requests
	if useCache && s.cache != nil && page == 1 {
		_ = s.cache.SetByUserID(ctx, userID, domains)
	}

	result := make([]*domain.DomainWithDetails, len(domains))
	var wg sync.WaitGroup
	for i, d := range domains {
		wg.Add(1)
		go func(idx int, dom *domain.SendingDomain) {
			defer wg.Done()
			result[idx] = s.buildDomainWithDetails(dom)
		}(i, d)
	}
	wg.Wait()

	return result, total, nil
}

// GetVerifiedDomainForSending retrieves a domain by name for sender validation.
// Uses a fast-path Redis cache for high-frequency lookups during email sending.
func (s *DomainService) GetVerifiedDomainForSending(ctx context.Context, userID, domainName string) (*domain.SendingDomain, error) {
	// Fast-path: check Redis cache first (sub-ms latency)
	if s.cache != nil {
		if cached, _ := s.cache.GetSendingStatus(ctx, userID, domainName); cached != nil {
			return cached, nil
		}
	}

	// Cache miss: query database
	d, err := s.store.Domains().GetByDomainName(ctx, userID, domainName)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}

	// Populate cache for next lookup
	if s.cache != nil {
		_ = s.cache.SetSendingStatus(ctx, d)
	}

	return d, nil
}

// Delete removes a domain from the account and SES.
func (s *DomainService) Delete(ctx context.Context, userID, domainID string) error {
	d, err := s.store.Domains().GetByID(ctx, domainID)
	if err != nil {
		return fmt.Errorf("failed to get domain: %w", err)
	}

	if d.UserID != userID {
		return fmt.Errorf("domain not found")
	}

	// Delete from SES
	if err := s.ses.DeleteEmailIdentity(ctx, d.Domain); err != nil {
		return fmt.Errorf("failed to delete email identity: %w", err)
	}

	// Delete from database
	if err := s.store.Domains().Delete(ctx, domainID); err != nil {
		return fmt.Errorf("failed to delete domain: %w", err)
	}

	// Invalidate cache (including fast-path sending cache)
	if s.cache != nil {
		_ = s.cache.InvalidateAll(ctx, domainID, userID)
		_ = s.cache.InvalidateSendingStatus(ctx, userID, d.Domain)
	}

	// Log activity
	if s.activity != nil {
		_ = s.activity.LogDomain(ctx, userID, domainID, "delete", "success", fmt.Sprintf("Domain %s deleted", d.Domain))
	}

	return nil
}

// Verify forces a fresh check of DNS records and SES status.
func (s *DomainService) Verify(ctx context.Context, userID, domainID string) (*domain.VerifyResult, error) {
	d, err := s.store.Domains().GetByID(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}

	if d.UserID != userID {
		return nil, fmt.Errorf("domain not found")
	}

	// Check cooldown to prevent SES rate limiting
	canVerify, nextRetry := s.canVerify(d)
	if !canVerify {
		return &domain.VerifyResult{
			Domain:       s.buildDomainWithDetails(d),
			WasRefreshed: false,
			NextRetryAt:  &nextRetry,
			Message:      fmt.Sprintf("Rate limited. Next check allowed in %s", time.Until(nextRetry).Round(time.Second)),
		}, nil
	}

	// Refresh from SES
	d, err = s.refreshFromSES(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh from SES: %w", err)
	}

	// Build domain with records (without summary yet)
	records := s.buildDomainRecords(d)
	details := &domain.DomainWithDetails{
		SendingDomain: d,
		Records:       records,
	}

	// Perform live DNS validation
	s.validateDNS(ctx, details)

	// Build summary AFTER DNS validation so it has correct statuses
	details.Summary = s.buildSummary(d, records)

	// Update domain status based on DNS results
	s.updateStatusFromRecords(d, details)

	// Save updated status
	if err := s.store.Domains().Update(ctx, d); err != nil {
		return nil, fmt.Errorf("failed to update domain: %w", err)
	}

	// Log verification activity
	if s.activity != nil {
		status := "success"
		if d.Status == domain.DomainStatusFailed {
			status = "failed"
		}
		_ = s.activity.LogDomain(ctx, userID, domainID, "verify", status, fmt.Sprintf("Domain verification: %s", d.Status))
	}

	return &domain.VerifyResult{
		Domain:       details,
		WasRefreshed: true,
		Message:      s.getVerifyMessage(details),
	}, nil
}

// refreshFromSES fetches current status from AWS SES.
func (s *DomainService) refreshFromSES(ctx context.Context, d *domain.SendingDomain) (*domain.SendingDomain, error) {
	result, err := s.ses.GetEmailIdentity(ctx, d.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get email identity: %w", err)
	}

	now := time.Now()
	d.VerifiedForSending = result.VerifiedForSendingStatus
	d.DkimStatus = toDomainStatus(result.DkimStatus)
	if result.MailFromDomain != "" {
		d.MailFromDomain = result.MailFromDomain
		d.MailFromStatus = toDomainStatus(result.MailFromStatus)
	}
	d.UpdatedAt = now
	d.LastCheckedAt = &now

	if err := s.store.Domains().Update(ctx, d); err != nil {
		return nil, fmt.Errorf("failed to update domain: %w", err)
	}

	return d, nil
}

// validateDNS performs live DNS lookups and updates record statuses.
func (s *DomainService) validateDNS(ctx context.Context, details *domain.DomainWithDetails) {
	if details.Records == nil {
		return
	}

	// Build expected records list
	var expected []internaldns.ExpectedRecord

	for _, rec := range details.Records.DkimRecords {
		expected = append(expected, internaldns.ExpectedRecord{
			Type:  rec.Type,
			Name:  rec.Name,
			Value: rec.Value,
		})
	}

	if details.Records.SpfRecord != nil {
		expected = append(expected, internaldns.ExpectedRecord{
			Type:  details.Records.SpfRecord.Type,
			Name:  details.Records.SpfRecord.Name,
			Value: details.Records.SpfRecord.Value,
		})
	}

	if details.Records.DmarcRecord != nil {
		expected = append(expected, internaldns.ExpectedRecord{
			Type:  details.Records.DmarcRecord.Type,
			Name:  details.Records.DmarcRecord.Name,
			Value: details.Records.DmarcRecord.Value,
		})
	}

	for _, rec := range details.Records.MxRecords {
		expected = append(expected, internaldns.ExpectedRecord{
			Type:     rec.Type,
			Name:     rec.Name,
			Value:    rec.Value,
			Priority: rec.Priority,
		})
	}

	for _, rec := range details.Records.MailFromRecords {
		expected = append(expected, internaldns.ExpectedRecord{
			Type:     rec.Type,
			Name:     rec.Name,
			Value:    rec.Value,
			Priority: rec.Priority,
		})
	}

	// Perform DNS validation
	result := s.dns.ValidateRecords(ctx, expected)

	// Update record statuses
	resultIdx := 0
	for i := range details.Records.DkimRecords {
		if resultIdx < len(result.Records) {
			details.Records.DkimRecords[i].Status = toRecordStatus(result.Records[resultIdx].Status)
			details.Records.DkimRecords[i].DiscoveredValue = result.Records[resultIdx].DiscoveredValue
			resultIdx++
		}
	}

	if details.Records.SpfRecord != nil && resultIdx < len(result.Records) {
		details.Records.SpfRecord.Status = toRecordStatus(result.Records[resultIdx].Status)
		details.Records.SpfRecord.DiscoveredValue = result.Records[resultIdx].DiscoveredValue
		resultIdx++
	}

	if details.Records.DmarcRecord != nil && resultIdx < len(result.Records) {
		details.Records.DmarcRecord.Status = toRecordStatus(result.Records[resultIdx].Status)
		details.Records.DmarcRecord.DiscoveredValue = result.Records[resultIdx].DiscoveredValue
		resultIdx++
	}

	for i := range details.Records.MxRecords {
		if resultIdx < len(result.Records) {
			details.Records.MxRecords[i].Status = toRecordStatus(result.Records[resultIdx].Status)
			details.Records.MxRecords[i].DiscoveredValue = result.Records[resultIdx].DiscoveredValue
			resultIdx++
		}
	}

	for i := range details.Records.MailFromRecords {
		if resultIdx < len(result.Records) {
			details.Records.MailFromRecords[i].Status = toRecordStatus(result.Records[resultIdx].Status)
			details.Records.MailFromRecords[i].DiscoveredValue = result.Records[resultIdx].DiscoveredValue
			resultIdx++
		}
	}
}

// updateStatusFromRecords updates domain status based on record verification.
func (s *DomainService) updateStatusFromRecords(d *domain.SendingDomain, details *domain.DomainWithDetails) {
	if details.Summary.CanSend {
		if details.Summary.RecordsPending == 0 {
			d.Status = domain.DomainStatusReady
		} else {
			d.Status = domain.DomainStatusDegraded
		}
	} else if details.Summary.RecordsConfigured > 0 {
		d.Status = domain.DomainStatusVerifying
	} else {
		d.Status = domain.DomainStatusPending
	}
}

// getVerifyMessage returns a human-readable verification result message.
func (s *DomainService) getVerifyMessage(details *domain.DomainWithDetails) string {
	if details.Summary.CanSend && details.Summary.RecordsPending == 0 {
		return "All records verified! Your domain is ready to send emails."
	}
	if details.Summary.CanSend {
		return fmt.Sprintf("Domain verified for sending. %d optional records still pending.", details.Summary.RecordsPending)
	}
	if details.Summary.RecordsConfigured > 0 {
		return fmt.Sprintf("%d of %d records configured. Waiting for remaining records.",
			details.Summary.RecordsConfigured,
			details.Summary.RecordsConfigured+details.Summary.RecordsPending)
	}
	return "No records configured yet. Add the DNS records below to verify your domain."
}

// buildDomainWithDetails creates a complete domain response.
func (s *DomainService) buildDomainWithDetails(d *domain.SendingDomain) *domain.DomainWithDetails {
	records := s.buildDomainRecords(d)
	summary := s.buildSummary(d, records)

	return &domain.DomainWithDetails{
		SendingDomain: d,
		Summary:       summary,
		Records:       records,
	}
}

// buildSummary creates a human-readable summary.
func (s *DomainService) buildSummary(d *domain.SendingDomain, records *domain.DomainRecords) *domain.DomainSummary {
	// Count records
	pending := 0
	configured := 0

	countRecord := func(r *domain.DnsRecord) {
		if r == nil {
			return
		}
		if r.Status == domain.RecordStatusFound {
			configured++
		} else {
			pending++
		}
	}

	for _, r := range records.DkimRecords {
		countRecord(&r)
	}
	countRecord(records.SpfRecord)
	countRecord(records.DmarcRecord)
	for _, r := range records.MxRecords {
		countRecord(&r)
	}
	for _, r := range records.MailFromRecords {
		countRecord(&r)
	}

	canSend := d.VerifiedForSending
	// Check if inbound MX records are configured for receiving emails
	canReceive := false
	for _, mx := range records.MxRecords {
		if mx.RecordType == domain.RecordTypeMXInbound && mx.Status == domain.RecordStatusFound {
			canReceive = true
			break
		}
	}

	var message, nextAction string
	if canSend && pending == 0 {
		message = "Ready to send emails!"
		nextAction = "NONE"
	} else if canSend {
		message = fmt.Sprintf("Can send emails. %d optional records pending.", pending)
		nextAction = "NONE"
	} else if configured > 0 {
		message = fmt.Sprintf("Waiting for DNS propagation. %d of %d records found.", configured, configured+pending)
		nextAction = "WAIT"
	} else {
		message = fmt.Sprintf("Add %d DNS records to start sending emails.", pending)
		nextAction = "CONFIGURE_DNS"
	}

	return &domain.DomainSummary{
		Message:           message,
		NextAction:        nextAction,
		RecordsPending:    pending,
		RecordsConfigured: configured,
		CanSend:           canSend,
		CanReceive:        canReceive,
	}
}

// buildDomainRecords creates the DNS records structure.
func (s *DomainService) buildDomainRecords(d *domain.SendingDomain) *domain.DomainRecords {
	records := &domain.DomainRecords{
		DkimRecords:     s.buildDkimRecords(d),
		SpfRecord:       s.buildSpfRecord(d),
		DmarcRecord:     s.buildDmarcRecord(d),
		MxRecords:       s.buildMxInboundRecords(d),
		MailFromRecords: s.buildMailFromRecords(d),
	}

	return records
}

// buildDkimRecords creates DKIM CNAME records from tokens.
// Records always start with pending status - DNS validation will update the actual status.
func (s *DomainService) buildDkimRecords(d *domain.SendingDomain) []domain.DnsRecord {
	if len(d.DkimTokens) == 0 {
		return nil
	}

	records := make([]domain.DnsRecord, len(d.DkimTokens))

	for i, token := range d.DkimTokens {
		fullName := fmt.Sprintf("%s._domainkey.%s", token, d.Domain)
		records[i] = domain.DnsRecord{
			Type:         "CNAME",
			Name:         fullName,
			Value:        fmt.Sprintf("%s.dkim.amazonses.com", token),
			RecordType:   domain.RecordTypeDKIM,
			Status:       domain.RecordStatusPending, // Always start pending, DNS validation will update
			NameShort:    fmt.Sprintf("%s._domainkey", token),
			Instructions: "Add this CNAME record for DKIM email signing.",
		}
	}

	return records
}

// buildSpfRecord creates the SPF TXT record.
func (s *DomainService) buildSpfRecord(d *domain.SendingDomain) *domain.DnsRecord {
	return &domain.DnsRecord{
		Type:         "TXT",
		Name:         d.Domain,
		Value:        "v=spf1 include:amazonses.com ~all",
		RecordType:   domain.RecordTypeSPF,
		Status:       domain.RecordStatusPending,
		NameShort:    "@",
		Instructions: "Add this TXT record to authorize Amazon SES to send emails.",
	}
}

// buildDmarcRecord creates the DMARC TXT record.
func (s *DomainService) buildDmarcRecord(d *domain.SendingDomain) *domain.DnsRecord {
	return &domain.DnsRecord{
		Type:         "TXT",
		Name:         fmt.Sprintf("_dmarc.%s", d.Domain),
		Value:        fmt.Sprintf("v=DMARC1; p=none; rua=mailto:dmarc@%s", d.Domain),
		RecordType:   domain.RecordTypeDMARC,
		Status:       domain.RecordStatusPending,
		NameShort:    "_dmarc",
		Instructions: "Add this TXT record for DMARC policy. Start with p=none, then increase to p=quarantine or p=reject.",
	}
}

// buildMxInboundRecords creates MX records for inbound email.
func (s *DomainService) buildMxInboundRecords(d *domain.SendingDomain) []domain.DnsRecord {
	return []domain.DnsRecord{
		{
			Type:         "MX",
			Name:         d.Domain,
			Value:        fmt.Sprintf("inbound-smtp.%s.amazonaws.com", s.region),
			Priority:     10,
			RecordType:   domain.RecordTypeMXInbound,
			Status:       domain.RecordStatusPending,
			NameShort:    "@",
			Instructions: "Add this MX record to receive inbound emails via Amazon SES.",
		},
	}
}

// buildMailFromRecords creates MX and SPF records for custom MAIL FROM.
// Records always start with pending status - DNS validation will update the actual status.
func (s *DomainService) buildMailFromRecords(d *domain.SendingDomain) []domain.DnsRecord {
	if d.MailFromDomain == "" {
		return nil
	}

	// Extract short name (e.g., "mail" from "mail.example.com")
	shortName := strings.TrimSuffix(d.MailFromDomain, "."+d.Domain)

	return []domain.DnsRecord{
		{
			Type:         "MX",
			Name:         d.MailFromDomain,
			Value:        fmt.Sprintf("feedback-smtp.%s.amazonses.com", s.region),
			Priority:     10,
			RecordType:   domain.RecordTypeMailFromMX,
			Status:       domain.RecordStatusPending, // Always start pending, DNS validation will update
			NameShort:    shortName,
			Instructions: "Add this MX record for bounce handling.",
		},
		{
			Type:         "TXT",
			Name:         d.MailFromDomain,
			Value:        "v=spf1 include:amazonses.com ~all",
			RecordType:   domain.RecordTypeMailFromSPF,
			Status:       domain.RecordStatusPending, // Always start pending, DNS validation will update
			NameShort:    shortName,
			Instructions: "Add this SPF record for MAIL FROM authentication.",
		},
	}
}

// toDomainStatus converts SES status string to DomainStatus.
func toDomainStatus(status string) domain.DomainStatus {
	switch strings.ToUpper(status) {
	case "SUCCESS":
		return domain.DomainStatusReady
	case "FAILED":
		return domain.DomainStatusFailed
	case "TEMPORARY_FAILURE":
		return domain.DomainStatusDegraded
	default:
		return domain.DomainStatusPending
	}
}

// toRecordStatus converts DNS validation status to RecordStatus.
func toRecordStatus(status internaldns.RecordStatus) domain.RecordStatus {
	switch status {
	case internaldns.RecordStatusFound:
		return domain.RecordStatusFound
	case internaldns.RecordStatusMismatch:
		return domain.RecordStatusMismatch
	case internaldns.RecordStatusMissing:
		return domain.RecordStatusMissing
	default:
		return domain.RecordStatusPending
	}
}

// Note: toRecordStatusFromDomainStatus was removed.
// Record status should always be determined by live DNS validation,
// not derived from SES domain status.
