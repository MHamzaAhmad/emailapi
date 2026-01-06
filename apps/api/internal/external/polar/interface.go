package polar

//go:generate mockgen -destination=mocks/mock_polar.go -package=mocks github.com/emailapi/api/internal/external/polar Client

import "context"

// Client wraps the Polar Go SDK for subscription and billing management.
type Client interface {
	// CreateCustomer creates a Polar customer for a user.
	CreateCustomer(ctx context.Context, userID, email, name string) (string, error)

	// GetCustomerByExternalID finds customer by external ID (our user ID).
	GetCustomerByExternalID(ctx context.Context, userID string) (*Customer, error)

	// GetCustomerStateByExternalID fetches customer state including credit balance.
	GetCustomerStateByExternalID(ctx context.Context, userID string) (*CustomerState, error)

	// GetSubscription gets active subscription for customer.
	GetSubscription(ctx context.Context, customerID string) (*Subscription, error)

	// IngestEmailEvent sends a usage event to Polar for billing.
	IngestEmailEvent(ctx context.Context, userID string, count int64) error

	// CreateCheckoutSession creates a checkout session for plan upgrade.
	CreateCheckoutSession(ctx context.Context, params CheckoutParams) (string, error)

	// CreateCustomerPortal creates a customer portal session.
	CreateCustomerPortal(ctx context.Context, customerID string) (string, error)

	// CreateFreeSubscription creates a subscription for the free product (on signup).
	CreateFreeSubscription(ctx context.Context, customerID, freeProductID string) error
}

// CheckoutParams for creating a checkout session.
type CheckoutParams struct {
	ProductID          string
	ExternalCustomerID string
	SuccessURL         string
}

// Customer represents a Polar customer.
type Customer struct {
	ID         string
	Email      string
	ExternalID string
}

// Subscription represents an active subscription.
type Subscription struct {
	ID          string
	Status      string // active, past_due, canceled
	ProductID   string
	ProductName string
}

// CustomerState represents customer state with credit balance from Polar.
type CustomerState struct {
	CustomerID    string
	IsPaid        bool   // Has paid subscription (starter/growth)
	PlanType      string // "free", "starter", "growth"
	CreditBalance int64  // From active_meters[0].balance
	ProductID     string // Current product ID
}
