package svix

//go:generate mockgen -destination=mocks/mock_svix.go -package=mocks github.com/emailapi/api/internal/external/svix Client

import "context"

// Client defines the interface for Svix webhook operations.
type Client interface {
	// EnsureEventTypes registers all event types in Svix.
	// Should be called once on application startup.
	EnsureEventTypes(ctx context.Context) error

	// EnsureApp creates a Svix app for the user if it doesn't exist.
	EnsureApp(ctx context.Context, userID, userName string) error

	// GetAppPortalAccess returns the magic URL for embedding App Portal.
	GetAppPortalAccess(ctx context.Context, userID string) (url string, token string, err error)

	// SendMessage sends a webhook message to all user's configured endpoints.
	SendMessage(ctx context.Context, userID, eventType string, payload interface{}) error
}
