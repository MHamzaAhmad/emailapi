package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/ses"
)

const (
	// defaultRegion is the default AWS region for SES.
	defaultRegion = "us-east-1"

	// VerifyCooldown is the minimum time between SES verification calls per domain.
	// This prevents abuse and rate limiting from AWS SES.
	VerifyCooldown = 30 * time.Second

	// StaleThreshold is how old verification data can be before auto-refresh is allowed.
	// Used for smart refresh on read operations.
	StaleThreshold = 5 * time.Minute
)

// DomainService handles domain business logic.
type DomainService struct {
	store  Store
	ses    ses.Client
	region string
}

// NewDomainService creates a new DomainService.
func NewDomainService(store Store, sesClient ses.Client, region string) *DomainService {
	if region == "" {
		region = defaultRegion
	}
	return &DomainService{
		store:  store,
		ses:    sesClient,
		region: region,
	}
}

// canVerify checks if enough time has passed since the last verification.
// Returns true if verification is allowed, false otherwise with the next allowed time.
func (s *DomainService) canVerify(d *domain.SendingDomain) (bool, time.Time) {
	if d.LastVerifiedAt == nil {
		return true, time.Time{}
	}
	nextAllowed := d.LastVerifiedAt.Add(VerifyCooldown)
	if time.Now().After(nextAllowed) {
		return true, time.Time{}
	}
	return false, nextAllowed
}

// isStale checks if the domain verification data is outdated.
func (s *DomainService) isStale(d *domain.SendingDomain) bool {
	if d.LastVerifiedAt == nil {
		return true
	}
	return time.Now().After(d.LastVerifiedAt.Add(StaleThreshold))
}

// Add registers a new sending domain with AWS SES.
func (s *DomainService) Add(ctx context.Context, userID, domainName string) (*domain.SendingDomain, *domain.DomainRecords, error) {
	// Validate domain name
	domainName = strings.TrimSpace(strings.ToLower(domainName))
	if domainName == "" {
		return nil, nil, fmt.Errorf("domain name is required")
	}

	// Check if domain already exists
	existing, _ := s.store.Domains().GetByDomainName(ctx, userID, domainName)
	if existing != nil {
		return nil, nil, fmt.Errorf("domain %s already exists", domainName)
	}

	// Create identity in SES
	result, err := s.ses.CreateEmailIdentity(ctx, domainName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create email identity: %w", err)
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
		Region:             s.region,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Store domain
	if err := s.store.Domains().Create(ctx, d); err != nil {
		// Try to clean up the SES identity if DB storage fails
		_ = s.ses.DeleteEmailIdentity(ctx, domainName)
		return nil, nil, fmt.Errorf("failed to store domain: %w", err)
	}

	// Build DNS records
	records := s.buildDomainRecords(d)

	return d, records, nil
}

// Get retrieves a domain by ID with authorization check.
func (s *DomainService) Get(ctx context.Context, userID, domainID string) (*domain.SendingDomain, error) {
	d, err := s.store.Domains().GetByID(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}

	if d.UserID != userID {
		return nil, fmt.Errorf("domain not found")
	}

	return d, nil
}

// List retrieves all domains for a user.
func (s *DomainService) List(ctx context.Context, userID string) ([]*domain.SendingDomain, error) {
	return s.store.Domains().GetByUserID(ctx, userID)
}

// GetVerifiedDomainForSending retrieves a domain by name for sender validation.
// Returns the domain if it exists and belongs to the user, nil otherwise.
func (s *DomainService) GetVerifiedDomainForSending(ctx context.Context, userID, domainName string) (*domain.SendingDomain, error) {
	d, err := s.store.Domains().GetByDomainName(ctx, userID, domainName)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}
	return d, nil
}

// Delete removes a domain from the account and SES.
func (s *DomainService) Delete(ctx context.Context, userID, domainID string) error {
	// Get domain with authorization check
	d, err := s.Get(ctx, userID, domainID)
	if err != nil {
		return err
	}

	// Delete from SES
	if err := s.ses.DeleteEmailIdentity(ctx, d.Domain); err != nil {
		return fmt.Errorf("failed to delete email identity: %w", err)
	}

	// Delete from database
	if err := s.store.Domains().Delete(ctx, domainID); err != nil {
		return fmt.Errorf("failed to delete domain: %w", err)
	}

	return nil
}

// Verify refreshes the verification status from AWS SES with rate limiting.
// Returns a VerifyResult indicating whether the refresh actually occurred or was rate-limited.
func (s *DomainService) Verify(ctx context.Context, userID, domainID string) (*domain.VerifyResult, error) {
	// Get domain with authorization check
	d, err := s.Get(ctx, userID, domainID)
	if err != nil {
		return nil, err
	}

	// Check cooldown to prevent SES rate limiting
	canVerify, nextRetry := s.canVerify(d)
	if !canVerify {
		return &domain.VerifyResult{
			Domain:       d,
			WasRefreshed: false,
			NextRetryAt:  &nextRetry,
			Message:      fmt.Sprintf("Rate limited. Next verification allowed at %s", nextRetry.Format(time.RFC3339)),
		}, nil
	}

	// Get current status from SES
	result, err := s.ses.GetEmailIdentity(ctx, d.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get email identity: %w", err)
	}

	// Update domain status from SES response
	now := time.Now()
	d.VerifiedForSending = result.VerifiedForSendingStatus
	d.DkimStatus = toDomainStatus(result.DkimStatus)
	if result.MailFromDomain != "" {
		d.MailFromDomain = result.MailFromDomain
		d.MailFromStatus = toDomainStatus(result.MailFromStatus)
	}

	// Determine overall status
	if result.VerifiedForSendingStatus {
		d.Status = domain.DomainStatusSuccess
	} else if result.DkimStatus == "FAILED" {
		d.Status = domain.DomainStatusFailed
	} else if result.DkimStatus == "TEMPORARY_FAILURE" {
		d.Status = domain.DomainStatusTemporaryFailure
	} else {
		d.Status = domain.DomainStatusPending
	}

	d.UpdatedAt = now
	d.LastVerifiedAt = &now

	// Save updated status
	if err := s.store.Domains().Update(ctx, d); err != nil {
		return nil, fmt.Errorf("failed to update domain: %w", err)
	}

	return &domain.VerifyResult{
		Domain:       d,
		WasRefreshed: true,
		Message:      "Verification status refreshed from SES",
	}, nil
}

// GetRecords returns all DNS records needed for the domain.
func (s *DomainService) GetRecords(ctx context.Context, userID, domainID string) (*domain.DomainRecords, error) {
	// Get domain with authorization check
	d, err := s.Get(ctx, userID, domainID)
	if err != nil {
		return nil, err
	}

	return s.buildDomainRecords(d), nil
}

// GetWithRecords retrieves a domain with its DNS records in a single call.
// If autoRefresh is true, it will automatically refresh stale verification data from SES.
func (s *DomainService) GetWithRecords(ctx context.Context, userID, domainID string, autoRefresh bool) (*domain.DomainWithRecords, error) {
	d, err := s.Get(ctx, userID, domainID)
	if err != nil {
		return nil, err
	}

	// Auto-refresh if stale and requested
	if autoRefresh && s.isStale(d) {
		if canVerify, _ := s.canVerify(d); canVerify {
			result, err := s.Verify(ctx, userID, domainID)
			if err == nil && result.WasRefreshed {
				d = result.Domain
			}
		}
	}

	return &domain.DomainWithRecords{
		Domain:  d,
		Records: s.buildDomainRecords(d),
	}, nil
}

// SetMailFrom configures a custom MAIL FROM subdomain.
func (s *DomainService) SetMailFrom(ctx context.Context, userID, domainID, mailFromSubdomain string) (*domain.SendingDomain, error) {
	// Get domain with authorization check
	d, err := s.Get(ctx, userID, domainID)
	if err != nil {
		return nil, err
	}

	// Build full MAIL FROM domain
	mailFromDomain := mailFromSubdomain + "." + d.Domain

	// Configure in SES
	if err := s.ses.PutEmailIdentityMailFromAttributes(ctx, d.Domain, mailFromDomain); err != nil {
		return nil, fmt.Errorf("failed to set mail from attributes: %w", err)
	}

	// Update domain
	d.MailFromDomain = mailFromDomain
	d.MailFromStatus = domain.DomainStatusPending
	d.UpdatedAt = time.Now()

	if err := s.store.Domains().Update(ctx, d); err != nil {
		return nil, fmt.Errorf("failed to update domain: %w", err)
	}

	return d, nil
}

// buildDomainRecords creates the full DNS records structure.
func (s *DomainService) buildDomainRecords(d *domain.SendingDomain) *domain.DomainRecords {
	records := &domain.DomainRecords{
		Domain:      d.Domain,
		DkimRecords: s.buildDkimRecords(d),
		SpfRecord:   s.buildSpfRecord(d),
		DmarcRecord: s.buildDmarcRecord(d),
		MxRecords:   s.buildMxInboundRecords(d),
	}

	// Add MAIL FROM records if configured
	if d.MailFromDomain != "" {
		records.MailFromRecords = s.buildMailFromRecords(d)
	}

	// Determine status flags
	records.IsReadyToSend = d.VerifiedForSending
	records.IsReadyToReceive = len(records.MxRecords) > 0
	records.IsFullyConfigured = records.IsReadyToSend &&
		d.DkimStatus == domain.DomainStatusSuccess &&
		(d.MailFromDomain == "" || d.MailFromStatus == domain.DomainStatusSuccess)

	return records
}

// buildDkimRecords creates DKIM CNAME records from tokens.
func (s *DomainService) buildDkimRecords(d *domain.SendingDomain) []domain.DnsRecord {
	if len(d.DkimTokens) == 0 {
		return nil
	}

	records := make([]domain.DnsRecord, len(d.DkimTokens))
	status := toRecordStatus(d.DkimStatus)

	for i, token := range d.DkimTokens {
		records[i] = domain.DnsRecord{
			DnsType:      "CNAME",
			Name:         fmt.Sprintf("%s._domainkey.%s", token, d.Domain),
			Value:        fmt.Sprintf("%s.dkim.amazonses.com", token),
			RecordType:   domain.RecordTypeDKIM,
			Status:       status,
			Instructions: "Add this CNAME record to enable DKIM email signing.",
		}
	}

	return records
}

// buildSpfRecord creates the SPF TXT record.
func (s *DomainService) buildSpfRecord(d *domain.SendingDomain) *domain.DnsRecord {
	return &domain.DnsRecord{
		DnsType:      "TXT",
		Name:         d.Domain,
		Value:        "v=spf1 include:amazonses.com ~all",
		RecordType:   domain.RecordTypeSPF,
		Status:       domain.RecordStatusPending,
		Instructions: "Add this TXT record to authorize Amazon SES to send emails on your behalf.",
	}
}

// buildDmarcRecord creates the recommended DMARC TXT record.
func (s *DomainService) buildDmarcRecord(d *domain.SendingDomain) *domain.DnsRecord {
	return &domain.DnsRecord{
		DnsType:      "TXT",
		Name:         fmt.Sprintf("_dmarc.%s", d.Domain),
		Value:        fmt.Sprintf("v=DMARC1; p=none; rua=mailto:dmarc@%s", d.Domain),
		RecordType:   domain.RecordTypeDMARC,
		Status:       domain.RecordStatusPending,
		Instructions: "Add this TXT record to enable DMARC policy. Start with p=none for monitoring, then gradually increase to p=quarantine or p=reject.",
	}
}

// buildMxInboundRecords creates MX records for inbound email.
func (s *DomainService) buildMxInboundRecords(d *domain.SendingDomain) []domain.DnsRecord {
	return []domain.DnsRecord{
		{
			DnsType:      "MX",
			Name:         d.Domain,
			Value:        fmt.Sprintf("inbound-smtp.%s.amazonaws.com", s.region),
			Priority:     10,
			RecordType:   domain.RecordTypeMXInbound,
			Status:       domain.RecordStatusPending,
			Instructions: "Add this MX record to receive inbound emails via Amazon SES.",
		},
	}
}

// buildMailFromRecords creates MX and SPF records for custom MAIL FROM.
func (s *DomainService) buildMailFromRecords(d *domain.SendingDomain) []domain.DnsRecord {
	status := toRecordStatus(d.MailFromStatus)

	return []domain.DnsRecord{
		{
			DnsType:      "MX",
			Name:         d.MailFromDomain,
			Value:        fmt.Sprintf("feedback-smtp.%s.amazonses.com", s.region),
			Priority:     10,
			RecordType:   domain.RecordTypeMailFromMX,
			Status:       status,
			Instructions: "Add this MX record to your MAIL FROM subdomain for bounce handling.",
		},
		{
			DnsType:      "TXT",
			Name:         d.MailFromDomain,
			Value:        "v=spf1 include:amazonses.com ~all",
			RecordType:   domain.RecordTypeMailFromSPF,
			Status:       status,
			Instructions: "Add this SPF record to your MAIL FROM subdomain.",
		},
	}
}

// toDomainStatus converts SES status string to DomainStatus.
func toDomainStatus(status string) domain.DomainStatus {
	switch strings.ToUpper(status) {
	case "SUCCESS":
		return domain.DomainStatusSuccess
	case "FAILED":
		return domain.DomainStatusFailed
	case "TEMPORARY_FAILURE":
		return domain.DomainStatusTemporaryFailure
	default:
		return domain.DomainStatusPending
	}
}

// toRecordStatus converts DomainStatus to RecordStatus.
func toRecordStatus(status domain.DomainStatus) domain.RecordStatus {
	switch status {
	case domain.DomainStatusSuccess:
		return domain.RecordStatusVerified
	case domain.DomainStatusFailed:
		return domain.RecordStatusFailed
	default:
		return domain.RecordStatusPending
	}
}
