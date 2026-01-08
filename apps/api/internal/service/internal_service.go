package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"net/http"

	"github.com/rs/zerolog/log"
	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
	svix "github.com/svix/svix-webhooks/go"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/polar"
)

// InternalService handles internal webhook endpoints.
type InternalService struct {
	userService        *UserService
	billingService     *BillingService // New: for delegating business logic
	store              Store
	clerkWebhookSecret string
	polarWebhookSecret string
	polarClient        polar.Client
	freeProductID      string
}

// InternalServiceConfig holds configuration for InternalService.
type InternalServiceConfig struct {
	UserService        *UserService
	BillingService     *BillingService // Inject to delegate subscription logic
	Store              Store
	ClerkWebhookSecret string
	PolarWebhookSecret string
	PolarClient        polar.Client
	FreeProductID      string
}

// NewInternalService creates a new InternalService.
func NewInternalService(cfg InternalServiceConfig) *InternalService {
	return &InternalService{
		userService:        cfg.UserService,
		billingService:     cfg.BillingService,
		store:              cfg.Store,
		clerkWebhookSecret: cfg.ClerkWebhookSecret,
		polarWebhookSecret: cfg.PolarWebhookSecret,
		polarClient:        cfg.PolarClient,
		freeProductID:      cfg.FreeProductID,
	}
}

// GuardDutyScanResult represents the scan result from GuardDuty via EventBridge.
type GuardDutyScanResult struct {
	S3Bucket   string `json:"s3_bucket"`
	S3Key      string `json:"s3_key"`
	ScanStatus string `json:"scan_status"` // CLEAN, THREATS_FOUND, UNSUPPORTED, FAILED
	ThreatName string `json:"threat_name"`
}

// HandleGuardDutyScanResult is a placeholder for scan result handling.
// In the stateless architecture, scan results are handled inline during attachment processing.
// This endpoint can be used for logging/monitoring if needed.
func (s *InternalService) HandleGuardDutyScanResult(ctx context.Context, result *GuardDutyScanResult) error {
	// With the new stateless architecture, attachment scanning is handled
	// in the attachment worker before enqueuing the send job.
	// This endpoint is kept for backward compatibility but does nothing.
	return nil
}

// ClerkWebhookPayload represents the Clerk webhook event structure.
type ClerkWebhookPayload struct {
	Type   string          `json:"type"`
	Object string          `json:"object"`
	Data   json.RawMessage `json:"data"`
}

// ClerkUserCreatedData represents user data from Clerk user.created event.
type ClerkUserCreatedData struct {
	ID             string `json:"id"`
	EmailAddresses []struct {
		ID           string `json:"id"`
		EmailAddress string `json:"email_address"`
	} `json:"email_addresses"`
	PrimaryEmailAddressID string `json:"primary_email_address_id"`
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name"`
}

// HandleClerkWebhook processes Clerk user lifecycle events.
func (s *InternalService) HandleClerkWebhook(ctx context.Context, payload []byte, headers http.Header) (bool, string, error) {
	// Verify Svix webhook signature
	wh, err := svix.NewWebhook(s.clerkWebhookSecret)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Svix webhook verifier")
		return false, "", domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "setup_webhook_verifier")
	}

	err = wh.Verify(payload, headers)
	if err != nil {
		log.Warn().Err(err).Msg("Clerk webhook signature verification failed")
		return false, "", domain.ErrPermissionDenied.Clone().WithCause(err).WithMeta("reason", "invalid_signature")
	}

	// Parse the webhook event (payload is verified, use original)
	var event ClerkWebhookPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Error().Err(err).Msg("Failed to parse Clerk webhook payload")
		return false, "", domain.ErrInvalidArgument.Clone().WithCause(err).WithMeta("reason", "invalid_payload")
	}

	log.Info().Str("type", event.Type).Msg("Processing Clerk webhook event")

	// Handle different event types
	switch event.Type {
	case "user.created":
		return s.handleUserCreated(ctx, event.Data)
	case "user.updated":
		// Could handle user updates if needed
		return true, "user.updated event received (no action taken)", nil
	case "user.deleted":
		// Could handle user deletion if needed
		return true, "user.deleted event received (no action taken)", nil
	default:
		log.Debug().Str("type", event.Type).Msg("Ignoring unknown Clerk webhook event type")
		return true, fmt.Sprintf("event type %s ignored", event.Type), nil
	}
}

// handleUserCreated creates a new user from Clerk signup.
func (s *InternalService) handleUserCreated(ctx context.Context, data json.RawMessage) (bool, string, error) {
	var userData ClerkUserCreatedData
	if err := json.Unmarshal(data, &userData); err != nil {
		log.Error().Err(err).Msg("Failed to parse Clerk user data")
		return false, "", domain.ErrInvalidArgument.Clone().WithCause(err).WithMeta("reason", "invalid_user_data")
	}

	// Find primary email address
	var primaryEmail string
	for _, email := range userData.EmailAddresses {
		if email.ID == userData.PrimaryEmailAddressID {
			primaryEmail = email.EmailAddress
			break
		}
	}
	if primaryEmail == "" && len(userData.EmailAddresses) > 0 {
		primaryEmail = userData.EmailAddresses[0].EmailAddress
	}

	if primaryEmail == "" {
		log.Warn().Str("clerk_user_id", userData.ID).Msg("Clerk user has no email address")
		return false, "", domain.ErrInvalidArgument.Clone().WithMeta("reason", "missing_email")
	}

	// Construct user name
	name := fmt.Sprintf("%s %s", userData.FirstName, userData.LastName)
	if name == " " {
		name = primaryEmail // Fallback to email if no name
	}

	// Create user with external ID
	externalID := userData.ID
	createReq := &domain.CreateUserRequest{
		Email:      primaryEmail,
		Name:       name,
		Role:       domain.UserRoleMember,
		ExternalID: &externalID,
	}

	user, err := s.userService.Create(ctx, createReq)
	if err != nil {
		log.Error().Err(err).Str("clerk_user_id", userData.ID).Str("email", primaryEmail).Msg("Failed to create user from Clerk webhook")
		return false, "", domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "create_user_from_webhook")
	}

	log.Info().
		Str("user_id", user.ID).
		Str("clerk_user_id", userData.ID).
		Str("email", primaryEmail).
		Msg("Created new user from Clerk webhook")

	// Create Polar customer and free subscription (async, non-blocking)
	go s.provisionPolarCustomer(context.Background(), user)

	return true, fmt.Sprintf("user created: %s", user.ID), nil
}

// provisionPolarCustomer creates Polar customer and free subscription for new user.
func (s *InternalService) provisionPolarCustomer(ctx context.Context, user *domain.User) {
	if s.polarClient == nil {
		return // Polar not configured
	}

	// Create Polar customer with user ID as external ID
	customerID, err := s.polarClient.CreateCustomer(ctx, user.ID, user.Email, user.Name)
	if err != nil {
		log.Warn().Err(err).Str("user_id", user.ID).Msg("Failed to create Polar customer")
		return
	}

	// Save customer ID to user record
	if s.store != nil {
		if err := s.store.Users().UpdatePolarCustomerID(ctx, user.ID, customerID); err != nil {
			log.Warn().Err(err).Str("user_id", user.ID).Msg("Failed to save Polar customer ID")
		}
	}

	log.Info().
		Str("user_id", user.ID).
		Str("polar_customer_id", customerID).
		Msg("Created Polar customer")

	// Create free subscription if product ID is configured
	if s.freeProductID == "" {
		return
	}

	if err := s.polarClient.CreateFreeSubscription(ctx, customerID, s.freeProductID); err != nil {
		log.Warn().Err(err).Str("user_id", user.ID).Msg("Failed to create free subscription")
		return
	}

	log.Info().
		Str("user_id", user.ID).
		Str("product_id", s.freeProductID).
		Msg("Created free subscription")
}

// HandlePolarWebhook processes Polar subscription events.
func (s *InternalService) HandlePolarWebhook(ctx context.Context, payload []byte, headers http.Header) (bool, string, error) {
	encodedSecret := base64.StdEncoding.EncodeToString([]byte(s.polarWebhookSecret))

	wh, err := standardwebhooks.NewWebhook(encodedSecret)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Standard Webhooks verifier for Polar")
		return false, "", domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "setup_webhook_verifier_polar")
	}

	err = wh.Verify(payload, headers)
	if err != nil {
		log.Warn().Err(err).Msg("Polar webhook signature verification failed")
		return false, "", domain.ErrPermissionDenied.Clone().WithCause(err).WithMeta("reason", "invalid_signature_polar")
	}

	// Parse the webhook event
	var event PolarWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Error().Err(err).Msg("Failed to parse Polar webhook payload")
		return false, "", domain.ErrInvalidArgument.Clone().WithCause(err).WithMeta("reason", "invalid_payload_polar")
	}

	log.Info().Str("type", event.Type).Msg("Processing Polar webhook event")

	if s.billingService == nil {
		return false, "", domain.ErrInternal.Clone().WithMeta("config", "billing_service_missing")
	}

	// Delegate to BillingService
	if err := s.billingService.HandlePolarWebhook(ctx, &event); err != nil {
		log.Error().Err(err).Msg("Failed to handle Polar webhook in BillingService")
		return false, "", err
	}

	return true, "processed", nil
}
