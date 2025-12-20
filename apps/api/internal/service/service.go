package service

import (
	"github.com/emailapi/api/internal/external/ses"
)

// Service aggregates all business logic services.
// This is injected into the Transport layer.
type Service struct {
	store Store

	User   *UserService
	APIKey *APIKeyService
	Domain *DomainService
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
func NewWithSES(store Store, sesClient ses.Client, region string) *Service {
	svc := New(store)
	svc.Domain = NewDomainService(store, sesClient, region)
	return svc
}
