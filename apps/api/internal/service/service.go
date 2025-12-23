package service

import (
	"github.com/emailapi/api/internal/eventstream"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	"github.com/emailapi/api/internal/external/svix"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	rediscache "github.com/emailapi/api/internal/repository/redis"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/emailapi/api/internal/validation"
	"github.com/emailapi/api/internal/webhook"

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
	Activity        *ActivityService
	InboundEmail    *InboundEmailService
	SNSNotification *SNSNotificationService
}

// New creates a new Service with the given Store.
// Note: DomainService requires SES client and must be set separately using SetDomainService.
func New(store Store) *Service {
	svc := &Service{store: store}
	svc.APIKey = NewAPIKeyService(store, nil) // Cache set via NewWithDeps
	svc.User = NewUserService(store, svc.APIKey)
	return svc
}

// ServiceDeps holds dependencies for service initialization.
type ServiceDeps struct {
	Store               Store
	SESClient           ses.Client
	S3Factory           *s3.Factory
	Region              string
	SESConfigurationSet string
	RiverClient         *river.Client[pgx.Tx]
	CHEmailRepo         *chrepo.EmailRepository
	CHActivityRepo      *chrepo.ActivityRepository
	SvixClient          svix.Client // Used for WebhookService (portal access)
	WebhookSender       webhook.Sender
	SuppressionRepo     *suppression.Repository
	// Cache repositories
	DomainCache *rediscache.DomainCache
	APIKeyCache *rediscache.APIKeyCache
	UserCache   *rediscache.UserCache
	// Event streaming
	EventConsumer eventstream.Consumer
	// Clerk configuration
	ClerkWebhookSecret string
}

// NewWithDeps creates a new Service with all dependencies.
func NewWithDeps(deps ServiceDeps) *Service {
	svc := New(deps.Store)
	svc.Domain = NewDomainService(deps.Store, deps.SESClient, deps.DomainCache, deps.Region, deps.SESConfigurationSet)
	svc.APIKey = NewAPIKeyService(deps.Store, deps.APIKeyCache)

	// Create email validator with domain checker and suppression checker
	emailValidator := validation.NewEmailValidator(svc.Domain, deps.SuppressionRepo)

	svc.Email = NewEmailService(deps.RiverClient, deps.CHEmailRepo, emailValidator, deps.SESClient, deps.WebhookSender, deps.EventConsumer)
	svc.Internal = NewInternalService(InternalServiceConfig{
		UserService:        svc.User,
		ClerkWebhookSecret: deps.ClerkWebhookSecret,
	})

	// Initialize SNS notification service
	svc.SNSNotification = NewSNSNotificationService(
		deps.CHEmailRepo,
		deps.CHActivityRepo,
		deps.WebhookSender,
		deps.SuppressionRepo,
	)

	// Initialize Activity service
	svc.Activity = NewActivityService(deps.CHActivityRepo)

	// Initialize Svix-dependent services if client is available
	if deps.SvixClient != nil {
		svc.Webhook = NewWebhookService(deps.SvixClient)
	}
	if deps.WebhookSender != nil {
		svc.InboundEmail = NewInboundEmailService(
			deps.S3Factory,
			deps.CHEmailRepo,
			deps.WebhookSender,
		)
	}

	return svc
}
