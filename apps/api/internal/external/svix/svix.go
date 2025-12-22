package svix

import "context"

// Client interface for Svix webhook operations.
type Client interface {
	// EnsureApp creates a Svix app for the user if it doesn't exist.
	EnsureApp(ctx context.Context, userID, userName string) error

	// GetAppPortalAccess returns the magic URL for embedding App Portal.
	GetAppPortalAccess(ctx context.Context, userID string) (url, token string, err error)

	// SendMessage sends a webhook message to all user's configured endpoints.
	SendMessage(ctx context.Context, userID, eventType string, payload interface{}) error
}
