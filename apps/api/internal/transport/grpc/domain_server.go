package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/service"
)

// DomainServer implements the DomainService gRPC server.
type DomainServer struct {
	emailapiv1.UnimplementedDomainServiceServer
	svc *service.DomainService
}

// NewDomainServer creates a new DomainServer.
func NewDomainServer(svc *service.DomainService) *DomainServer {
	return &DomainServer{svc: svc}
}

// AddDomain handles adding a new domain.
func (s *DomainServer) AddDomain(ctx context.Context, req *emailapiv1.AddDomainRequest) (*emailapiv1.AddDomainResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	d, records, err := s.svc.Add(ctx, userID, req.Domain)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add domain: %v", err)
	}

	return &emailapiv1.AddDomainResponse{
		Domain:  toProtoDomain(d),
		Records: toProtoRecords(records),
		Message: "Domain added successfully. Please configure the returned DNS records to verify your domain.",
	}, nil
}

// GetDomain handles retrieving a domain.
func (s *DomainServer) GetDomain(ctx context.Context, req *emailapiv1.GetDomainRequest) (*emailapiv1.Domain, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	d, err := s.svc.Get(ctx, userID, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "domain not found or access denied")
	}

	return toProtoDomain(d), nil
}

// ListDomains handles listing all domains for a user.
func (s *DomainServer) ListDomains(ctx context.Context, req *emailapiv1.ListDomainsRequest) (*emailapiv1.ListDomainsResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	domains, err := s.svc.List(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list domains: %v", err)
	}

	protoDomains := make([]*emailapiv1.Domain, len(domains))
	for i, d := range domains {
		protoDomains[i] = toProtoDomain(d)
	}

	return &emailapiv1.ListDomainsResponse{
		Data: protoDomains,
	}, nil
}

// DeleteDomain handles deleting a domain.
func (s *DomainServer) DeleteDomain(ctx context.Context, req *emailapiv1.DeleteDomainRequest) (*emailapiv1.DeleteDomainResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if err := s.svc.Delete(ctx, userID, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete domain: %v", err)
	}

	return &emailapiv1.DeleteDomainResponse{
		Message: "Domain deleted successfully.",
	}, nil
}

// VerifyDomain handles verifying a domain status with rate limiting.
func (s *DomainServer) VerifyDomain(ctx context.Context, req *emailapiv1.VerifyDomainRequest) (*emailapiv1.VerifyDomainResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	result, err := s.svc.Verify(ctx, userID, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify domain: %v", err)
	}

	resp := &emailapiv1.VerifyDomainResponse{
		Domain:       toProtoDomain(result.Domain),
		WasRefreshed: result.WasRefreshed,
		Message:      result.Message,
	}
	if result.NextRetryAt != nil {
		resp.NextRetryAt = timestamppb.New(*result.NextRetryAt)
	}
	return resp, nil
}

// GetDomainRecords handles retrieving DNS records for a domain.
func (s *DomainServer) GetDomainRecords(ctx context.Context, req *emailapiv1.GetDomainRecordsRequest) (*emailapiv1.DomainRecords, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	records, err := s.svc.GetRecords(ctx, userID, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get domain records: %v", err)
	}

	return toProtoRecords(records), nil
}

// SetMailFromDomain handles configuring custom MAIL FROM domain.
func (s *DomainServer) SetMailFromDomain(ctx context.Context, req *emailapiv1.SetMailFromDomainRequest) (*emailapiv1.Domain, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	d, err := s.svc.SetMailFrom(ctx, userID, req.Id, req.MailFromSubdomain)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set mail from domain: %v", err)
	}

	return toProtoDomain(d), nil
}

// Helper functions for conversions

func toProtoDomain(d *domain.SendingDomain) *emailapiv1.Domain {
	pb := &emailapiv1.Domain{
		Id:                 d.ID,
		Domain:             d.Domain,
		Status:             toProtoDomainStatus(d.Status),
		VerifiedForSending: d.VerifiedForSending,
		MailFromDomain:     d.MailFromDomain,
		MailFromStatus:     toProtoDomainStatus(d.MailFromStatus),
		Region:             d.Region,
		CreatedAt:          timestamppb.New(d.CreatedAt),
		UpdatedAt:          timestamppb.New(d.UpdatedAt),
	}
	if d.LastVerifiedAt != nil {
		pb.LastVerifiedAt = timestamppb.New(*d.LastVerifiedAt)
	}
	return pb
}

func toProtoDomainStatus(s domain.DomainStatus) emailapiv1.DomainStatus {
	switch s {
	case domain.DomainStatusPending:
		return emailapiv1.DomainStatus_DOMAIN_STATUS_PENDING
	case domain.DomainStatusSuccess:
		return emailapiv1.DomainStatus_DOMAIN_STATUS_SUCCESS
	case domain.DomainStatusFailed:
		return emailapiv1.DomainStatus_DOMAIN_STATUS_FAILED
	case domain.DomainStatusTemporaryFailure:
		return emailapiv1.DomainStatus_DOMAIN_STATUS_TEMPORARY_FAILURE
	default:
		return emailapiv1.DomainStatus_DOMAIN_STATUS_UNSPECIFIED
	}
}

func toProtoRecords(r *domain.DomainRecords) *emailapiv1.DomainRecords {
	pb := &emailapiv1.DomainRecords{
		Domain:            r.Domain,
		IsReadyToSend:     r.IsReadyToSend,
		IsReadyToReceive:  r.IsReadyToReceive,
		IsFullyConfigured: r.IsFullyConfigured,
	}

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
		DnsType:      r.DnsType,
		Name:         r.Name,
		Value:        r.Value,
		Priority:     int32(r.Priority),
		RecordType:   toProtoRecordType(r.RecordType),
		Status:       toProtoRecordStatus(r.Status),
		Instructions: r.Instructions,
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
	case domain.RecordStatusVerified:
		return emailapiv1.RecordStatus_RECORD_STATUS_VERIFIED
	case domain.RecordStatusFailed:
		return emailapiv1.RecordStatus_RECORD_STATUS_FAILED
	default:
		return emailapiv1.RecordStatus_RECORD_STATUS_UNSPECIFIED
	}
}
