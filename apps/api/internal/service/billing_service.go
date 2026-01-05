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
	usageCache  redisrepo.UsageCacheInterface
	store       Store
}

// NewBillingService creates a new BillingService.
func NewBillingService(polarClient polar.Client, usageCache redisrepo.UsageCacheInterface, store Store) *BillingService {
	return &BillingService{
		polarClient: polarClient,
		usageCache:  usageCache,
		store:       store,
	}
}

// SyncSubscription queries Polar API directly and updates user plan.
// Called by frontend immediately after checkout success redirect.
// This bypasses webhook delay - user sees upgrade instantly.
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

	// Query Polar for current subscription
	sub, err := s.polarClient.GetSubscription(ctx, *user.PolarCustomerID)
	if err != nil {
		// No active subscription - stay on free
		log.Debug().Str("user_id", userID).Msg("No active Polar subscription")
		return user, nil
	}

	// Map subscription to plan
	newPlan := mapProductToPlan(sub.ProductID)

	// Update if different
	if newPlan != user.Plan {
		if err := s.store.Users().UpdatePlan(ctx, userID, newPlan); err != nil {
			return nil, fmt.Errorf("failed to update plan: %w", err)
		}
		oldPlan := user.Plan
		user.Plan = newPlan

		// Invalidate cache so new limits apply immediately
		if s.usageCache != nil {
			if err := s.usageCache.InvalidatePlanState(ctx, userID); err != nil {
				log.Warn().Err(err).Str("user_id", userID).Msg("Failed to invalidate cache")
			}
		}

		log.Info().
			Str("user_id", userID).
			Str("old_plan", string(oldPlan)).
			Str("new_plan", string(newPlan)).
			Msg("User plan upgraded via sync")
	}

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

// HandlePolarWebhook processes Polar subscription webhooks (backup for sync).
func (s *BillingService) HandlePolarWebhook(ctx context.Context, event *PolarWebhookEvent) error {
	switch event.Type {
	case "subscription.active":
		return s.handleSubscriptionActive(ctx, event)
	case "subscription.canceled":
		return s.handleSubscriptionCanceled(ctx, event)
	}
	return nil
}

func (s *BillingService) handleSubscriptionActive(ctx context.Context, event *PolarWebhookEvent) error {
	// Find user by Polar customer ID
	user, err := s.store.Users().GetByPolarCustomerID(ctx, event.CustomerID)
	if err != nil {
		return fmt.Errorf("user not found for Polar customer %s: %w", event.CustomerID, err)
	}

	// Map Polar product to our plan
	plan := mapProductToPlan(event.ProductID)

	// Only update if different (sync might have already set it)
	if plan != user.Plan {
		if err := s.store.Users().UpdatePlan(ctx, user.ID, plan); err != nil {
			return fmt.Errorf("failed to update plan: %w", err)
		}
		// Invalidate cached state
		if s.usageCache != nil {
			if err := s.usageCache.InvalidatePlanState(ctx, user.ID); err != nil {
				log.Warn().Err(err).Str("user_id", user.ID).Msg("Failed to invalidate cache")
			}
		}
		log.Info().Str("user_id", user.ID).Str("plan", string(plan)).Msg("Plan updated via webhook")
	}
	return nil
}

func (s *BillingService) handleSubscriptionCanceled(ctx context.Context, event *PolarWebhookEvent) error {
	// Find user by Polar customer ID
	user, err := s.store.Users().GetByPolarCustomerID(ctx, event.CustomerID)
	if err != nil {
		return fmt.Errorf("user not found for Polar customer %s: %w", event.CustomerID, err)
	}

	// Downgrade to free
	if err := s.store.Users().UpdatePlan(ctx, user.ID, domain.UserPlanFree); err != nil {
		return fmt.Errorf("failed to downgrade plan: %w", err)
	}

	// Invalidate cached state
	if s.usageCache != nil {
		if err := s.usageCache.InvalidatePlanState(ctx, user.ID); err != nil {
			log.Warn().Err(err).Str("user_id", user.ID).Msg("Failed to invalidate cache")
		}
	}

	log.Info().Str("user_id", user.ID).Msg("Plan downgraded to free via webhook")
	return nil
}

// ProductPlanMap maps Polar product IDs to our plan enum.
// This should be configured via environment variables in production.
var ProductPlanMap = map[string]domain.UserPlan{
	"prod_scale_monthly": domain.UserPlanScale,
	"prod_scale_yearly":  domain.UserPlanScale,
	"prod_payg":          domain.UserPlanPAYG,
}

// mapProductToPlan maps Polar product IDs to our plan enum.
func mapProductToPlan(productID string) domain.UserPlan {
	if plan, ok := ProductPlanMap[productID]; ok {
		return plan
	}
	return domain.UserPlanFree
}
