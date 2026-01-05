package polar

import (
	"context"
	"fmt"
	"time"

	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
	"github.com/polarsource/polar-go/models/operations"
)

// Client wraps the Polar Go SDK for subscription and billing management.
//
//go:generate mockgen -destination=mocks/mock_polar.go -package=mocks github.com/emailapi/api/internal/external/polar Client
type Client interface {
	// CreateCustomer creates a Polar customer for a user.
	CreateCustomer(ctx context.Context, userID, email, name string) (string, error)

	// GetCustomerByExternalID finds customer by external ID (our user ID).
	GetCustomerByExternalID(ctx context.Context, userID string) (*Customer, error)

	// GetSubscription gets active subscription for customer.
	GetSubscription(ctx context.Context, customerID string) (*Subscription, error)

	// IngestEmailEvent sends a usage event to Polar for billing.
	IngestEmailEvent(ctx context.Context, userID string, count int64) error

	// CreateCheckoutSession creates a checkout session for plan upgrade.
	CreateCheckoutSession(ctx context.Context, params CheckoutParams) (string, error)

	// CreateCustomerPortal creates a customer portal session.
	CreateCustomerPortal(ctx context.Context, customerID string) (string, error)
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

// polarClient implements Client.
type polarClient struct {
	sdk     *polargo.Polar
	meterID string // Email meter ID for event ingestion
}

// NewClient creates a new Polar client.
func NewClient(accessToken, meterID string) (Client, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("polar access token is required")
	}

	sdk := polargo.New(polargo.WithSecurity(accessToken))

	return &polarClient{
		sdk:     sdk,
		meterID: meterID,
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
		return nil, fmt.Errorf("no active subscription found")
	}

	sub := resp.ListResourceSubscription.Items[0]
	return &Subscription{
		ID:          sub.ID,
		Status:      string(sub.Status),
		ProductID:   sub.ProductID,
		ProductName: sub.Product.Name,
	}, nil
}

// IngestEmailEvent sends a usage event to Polar for billing.
func (c *polarClient) IngestEmailEvent(ctx context.Context, userID string, count int64) error {
	event := components.CreateEventsEventCreateExternalCustomer(components.EventCreateExternalCustomer{
		Name:               "email_sent",
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
	resp, err := c.sdk.Checkouts.Create(ctx, components.CheckoutCreate{
		Products:           []string{params.ProductID},
		ExternalCustomerID: polargo.String(params.ExternalCustomerID),
		SuccessURL:         polargo.String(params.SuccessURL),
	})
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
