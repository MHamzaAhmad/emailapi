package polar

import (
	"context"
	"fmt"
	"time"

	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
	"github.com/polarsource/polar-go/models/operations"
)

// polarClient implements Client.
type polarClient struct {
	sdk       *polargo.Polar
	meterName string // Event name for meter (e.g., "emails")
}

// NewClient creates a new Polar client.
func NewClient(accessToken, meterName string) (Client, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("polar access token is required")
	}

	sdk := polargo.New(polargo.WithSecurity(accessToken), polargo.WithServerURL("https://sandbox-api.polar.sh"))

	return &polarClient{
		sdk:       sdk,
		meterName: meterName,
	}, nil
}

// CreateCustomer creates a Polar customer for a user.
func (c *polarClient) CreateCustomer(ctx context.Context, userID, email, name string) (string, error) {
	resp, err := c.sdk.Customers.Create(ctx, components.CustomerCreate{
		Email:      email,
		Name:       polargo.String(name),
		ExternalID: polargo.String(userID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create Polar customer: %w", err)
	}

	return resp.Customer.ID, nil
}

// GetCustomerByExternalID finds customer by external ID (our user ID).
func (c *polarClient) GetCustomerByExternalID(ctx context.Context, userID string) (*Customer, error) {
	resp, err := c.sdk.Customers.GetExternal(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Polar customer: %w", err)
	}

	var externalID string
	if resp.Customer.ExternalID != nil {
		externalID = *resp.Customer.ExternalID
	}

	return &Customer{
		ID:         resp.Customer.ID,
		Email:      resp.Customer.Email,
		ExternalID: externalID,
	}, nil
}

// GetSubscription gets active subscription for customer.
func (c *polarClient) GetSubscription(ctx context.Context, customerID string) (*Subscription, error) {
	resp, err := c.sdk.Subscriptions.List(ctx, operations.SubscriptionsListRequest{
		CustomerID: polargo.Pointer(operations.CreateCustomerIDFilterStr(customerID)),
		Active:     polargo.Bool(true),
		Limit:      polargo.Int64(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list Polar subscriptions: %w", err)
	}

	if len(resp.ListResourceSubscription.Items) == 0 {
		return nil, nil // No active subscription
	}

	sub := resp.ListResourceSubscription.Items[0]
	return &Subscription{
		ID:          sub.ID,
		Status:      string(sub.Status),
		ProductID:   sub.ProductID,
		ProductName: sub.Product.Name,
		Amount:      sub.Amount,
	}, nil
}

// GetActiveSubscriptionByExternalID gets active subscription by external customer ID.
func (c *polarClient) GetActiveSubscriptionByExternalID(ctx context.Context, userID string) (*Subscription, error) {
	// First get customer by external ID
	customer, err := c.GetCustomerByExternalID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return c.GetSubscription(ctx, customer.ID)
}

// IngestEmailEvent sends a usage event to Polar for billing.
func (c *polarClient) IngestEmailEvent(ctx context.Context, userID string, count int64) error {
	// Use configured meter name as event name (Polar matches events to meters by name)
	eventName := c.meterName
	if eventName == "" {
		eventName = "emails" // Default fallback
	}

	event := components.CreateEventsEventCreateExternalCustomer(components.EventCreateExternalCustomer{
		Name:               eventName,
		ExternalCustomerID: userID,
		Timestamp:          polargo.Pointer(time.Now().UTC()),
		Metadata: map[string]components.EventMetadataInput{
			"count": components.CreateEventMetadataInputInteger(count),
		},
	})

	_, err := c.sdk.Events.Ingest(ctx, components.EventsIngest{
		Events: []components.Events{event},
	})
	if err != nil {
		return fmt.Errorf("failed to ingest Polar event: %w", err)
	}

	return nil
}

// CreateCheckoutSession creates a checkout session for plan upgrade.
func (c *polarClient) CreateCheckoutSession(ctx context.Context, params CheckoutParams) (string, error) {
	checkoutCreate := components.CheckoutCreate{
		Products:           []string{params.ProductID},
		ExternalCustomerID: polargo.String(params.ExternalCustomerID),
		SuccessURL:         polargo.String(params.SuccessURL),
	}

	// If upgrading an existing subscription, include subscription_id
	if params.SubscriptionID != "" {
		checkoutCreate.SubscriptionID = polargo.String(params.SubscriptionID)
	}

	resp, err := c.sdk.Checkouts.Create(ctx, checkoutCreate)
	if err != nil {
		return "", fmt.Errorf("failed to create Polar checkout: %w", err)
	}

	return resp.Checkout.URL, nil
}

// CreateCustomerPortal creates a customer portal session.
func (c *polarClient) CreateCustomerPortal(ctx context.Context, customerID string) (string, error) {
	resp, err := c.sdk.CustomerSessions.Create(ctx,
		operations.CreateCustomerSessionsCreateCustomerSessionCreateCustomerSessionCustomerIDCreate(
			components.CustomerSessionCustomerIDCreate{
				CustomerID: customerID,
			},
		),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create Polar customer portal: %w", err)
	}

	return resp.CustomerSession.CustomerPortalURL, nil
}

// GetCustomerStateByExternalID fetches customer state including credit balance.
func (c *polarClient) GetCustomerStateByExternalID(ctx context.Context, userID string) (*CustomerState, error) {
	resp, err := c.sdk.Customers.GetStateExternal(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Polar customer state: %w", err)
	}

	state := &CustomerState{
		CustomerID:    resp.CustomerState.ID,
		IsPaid:        false,
		PlanType:      "free",
		CreditBalance: 0,
	}

	// Check active subscriptions to determine plan type
	if len(resp.CustomerState.ActiveSubscriptions) > 0 {
		sub := resp.CustomerState.ActiveSubscriptions[0]
		state.ProductID = sub.ProductID

		// Determine if paid subscription based on amount (free = 0)
		if sub.Amount > 0 {
			state.IsPaid = true
			// Map product ID to plan type will be done by caller using domain.GetPlanFromProductID
			state.PlanType = "paid" // Caller maps to specific plan
		}
	}

	// Get credit balance from active meters
	if len(resp.CustomerState.ActiveMeters) > 0 {
		state.CreditBalance = int64(resp.CustomerState.ActiveMeters[0].Balance)
	}

	return state, nil
}

// CreateFreeSubscription creates a subscription for the free product (on signup).
func (c *polarClient) CreateFreeSubscription(ctx context.Context, customerID, freeProductID string) error {
	_, err := c.sdk.Subscriptions.Create(ctx,
		operations.CreateSubscriptionsCreateSubscriptionCreateSubscriptionCreateCustomer(
			components.SubscriptionCreateCustomer{
				ProductID:  freeProductID,
				CustomerID: customerID,
			},
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create free subscription: %w", err)
	}

	return nil
}
