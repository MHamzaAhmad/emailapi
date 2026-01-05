package service

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/polar"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
)

// BillingService handles plan and subscription management.
type BillingService struct {
	polarClient polar.Client
	creditCache redisrepo.CreditCacheInterface
	store       Store
}

// NewBillingService creates a new BillingService.
func NewBillingService(polarClient polar.Client, creditCache redisrepo.CreditCacheInterface, store Store) *BillingService {
	return &BillingService{
		polarClient: polarClient,
		creditCache: creditCache,
		store:       store,
	}
}

// SyncSubscription queries Polar API and invalidates cache so new plan applies immediately.
// Called by frontend immediately after checkout success redirect.
func (s *BillingService) SyncSubscription(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.store.Users().GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if s.polarClient == nil {
		return user, nil // Polar not configured
	}

	// If user has no Polar customer ID, try to find by external ID
	if user.PolarCustomerID == nil || *user.PolarCustomerID == "" {
		customer, err := s.polarClient.GetCustomerByExternalID(ctx, userID)
		if err != nil {
			// No Polar customer yet - stay on current plan
			log.Debug().Str("user_id", userID).Msg("No Polar customer found")
			return user, nil
		}
		// Save the customer ID for future lookups
		if err := s.store.Users().UpdatePolarCustomerID(ctx, userID, customer.ID); err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to save Polar customer ID")
		}
		user.PolarCustomerID = &customer.ID
	}

	// Invalidate credit cache so next request fetches fresh state from Polar
	if s.creditCache != nil {
		if err := s.creditCache.Invalidate(ctx, userID); err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to invalidate credit cache")
		}
	}

	log.Info().Str("user_id", userID).Msg("Subscription synced, cache invalidated")
	return user, nil
}

// CreatePolarCustomer creates a Polar customer for a user if needed.
func (s *BillingService) CreatePolarCustomer(ctx context.Context, userID string) error {
	if s.polarClient == nil {
		return nil // Polar not configured
	}

	user, err := s.store.Users().GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Already has Polar customer
	if user.PolarCustomerID != nil && *user.PolarCustomerID != "" {
		return nil
	}

	// Create customer
	customerID, err := s.polarClient.CreateCustomer(ctx, userID, user.Email, user.Name)
	if err != nil {
		return fmt.Errorf("failed to create Polar customer: %w", err)
	}

	// Save customer ID
	if err := s.store.Users().UpdatePolarCustomerID(ctx, userID, customerID); err != nil {
		return fmt.Errorf("failed to save Polar customer ID: %w", err)
	}

	log.Info().Str("user_id", userID).Str("polar_customer_id", customerID).Msg("Created Polar customer")
	return nil
}

// PolarWebhookEvent represents a Polar webhook event.
type PolarWebhookEvent struct {
	Type       string `json:"type"`
	CustomerID string `json:"customer_id"`
	ProductID  string `json:"product_id"`
}

// HandlePolarWebhook processes Polar subscription webhooks.
// With credit cache architecture, we just invalidate cache on any subscription change.
func (s *BillingService) HandlePolarWebhook(ctx context.Context, event *PolarWebhookEvent) error {
	switch event.Type {
	case "subscription.active", "subscription.canceled", "subscription.updated":
		return s.handleSubscriptionChange(ctx, event)
	}
	return nil
}

func (s *BillingService) handleSubscriptionChange(ctx context.Context, event *PolarWebhookEvent) error {
	// Find user by Polar customer ID
	user, err := s.store.Users().GetByPolarCustomerID(ctx, event.CustomerID)
	if err != nil {
		return fmt.Errorf("user not found for Polar customer %s: %w", event.CustomerID, err)
	}

	// Invalidate credit cache so next request fetches fresh state from Polar
	if s.creditCache != nil {
		if err := s.creditCache.Invalidate(ctx, user.ID); err != nil {
			log.Warn().Err(err).Str("user_id", user.ID).Msg("Failed to invalidate credit cache")
		}
	}

	log.Info().Str("user_id", user.ID).Str("event", event.Type).Msg("Credit cache invalidated via webhook")
	return nil
}

// GetPlans returns all available pricing plans.
func (s *BillingService) GetPlans() []domain.PlanInfo {
	plans := domain.AllPlans()
	result := make([]domain.PlanInfo, 0, len(plans))
	for _, p := range plans {
		info := p.GetInfo()
		result = append(result, info)
	}
	return result
}

// CreateCheckoutSession creates a checkout session for upgrading to a plan.
func (s *BillingService) CreateCheckoutSession(ctx context.Context, userID, planID, successURL string) (string, error) {
	if s.polarClient == nil {
		return "", fmt.Errorf("polar not configured")
	}

	// Map plan ID to product ID
	productID, ok := domain.GetProductIDFromPlanID(planID)
	if !ok {
		return "", fmt.Errorf("invalid plan: %s", planID)
	}

	checkoutURL, err := s.polarClient.CreateCheckoutSession(ctx, polar.CheckoutParams{
		ProductID:          productID,
		ExternalCustomerID: userID,
		SuccessURL:         successURL,
	})
	if err != nil {
		return "", err
	}

	return checkoutURL, nil
}

// GetCustomerPortalUrl returns the customer portal URL for subscription management.
func (s *BillingService) GetCustomerPortalUrl(ctx context.Context, userID string) (string, error) {
	if s.polarClient == nil {
		return "", fmt.Errorf("polar not configured")
	}

	user, err := s.store.Users().GetByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	if user.PolarCustomerID == nil || *user.PolarCustomerID == "" {
		return "", fmt.Errorf("user has no Polar customer ID")
	}

	portalURL, err := s.polarClient.CreateCustomerPortal(ctx, *user.PolarCustomerID)
	if err != nil {
		return "", err
	}

	return portalURL, nil
}
