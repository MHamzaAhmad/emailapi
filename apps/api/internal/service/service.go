package service

import (
	"github.com/emailapi/api/internal/eventstream"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	"github.com/emailapi/api/internal/external/svix"
	"github.com/emailapi/api/internal/external/webrisk"
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
	svc.APIKey = NewAPIKeyService(store, nil, nil)    // Cache and Activity set via NewWithDeps
	svc.User = NewUserService(store, svc.APIKey, nil) // Cache set via NewWithDeps
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
	MXCache     *rediscache.MXCache
	// Event streaming
	EventConsumer eventstream.Consumer
	// Clerk configuration
	ClerkWebhookSecret string
	// Web Risk client for URL safety validation
	WebRiskClient webrisk.Client
	RedisClient   *rediscache.Client
}

// NewWithDeps creates a new Service with all dependencies.
func NewWithDeps(deps ServiceDeps) *Service {
	svc := New(deps.Store)
	svc.Domain = NewDomainService(deps.Store, deps.SESClient, deps.DomainCache, deps.CHActivityRepo, deps.Region, deps.SESConfigurationSet)
	svc.APIKey = NewAPIKeyService(deps.Store, deps.APIKeyCache, deps.CHActivityRepo)
	svc.User = NewUserService(deps.Store, svc.APIKey, deps.UserCache)

	// Create body validator for URL safety checking (optional if Web Risk not configured)
	var bodyValidator *validation.BodyValidator
	if deps.WebRiskClient != nil && deps.RedisClient != nil {
		bodyValidator = validation.NewBodyValidator(deps.RedisClient, deps.WebRiskClient)
	}

	// Create email validator with domain checker, suppression checker, body validator, and MX cache
	emailValidator := validation.NewEmailValidator(svc.Domain, deps.SuppressionRepo, bodyValidator, deps.MXCache)

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
