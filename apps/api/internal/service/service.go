package service

import (
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	"github.com/emailapi/api/internal/repository/clickhouse"
	"github.com/riverqueue/river"
)

// Service aggregates all business logic services.
// This is injected into the Transport layer.
type Service struct {
	store Store

	User   *UserService
	APIKey *APIKeyService
	Domain *DomainService
	Email  *EmailService
}

// New creates a new Service with the given Store.
// Note: DomainService requires SES client and must be set separately using SetDomainService.
func New(store Store) *Service {
	svc := &Service{store: store}
	svc.APIKey = NewAPIKeyService(store)
	svc.User = NewUserService(store, svc.APIKey)
	return svc
}

// NewWithSES creates a new Service with the given Store and SES client.
// It also initializes EmailService.
func NewWithSES(
	store Store,
	sesClient ses.Client,
	region string,
	riverClient *river.Client[any],
	s3Client s3.Client,
	chRepo *clickhouse.EmailRepository,
) *Service {
	svc := New(store)
	svc.Domain = NewDomainService(store, sesClient, region)
	svc.Email = NewEmailService(riverClient, s3Client, chRepo)
	return svc
}
