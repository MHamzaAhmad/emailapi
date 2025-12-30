package service

import (
	"github.com/emailapi/api/internal/repository"
)

// Store is the single interface that aggregates all repository access.
// This is injected into the Service layer for testability.
type Store interface {
	// Users returns the user repository.
	Users() repository.UserRepository

	// APIKeys returns the API key repository.
	APIKeys() repository.APIKeyRepository

	// Domains returns the domain repository.
	Domains() repository.DomainRepository

	// Reputation returns the reputation repository.
	Reputation() repository.ReputationRepository

	// Unsubscribe returns the unsubscribe repository.
	Unsubscribe() repository.UnsubscribeRepository

	// Close closes any underlying connections.
	Close()
}
