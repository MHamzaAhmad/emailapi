package service

import (
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	"github.com/emailapi/api/internal/external/svix"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	pgrepo "github.com/emailapi/api/internal/repository/postgres"
	"github.com/emailapi/api/internal/validation"

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
	InboundEmail *InboundEmailService
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
	Store           Store
	SESClient       ses.Client
	S3Client        s3.Client
	Region          string
	RiverClient     *river.Client[pgx.Tx]
	PGEmailRepo     *pgrepo.EmailRepository
	CHEmailRepo     *chrepo.EmailRepository
	SvixClient      svix.Client
	S3InboundBucket string
}

// NewWithDeps creates a new Service with all dependencies.
func NewWithDeps(deps ServiceDeps) *Service {
	svc := New(deps.Store)
	svc.Domain = NewDomainService(deps.Store, deps.SESClient, deps.Region)

	// Create email validator with domain checker
	emailValidator := validation.NewEmailValidator(svc.Domain)

	svc.Email = NewEmailService(deps.RiverClient, deps.PGEmailRepo, deps.CHEmailRepo, emailValidator, deps.SESClient)
	svc.Internal = NewInternalService(deps.PGEmailRepo, deps.RiverClient)

	// Initialize Svix-dependent services if client is available
	if deps.SvixClient != nil {
		svc.Webhook = NewWebhookService(deps.SvixClient)
		svc.InboundEmail = NewInboundEmailService(
			deps.S3Client,
			deps.PGEmailRepo,
			deps.CHEmailRepo,
			deps.SvixClient,
			deps.S3InboundBucket,
		)
	}

	return svc
}

// NewWithSES creates a new Service with the given Store and SES client (legacy).
// Deprecated: Use NewWithDeps for new code.
func NewWithSES(
	store Store,
	sesClient ses.Client,
	region string,
	riverClient *river.Client[pgx.Tx],
	s3Client s3.Client,
	pgEmailRepo *pgrepo.EmailRepository,
	chRepo *chrepo.EmailRepository,
) *Service {
	return NewWithDeps(ServiceDeps{
		Store:       store,
		SESClient:   sesClient,
		S3Client:    s3Client,
		Region:      region,
		RiverClient: riverClient,
		PGEmailRepo: pgEmailRepo,
		CHEmailRepo: chRepo,
	})
}
