package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/emailapiv1connect"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
)

// DomainHandler implements the Connect DomainServiceHandler.
type DomainHandler struct {
	emailapiv1connect.UnimplementedDomainServiceHandler
	svc *service.DomainService
}

// NewDomainHandler creates a new DomainHandler.
func NewDomainHandler(svc *service.DomainService) *DomainHandler {
	return &DomainHandler{svc: svc}
}

// AddDomain handles adding a new domain.
func (h *DomainHandler) AddDomain(
	ctx context.Context,
	req *connect.Request[emailapiv1.AddDomainRequest],
) (*connect.Response[emailapiv1.AddDomainResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	details, err := h.svc.Add(ctx, userID, req.Msg.Domain)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emailapiv1.AddDomainResponse{
		Domain:  toProtoDomain(details),
		Message: "Domain added. Configure the DNS records below to start sending emails.",
	}), nil
}

// GetDomain handles retrieving a domain.
func (h *DomainHandler) GetDomain(
	ctx context.Context,
	req *connect.Request[emailapiv1.GetDomainRequest],
) (*connect.Response[emailapiv1.GetDomainResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	details, err := h.svc.Get(ctx, userID, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("domain not found or access denied"))
	}

	return connect.NewResponse(&emailapiv1.GetDomainResponse{
		Domain: toProtoDomain(details),
	}), nil
}

// ListDomains handles listing all domains for a user.
func (h *DomainHandler) ListDomains(
	ctx context.Context,
	req *connect.Request[emailapiv1.ListDomainsRequest],
) (*connect.Response[emailapiv1.ListDomainsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	domains, err := h.svc.List(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoDomains := make([]*emailapiv1.Domain, len(domains))
	for i, d := range domains {
		protoDomains[i] = toProtoDomain(d)
	}

	return connect.NewResponse(&emailapiv1.ListDomainsResponse{
		Data: protoDomains,
	}), nil
}

// DeleteDomain handles deleting a domain.
func (h *DomainHandler) DeleteDomain(
	ctx context.Context,
	req *connect.Request[emailapiv1.DeleteDomainRequest],
) (*connect.Response[emailapiv1.DeleteDomainResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	if err := h.svc.Delete(ctx, userID, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emailapiv1.DeleteDomainResponse{
		Message: "Domain deleted.",
	}), nil
}

// VerifyDomain handles verifying a domain status with rate limiting.
func (h *DomainHandler) VerifyDomain(
	ctx context.Context,
	req *connect.Request[emailapiv1.VerifyDomainRequest],
) (*connect.Response[emailapiv1.VerifyDomainResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	result, err := h.svc.Verify(ctx, userID, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &emailapiv1.VerifyDomainResponse{
		Domain:       toProtoDomain(result.Domain),
		WasRefreshed: result.WasRefreshed,
		Message:      result.Message,
	}
	if result.NextRetryAt != nil {
		resp.NextRetryAt = timestamppb.New(*result.NextRetryAt)
	}
	return connect.NewResponse(resp), nil
}

// ============================================================================
// Proto Conversion Helpers
// ============================================================================

func toProtoDomain(d *domain.DomainWithDetails) *emailapiv1.Domain {
	pb := &emailapiv1.Domain{
		Id:        d.ID,
		Domain:    d.Domain,
		Status:    toProtoDomainStatus(d.Status),
		Region:    d.Region,
		CreatedAt: timestamppb.New(d.CreatedAt),
		UpdatedAt: timestamppb.New(d.UpdatedAt),
		Summary:   toProtoSummary(d.Summary),
		Records:   toProtoRecords(d.Records),
	}
	if d.LastCheckedAt != nil {
		pb.LastCheckedAt = timestamppb.New(*d.LastCheckedAt)
	}
	return pb
}

func toProtoDomainStatus(s domain.DomainStatus) emailapiv1.DomainStatus {
	switch s {
	case domain.DomainStatusPending:
		return emailapiv1.DomainStatus_DOMAIN_STATUS_PENDING
	case domain.DomainStatusVerifying:
		return emailapiv1.DomainStatus_DOMAIN_STATUS_VERIFYING
	case domain.DomainStatusReady:
		return emailapiv1.DomainStatus_DOMAIN_STATUS_READY
	case domain.DomainStatusDegraded:
		return emailapiv1.DomainStatus_DOMAIN_STATUS_DEGRADED
	case domain.DomainStatusFailed:
		return emailapiv1.DomainStatus_DOMAIN_STATUS_FAILED
	default:
		return emailapiv1.DomainStatus_DOMAIN_STATUS_UNSPECIFIED
	}
}

func toProtoSummary(s *domain.DomainSummary) *emailapiv1.DomainSummary {
	if s == nil {
		return nil
	}
	return &emailapiv1.DomainSummary{
		Message:           s.Message,
		NextAction:        s.NextAction,
		RecordsPending:    int32(s.RecordsPending),
		RecordsConfigured: int32(s.RecordsConfigured),
		CanSend:           s.CanSend,
		CanReceive:        s.CanReceive,
	}
}

func toProtoRecords(r *domain.DomainRecords) *emailapiv1.DomainRecords {
	if r == nil {
		return nil
	}
	pb := &emailapiv1.DomainRecords{}

	for _, rec := range r.DkimRecords {
		pb.DkimRecords = append(pb.DkimRecords, toProtoRecord(&rec))
	}

	if r.SpfRecord != nil {
		pb.SpfRecord = toProtoRecord(r.SpfRecord)
	}

	if r.DmarcRecord != nil {
		pb.DmarcRecord = toProtoRecord(r.DmarcRecord)
	}

	for _, rec := range r.MxRecords {
		pb.MxRecords = append(pb.MxRecords, toProtoRecord(&rec))
	}

	for _, rec := range r.MailFromRecords {
		pb.MailFromRecords = append(pb.MailFromRecords, toProtoRecord(&rec))
	}

	return pb
}

func toProtoRecord(r *domain.DnsRecord) *emailapiv1.DnsRecord {
	return &emailapiv1.DnsRecord{
		Type:            r.Type,
		Name:            r.Name,
		Value:           r.Value,
		Priority:        int32(r.Priority),
		RecordType:      toProtoRecordType(r.RecordType),
		Status:          toProtoRecordStatus(r.Status),
		NameShort:       r.NameShort,
		DiscoveredValue: r.DiscoveredValue,
		Instructions:    r.Instructions,
	}
}

func toProtoRecordType(t domain.RecordType) emailapiv1.RecordType {
	switch t {
	case domain.RecordTypeDKIM:
		return emailapiv1.RecordType_RECORD_TYPE_DKIM
	case domain.RecordTypeSPF:
		return emailapiv1.RecordType_RECORD_TYPE_SPF
	case domain.RecordTypeDMARC:
		return emailapiv1.RecordType_RECORD_TYPE_DMARC
	case domain.RecordTypeMXInbound:
		return emailapiv1.RecordType_RECORD_TYPE_MX_INBOUND
	case domain.RecordTypeMailFromMX:
		return emailapiv1.RecordType_RECORD_TYPE_MAIL_FROM_MX
	case domain.RecordTypeMailFromSPF:
		return emailapiv1.RecordType_RECORD_TYPE_MAIL_FROM_SPF
	default:
		return emailapiv1.RecordType_RECORD_TYPE_UNSPECIFIED
	}
}

func toProtoRecordStatus(s domain.RecordStatus) emailapiv1.RecordStatus {
	switch s {
	case domain.RecordStatusPending:
		return emailapiv1.RecordStatus_RECORD_STATUS_PENDING
	case domain.RecordStatusFound:
		return emailapiv1.RecordStatus_RECORD_STATUS_FOUND
	case domain.RecordStatusMismatch:
		return emailapiv1.RecordStatus_RECORD_STATUS_MISMATCH
	case domain.RecordStatusMissing:
		return emailapiv1.RecordStatus_RECORD_STATUS_MISSING
	default:
		return emailapiv1.RecordStatus_RECORD_STATUS_UNSPECIFIED
	}
}
