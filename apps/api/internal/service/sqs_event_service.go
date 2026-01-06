package service

import (
	"context"

	"github.com/emailapi/api/internal/events"
	"github.com/emailapi/api/internal/events/handlers"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/sqs"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
	"github.com/emailapi/api/internal/webhook"
)

// SQSEventService processes email events from SQS.
// This replaces the SNSNotificationService for event processing.
type SQSEventService struct {
	sqsClient sqs.Client
	router    *events.Router
}

// SQSEventServiceDeps contains dependencies for SQSEventService.
type SQSEventServiceDeps struct {
	SQSClient        sqs.Client
	Analytics        Analytics
	WebhookSender    webhook.Sender
	SuppressionRepo  SuppressionManager
	ReputationSvc    ReputationRecorder
	S3Factory        s3.FactoryInterface
	InboundProcessor InboundEmailProcessor
	InboundBucket    string // S3 bucket name for inbound emails
	// GuardDuty integration
	PendingAttachmentCache redisrepo.PendingAttachmentCacheInterface
	RiverClient            handlers.RiverClient
}

// InboundEmailProcessor processes raw inbound emails.
type InboundEmailProcessor interface {
	ProcessRawEmail(ctx context.Context, bucket, key string) error
}

// handlerAnalyticsAdapter adapts the service.Analytics interface to handlers.Analytics
type handlerAnalyticsAdapter struct {
	analytics Analytics
}

func (a *handlerAnalyticsAdapter) Activity() handlers.ActivityLogger {
	return a.analytics.Activity()
}

func (a *handlerAnalyticsAdapter) Email() handlers.EmailRouter {
	return a.analytics.Email()
}

// NewSQSEventService creates a new SQS event service.
func NewSQSEventService(deps SQSEventServiceDeps) *SQSEventService {
	router := events.NewRouter()

	// Create handler dependencies
	handlerDeps := &handlers.Dependencies{
		Analytics:     &handlerAnalyticsAdapter{analytics: deps.Analytics},
		WebhookSender: deps.WebhookSender,
		SuppressRepo:  deps.SuppressionRepo,
		ReputationSvc: deps.ReputationSvc,
	}

	// Register SES event handlers
	router.RegisterHandler(events.EventTypeBounce, handlers.NewBounceHandler(handlerDeps))
	router.RegisterHandler(events.EventTypeComplaint, handlers.NewComplaintHandler(handlerDeps))
	router.RegisterHandler(events.EventTypeDelivery, handlers.NewDeliveryHandler(handlerDeps))

	// Register inbound email handler if processor and bucket are configured
	if deps.InboundProcessor != nil && deps.InboundBucket != "" {
		inboundHandler := handlers.NewInboundEmailHandler(deps.S3Factory, deps.InboundProcessor)
		router.RegisterInboundHandler(inboundHandler, deps.InboundBucket)
	}

	// Register GuardDuty handler if dependencies are available
	if deps.PendingAttachmentCache != nil && deps.RiverClient != nil {
		guarddutyHandler := handlers.NewGuardDutyHandler(
			deps.PendingAttachmentCache,
			deps.RiverClient,
			deps.S3Factory,
		)
		router.RegisterGuardDutyHandler(guarddutyHandler)
	}

	return &SQSEventService{
		sqsClient: deps.SQSClient,
		router:    router,
	}
}

// GetClient returns the SQS client for the poller worker.
func (s *SQSEventService) GetClient() sqs.Client {
	return s.sqsClient
}

// GetRouter returns the event router for the poller worker.
func (s *SQSEventService) GetRouter() *events.Router {
	return s.router
}
