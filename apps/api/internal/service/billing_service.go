package service

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/polar"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
)

// BillingService handles plan and subscription management.
type BillingService struct {
	polarClient polar.Client
	creditCache redisrepo.CreditCacheInterface
	usageCache  redisrepo.UsageCacheInterface
	store       Store
}

// NewBillingService creates a new BillingService.
// NewBillingService creates a new BillingService.
func NewBillingService(
	polarClient polar.Client,
	creditCache redisrepo.CreditCacheInterface,
	usageCache redisrepo.UsageCacheInterface,
	store Store,
) *BillingService {
	return &BillingService{
		polarClient: polarClient,
		creditCache: creditCache,
		usageCache:  usageCache,
		store:       store,
	}
}

// CreatePolarCustomer creates a Polar customer for a user if needed.
func (s *BillingService) CreatePolarCustomer(ctx context.Context, userID string) error {
	if s.polarClient == nil {
		return nil // Polar not configured
	}

	user, err := s.store.Users().GetByID(ctx, userID)
	if err != nil {
		return domain.ErrUserNotFound.Clone().WithCause(err)
	}

	// Already has Polar customer
	if user.PolarCustomerID != nil && *user.PolarCustomerID != "" {
		return nil
	}

	// Create customer
	customerID, err := s.polarClient.CreateCustomer(ctx, userID, user.Email, user.Name)
	if err != nil {
		return domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "create_polar_customer")
	}

	// Save customer ID
	if err := s.store.Users().UpdatePolarCustomerID(ctx, userID, customerID); err != nil {
		return domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "save_polar_customer_id")
	}

	log.Info().Str("user_id", userID).Str("polar_customer_id", customerID).Msg("Created Polar customer")
	return nil
}

// PolarWebhookEvent represents a Polar webhook event.
type PolarWebhookEvent struct {
	Type string `json:"type"`
	Data struct {
		CustomerID string `json:"customer_id"`
		ProductID  string `json:"product_id"`
		Customer   struct {
			ExternalID string `json:"external_id"` // This is our user_id
		} `json:"customer"`
	} `json:"data"`
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
	// Use external_id (our user_id) from the webhook for direct lookup
	userID := event.Data.Customer.ExternalID
	if userID == "" {
		// Fallback to looking up by Polar customer_id if external_id is missing
		user, err := s.store.Users().GetByPolarCustomerID(ctx, event.Data.CustomerID)
		if err != nil {
			return domain.ErrUserNotFound.Clone().WithCause(err).WithMeta("polar_customer_id", event.Data.CustomerID)
		}
		userID = user.ID
	}

	// Invalidate credit cache so next request fetches fresh state from Polar
	if s.creditCache != nil {
		if err := s.creditCache.Invalidate(ctx, userID); err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to invalidate credit cache")
		}
	}

	// Invalidate usage cache so plan limits are updated immediately
	if s.usageCache != nil {
		if err := s.usageCache.InvalidatePlanState(ctx, userID); err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to invalidate usage cache")
		}
	}

	log.Info().Str("user_id", userID).Str("event", event.Type).Msg("Caches invalidated via webhook")
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
// Returns error if user has a paid subscription (should use portal instead).
func (s *BillingService) CreateCheckoutSession(ctx context.Context, userID, planID, successURL string) (string, error) {
	if s.polarClient == nil {
		return "", domain.ErrInternal.Clone().WithMeta("config", "polar_missing")
	}

	// Map plan ID to product ID
	productID, ok := domain.GetProductIDFromPlanID(planID)
	if !ok {
		return "", domain.ErrInvalidArgument.Clone().WithMeta("field", "plan_id").WithMeta("value", planID)
	}

	// Check current subscription to determine checkout strategy
	sub, err := s.polarClient.GetActiveSubscriptionByExternalID(ctx, userID)
	if err != nil {
		// If error fetching subscription, proceed without subscription_id
		log.Warn().Err(err).Str("user_id", userID).Msg("Could not fetch subscription, proceeding without upgrade")
	}

	checkoutParams := polar.CheckoutParams{
		ProductID:          productID,
		ExternalCustomerID: userID,
		SuccessURL:         successURL,
	}

	if sub != nil {
		// User has an active subscription
		if sub.Amount > 0 {
			// Paid subscription - user should use customer portal to change plans
			return "", domain.ErrAlreadyExists.Clone().WithMeta("resource", "paid_subscription")
		}
		// Free subscription - pass subscription_id for upgrade
		checkoutParams.SubscriptionID = sub.ID
	}

	checkoutURL, err := s.polarClient.CreateCheckoutSession(ctx, checkoutParams)
	if err != nil {
		return "", err
	}

	return checkoutURL, nil
}

// GetCustomerPortalUrl returns the customer portal URL for subscription management.
func (s *BillingService) GetCustomerPortalUrl(ctx context.Context, userID string) (string, error) {
	if s.polarClient == nil {
		return "", domain.ErrInternal.Clone().WithMeta("config", "polar_missing")
	}

	user, err := s.store.Users().GetByID(ctx, userID)
	if err != nil {
		return "", domain.ErrUserNotFound.Clone().WithCause(err)
	}

	if user.PolarCustomerID == nil || *user.PolarCustomerID == "" {
		return "", domain.ErrInternal.Clone().WithMeta("reason", "missing_customer_id")
	}

	portalURL, err := s.polarClient.CreateCustomerPortal(ctx, *user.PolarCustomerID)
	if err != nil {
		return "", err
	}

	return portalURL, nil
}

// SubscriptionInfo contains current subscription details for frontend.
type SubscriptionInfo struct {
	HasSubscription bool   // Whether user has any active subscription
	IsPaid          bool   // Whether subscription is paid (vs free)
	PlanID          string // Current plan ID ("free", "starter", "growth")
	ProductID       string // Polar product ID
	SubscriptionID  string // Subscription ID for upgrades
	PolarCustomerID string // Polar Customer ID
}

// GetSubscriptionInfo returns current subscription info for the frontend.
func (s *BillingService) GetSubscriptionInfo(ctx context.Context, userID string) (*SubscriptionInfo, error) {
	info := &SubscriptionInfo{
		HasSubscription: false,
		IsPaid:          false,
		PlanID:          "free",
	}

	if s.polarClient == nil {
		return info, nil
	}

	// Check cache first
	if s.creditCache != nil {
		cached, err := s.creditCache.GetSubscription(ctx, userID)
		if err == nil && cached != nil {
			return &SubscriptionInfo{
				HasSubscription: cached.HasSubscription,
				IsPaid:          cached.IsPaid,
				PlanID:          cached.PlanID,
				ProductID:       cached.ProductID,
				SubscriptionID:  cached.SubscriptionID,
				PolarCustomerID: cached.PolarCustomerID,
			}, nil
		}
	}

	user, err := s.store.Users().GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrUserNotFound.Clone().WithCause(err)
	}
	if user.PolarCustomerID != nil {
		info.PolarCustomerID = *user.PolarCustomerID
	}

	sub, err := s.polarClient.GetActiveSubscriptionByExternalID(ctx, userID)
	if err != nil {
		log.Debug().Err(err).Str("user_id", userID).Msg("Could not fetch subscription")
		return info, nil // Return default free info
	}

	if sub == nil {
		return info, nil
	}

	info.HasSubscription = true
	info.SubscriptionID = sub.ID
	info.ProductID = sub.ProductID

	// Re-calculate basic fields based on sub
	if sub.Amount > 0 {
		info.IsPaid = true
	}
	// Determine plan ID
	switch {
	case sub.ProductName == "Starter" || sub.Amount == 1250:
		info.PlanID = "starter"
	case sub.ProductName == "Growth" || sub.Amount == 5000:
		info.PlanID = "growth"
	default:
		info.PlanID = "free"
	}

	// Update cache
	if s.creditCache != nil {
		if err := s.creditCache.SetSubscription(ctx, userID, &redisrepo.CachedSubscription{
			HasSubscription: info.HasSubscription,
			IsPaid:          info.IsPaid,
			PlanID:          info.PlanID,
			ProductID:       info.ProductID,
			SubscriptionID:  info.SubscriptionID,
			PolarCustomerID: info.PolarCustomerID,
		}); err != nil {
			log.Warn().Err(err).Msg("Failed to cache subscription")
		}
	}

	return info, nil
}
