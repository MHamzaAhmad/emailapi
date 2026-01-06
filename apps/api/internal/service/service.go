package service

import (
	internaldns "github.com/emailapi/api/internal/dns"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/eventstream"
	"github.com/emailapi/api/internal/external/polar"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	"github.com/emailapi/api/internal/external/sqs"
	"github.com/emailapi/api/internal/external/svix"
	"github.com/emailapi/api/internal/external/webrisk"
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

	User         *UserService
	APIKey       *APIKeyService
	Domain       *DomainService
	Email        *EmailService
	Internal     *InternalService
	Webhook      *WebhookService
	Activity     *ActivityService
	InboundEmail *InboundEmailService
	SQSEvent     *SQSEventService // New: SQS-based event processing
	Reputation   *ReputationService
	Unsubscribe  *UnsubscribeService
	Admin        *AdminService
	Billing      *BillingService
}

// New creates a new Service with the given Store.
// Note: DomainService requires SES client and must be set separately using SetDomainService.
func New(store Store) *Service {
	svc := &Service{store: store}
	svc.APIKey = NewAPIKeyService(store, nil, nil, "") // Cache, Analytics, and HMAC secret set via NewWithDeps
	svc.User = NewUserService(store, svc.APIKey, nil)  // Cache set via NewWithDeps
	return svc
}

// ServiceDeps holds dependencies for service initialization.
type ServiceDeps struct {
	Store               Store
	Cache               Cache     // Aggregated cache access
	Analytics           Analytics // Aggregated Tinybird access
	SESClient           ses.Client
	S3Factory           *s3.Factory
	SQSClient           sqs.Client // New: SQS client for event processing
	Region              string
	SESConfigurationSet string
	RiverClient         *river.Client[pgx.Tx]
	SvixClient          svix.Client // Used for WebhookService (portal access)
	WebhookSender       webhook.Sender
	SuppressionRepo     *suppression.Repository
	// Event streaming
	EventConsumer eventstream.Consumer
	// Clerk configuration
	ClerkWebhookSecret string
	// Polar Webhook Secret
	PolarWebhookSecret string
	// API Key Configuration
	APIKeyHMACSecret string
	// Web Risk client for URL safety validation
	WebRiskClient webrisk.Client
	RedisClient   *rediscache.Client
	// Unsubscribe configuration
	UnsubscribeBaseURL     string
	UnsubscribeTokenSecret string
	// SQS event processing
	InboundBucket string

	// Polar Client and Product IDs
	PolarClient           polar.Client
	PolarFreeProductID    string // Product ID for Free plan
	PolarStarterProductID string // Product ID for Starter plan
	PolarGrowthProductID  string // Product ID for Growth plan
}

// NewWithDeps creates a new Service with all dependencies.
func NewWithDeps(deps ServiceDeps) *Service {
	svc := New(deps.Store)

	// Initialize Reputation service first (needed by DomainService and EmailValidator)
	svc.Reputation = NewReputationService(deps.Store, deps.RiverClient, deps.Cache, deps.Analytics)

	// Initialize Domain service with reputation checker
	svc.Domain = NewDomainService(deps.Store, deps.SESClient, internaldns.NewValidator(), deps.Cache, deps.Analytics, svc.Reputation, deps.Region, deps.SESConfigurationSet)
	svc.APIKey = NewAPIKeyService(deps.Store, deps.Cache, deps.Analytics, deps.APIKeyHMACSecret)
	svc.User = NewUserService(deps.Store, svc.APIKey, deps.Cache)

	// Create body validator for URL safety checking (optional if Web Risk not configured)
	var bodyValidator *validation.BodyValidator
	if deps.WebRiskClient != nil && deps.RedisClient != nil {
		bodyValidator = validation.NewBodyValidator(deps.RedisClient, deps.WebRiskClient)
	}

	// Create email validator with domain checker, suppression checker, body validator, MX cache, and reputation checker
	var mxCache rediscache.MXCacheInterface
	if deps.Cache != nil {
		mxCache = deps.Cache.MX()
	}
	emailValidator := validation.NewEmailValidator(svc.Domain, deps.SuppressionRepo, bodyValidator, mxCache, svc.Reputation)

	// Initialize Unsubscribe service
	var unsubscribeSvc *UnsubscribeService
	if deps.UnsubscribeTokenSecret != "" {
		tokenSvc := NewUnsubscribeTokenService(deps.UnsubscribeTokenSecret)
		unsubscribeSvc = NewUnsubscribeService(
			deps.Store.Unsubscribe(),
			deps.Cache,
			tokenSvc,
			deps.UnsubscribeBaseURL,
		)
		svc.Unsubscribe = unsubscribeSvc
	}

	svc.Email = NewEmailService(deps.RiverClient, deps.Analytics, emailValidator, deps.SESClient, deps.WebhookSender, deps.EventConsumer, svc.Reputation, unsubscribeSvc)
	// Initialize Billing service
	var creditCache rediscache.CreditCacheInterface
	var usageCache rediscache.UsageCacheInterface
	if deps.Cache != nil {
		creditCache = deps.Cache.Credit()
		usageCache = deps.Cache.Usage()
	}
	svc.Billing = NewBillingService(deps.PolarClient, creditCache, usageCache, deps.Store)

	svc.Internal = NewInternalService(InternalServiceConfig{
		UserService:        svc.User,
		BillingService:     svc.Billing,
		Store:              deps.Store,
		ClerkWebhookSecret: deps.ClerkWebhookSecret,
		PolarWebhookSecret: deps.PolarWebhookSecret,
		PolarClient:        deps.PolarClient,
		FreeProductID:      deps.PolarFreeProductID,
	})

	// Initialize Activity service
	svc.Activity = NewActivityService(deps.Analytics)

	// Initialize Svix-dependent services if client is available
	if deps.SvixClient != nil {
		svc.Webhook = NewWebhookService(deps.SvixClient)
	}
	if deps.WebhookSender != nil {
		svc.InboundEmail = NewInboundEmailService(
			deps.S3Factory,
			deps.Analytics,
			deps.WebhookSender,
		)
	}

	// Initialize SQS event service if SQS client is available
	if deps.SQSClient != nil {
		// Get pending attachment cache for GuardDuty handler
		var pendingAttachmentCache rediscache.PendingAttachmentCacheInterface
		if deps.Cache != nil {
			pendingAttachmentCache = deps.Cache.PendingAttachment()
		}

		svc.SQSEvent = NewSQSEventService(SQSEventServiceDeps{
			SQSClient:              deps.SQSClient,
			Analytics:              deps.Analytics,
			WebhookSender:          deps.WebhookSender,
			SuppressionRepo:        deps.SuppressionRepo,
			ReputationSvc:          svc.Reputation,
			S3Factory:              deps.S3Factory,
			InboundProcessor:       svc.InboundEmail,   // Wire inbound processor
			InboundBucket:          deps.InboundBucket, // Should probably be configurable or discovered
			PendingAttachmentCache: pendingAttachmentCache,
			RiverClient:            deps.RiverClient,
		})
	}

	// Initialize Admin service for admin operations
	svc.Admin = NewAdminService(deps.Store, svc.User, svc.Reputation)

	// Initialize product mappings from config
	domain.InitProductMappings(deps.PolarStarterProductID, deps.PolarGrowthProductID)

	// Initialize Billing service

	return svc
}
