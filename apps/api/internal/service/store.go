package service

import (
	"github.com/emailapi/api/internal/repository"
)

// Store is the single interface that aggregates all repository access.
// This is injected into the Service layer for testability.
type Store interface {
	// Emails returns the email repository.
	Emails() repository.EmailRepository

	// Users returns the user repository.
	Users() repository.UserRepository

	// Webhooks returns the webhook repository.
	Webhooks() repository.WebhookRepository

	// Domains returns the domain repository.
	Domains() repository.DomainRepository

	// Close closes any underlying connections.
	Close()
}
