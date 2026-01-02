package service

import (
	"github.com/emailapi/api/internal/repository/postgres"
)

// Store is the single interface that aggregates all repository access.
// This is injected into the Service layer for testability.
type Store interface {
	// Users returns the user repository.
	Users() postgres.UserRepository

	// APIKeys returns the API key repository.
	APIKeys() postgres.APIKeyRepository

	// Domains returns the domain repository.
	Domains() postgres.DomainRepository

	// Reputation returns the reputation repository.
	Reputation() postgres.ReputationRepository

	// Unsubscribe returns the unsubscribe repository.
	Unsubscribe() postgres.UnsubscribeRepository

	// Close closes any underlying connections.
	Close()
}
