package domain

import (
	"time"
)

// Scope represents an API key permission scope.
type Scope string

const (
	// Email scopes
	ScopeEmailSend Scope = "email:send"
	ScopeEmailRead Scope = "email:read"

	// Domain scopes
	ScopeDomainRead  Scope = "domain:read"
	ScopeDomainWrite Scope = "domain:write"

	// API key management scopes
	ScopeApiKeyRead  Scope = "apikey:read"
	ScopeApiKeyWrite Scope = "apikey:write"

	// User scopes
	ScopeUserRead  Scope = "user:read"
	ScopeUserWrite Scope = "user:write"
)

// AllScopes returns all available scopes.
func AllScopes() []Scope {
	return []Scope{
		ScopeEmailSend,
		ScopeEmailRead,
		ScopeDomainRead,
		ScopeDomainWrite,
		ScopeApiKeyRead,
		ScopeApiKeyWrite,
		ScopeUserRead,
		ScopeUserWrite,
	}
}

// DefaultScopes returns the default scopes for new API keys.
func DefaultScopes() []Scope {
	return []Scope{
		ScopeEmailSend,
		ScopeEmailRead,
		ScopeDomainRead,
		ScopeApiKeyRead,
		ScopeUserRead,
	}
}

// ValidateScopes checks if all provided scopes are valid.
func ValidateScopes(scopes []Scope) bool {
	validScopes := make(map[Scope]bool)
	for _, s := range AllScopes() {
		validScopes[s] = true
	}
	for _, s := range scopes {
		if !validScopes[s] {
			return false
		}
	}
	return true
}

// APIKey represents an API key entity in the system.
type APIKey struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"` // Never expose in JSON
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []Scope    `json:"scopes"`
	IsActive   bool       `json:"is_active"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// IsExpired checks if the API key has expired.
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// HasScope checks if the API key has the specified scope.
func (k *APIKey) HasScope(scope Scope) bool {
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// CreateAPIKeyRequest represents a request to create a new API key.
type CreateAPIKeyRequest struct {
	Name      string     `json:"name" binding:"required"`
	Scopes    []Scope    `json:"scopes,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// UpdateAPIKeyRequest represents a request to update an API key.
type UpdateAPIKeyRequest struct {
	Name      *string    `json:"name,omitempty"`
	Scopes    []Scope    `json:"scopes,omitempty"`
	IsActive  *bool      `json:"is_active,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// APIKeyResponse represents the response when creating a new API key.
// The raw key is only returned once during creation.
type APIKeyResponse struct {
	APIKey *APIKey `json:"api_key"`
	RawKey string  `json:"raw_key"` // Only returned once during creation
}
