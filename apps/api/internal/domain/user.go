package domain

import (
	"time"
)

// UserRole represents a user's role in the system.
type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
	UserRoleMember UserRole = "member"
)

// User represents a user entity in the system.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         UserRole  `json:"role"`
	APIKey       string    `json:"-"` // Never expose in JSON
	APIKeyPrefix string    `json:"api_key_prefix,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateUserRequest represents a request to create a new user.
type CreateUserRequest struct {
	Email string   `json:"email" binding:"required,email"`
	Name  string   `json:"name" binding:"required"`
	Role  UserRole `json:"role,omitempty"`
}

// UpdateUserRequest represents a request to update a user.
type UpdateUserRequest struct {
	Email    *string   `json:"email,omitempty" binding:"omitempty,email"`
	Name     *string   `json:"name,omitempty"`
	Role     *UserRole `json:"role,omitempty"`
	IsActive *bool     `json:"is_active,omitempty"`
}

// APIKeyResponse represents the response when generating a new API key.
type APIKeyResponse struct {
	APIKey    string    `json:"api_key"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}
