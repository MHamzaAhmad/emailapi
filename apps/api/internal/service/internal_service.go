package service

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"

	"github.com/rs/zerolog/log"
	svix "github.com/svix/svix-webhooks/go"

	"github.com/emailapi/api/internal/domain"
)

// InternalService handles internal webhook endpoints.
type InternalService struct {
	userService        *UserService
	clerkWebhookSecret string
}

// InternalServiceConfig holds configuration for InternalService.
type InternalServiceConfig struct {
	UserService        *UserService
	ClerkWebhookSecret string
}

// NewInternalService creates a new InternalService.
func NewInternalService(cfg InternalServiceConfig) *InternalService {
	return &InternalService{
		userService:        cfg.UserService,
		clerkWebhookSecret: cfg.ClerkWebhookSecret,
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
		return false, "", fmt.Errorf("webhook verification setup failed: %w", err)
	}

	err = wh.Verify(payload, headers)
	if err != nil {
		log.Warn().Err(err).Msg("Clerk webhook signature verification failed")
		return false, "", fmt.Errorf("webhook signature verification failed: %w", err)
	}

	// Parse the webhook event (payload is verified, use original)
	var event ClerkWebhookPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Error().Err(err).Msg("Failed to parse Clerk webhook payload")
		return false, "", fmt.Errorf("failed to parse webhook payload: %w", err)
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
		return false, "", fmt.Errorf("failed to parse user data: %w", err)
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
		return false, "", fmt.Errorf("user has no email address")
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
		return false, "", fmt.Errorf("failed to create user: %w", err)
	}

	log.Info().
		Str("user_id", user.ID).
		Str("clerk_user_id", userData.ID).
		Str("email", primaryEmail).
		Msg("Created new user from Clerk webhook")

	return true, fmt.Sprintf("user created: %s", user.ID), nil
}
