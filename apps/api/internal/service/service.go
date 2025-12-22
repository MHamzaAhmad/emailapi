package service

import (
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	"github.com/emailapi/api/internal/external/svix"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	pgrepo "github.com/emailapi/api/internal/repository/postgres"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/emailapi/api/internal/validation"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// Service aggregates all business logic services.
// This is injected into the Transport layer.
type Service struct {
	store Store

	User            *UserService
	APIKey          *APIKeyService
	Domain          *DomainService
	Email           *EmailService
	Internal        *InternalService
	Webhook         *WebhookService
	InboundEmail    *InboundEmailService
	SNSNotification *SNSNotificationService
}

// New creates a new Service with the given Store.
// Note: DomainService requires SES client and must be set separately using SetDomainService.
func New(store Store) *Service {
	svc := &Service{store: store}
	svc.APIKey = NewAPIKeyService(store)
	svc.User = NewUserService(store, svc.APIKey)
	return svc
}

// ServiceDeps holds dependencies for service initialization.
type ServiceDeps struct {
	Store               Store
	SESClient           ses.Client
	S3Factory           *s3.Factory
	Region              string
	SESConfigurationSet string // SES configuration set for email event notifications
	RiverClient         *river.Client[pgx.Tx]
	PGEmailRepo         *pgrepo.EmailRepository
	CHEmailRepo         *chrepo.EmailRepository
	CHActivityRepo      *chrepo.ActivityRepository
	SvixClient          svix.Client
	SuppressionRepo     *suppression.Repository
}

// NewWithDeps creates a new Service with all dependencies.
func NewWithDeps(deps ServiceDeps) *Service {
	svc := New(deps.Store)
	svc.Domain = NewDomainService(deps.Store, deps.SESClient, deps.Region, deps.SESConfigurationSet)

	// Create email validator with domain checker and suppression checker
	emailValidator := validation.NewEmailValidator(svc.Domain, deps.SuppressionRepo)

	svc.Email = NewEmailService(deps.RiverClient, deps.PGEmailRepo, deps.CHEmailRepo, emailValidator, deps.SESClient)
	svc.Internal = NewInternalService(deps.PGEmailRepo, deps.RiverClient)

	// Initialize SNS notification service
	svc.SNSNotification = NewSNSNotificationService(
		deps.PGEmailRepo,
		deps.CHEmailRepo,
		deps.CHActivityRepo,
		deps.SvixClient,
		deps.S3Factory,
		deps.SuppressionRepo,
	)

	// Initialize Svix-dependent services if client is available
	if deps.SvixClient != nil {
		svc.Webhook = NewWebhookService(deps.SvixClient)
		svc.InboundEmail = NewInboundEmailService(
			deps.S3Factory,
			deps.CHEmailRepo,
			deps.SvixClient,
		)
	}

	return svc
}
