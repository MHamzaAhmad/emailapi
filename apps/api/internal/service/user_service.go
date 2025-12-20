package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	"github.com/emailapi/api/internal/domain"
)

// UserService handles user business logic.
type UserService struct {
	store Store
}

// NewUserService creates a new UserService.
func NewUserService(store Store) *UserService {
	return &UserService{store: store}
}

// Create creates a new user with an API key.
func (s *UserService) Create(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, string, error) {
	// Check if user already exists
	existing, _ := s.store.Users().GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, "", fmt.Errorf("user with email already exists")
	}

	// Generate API key
	apiKey, hashedKey, prefix, err := s.generateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Set default role
	role := req.Role
	if role == "" {
		role = domain.UserRoleMember
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		Name:         req.Name,
		Role:         role,
		APIKey:       hashedKey,
		APIKeyPrefix: prefix,
		IsActive:     true,
	}

	if err := s.store.Users().Create(ctx, user); err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	return user, apiKey, nil
}

// GetByID retrieves a user by ID.
func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.store.Users().GetByID(ctx, id)
}

// GetByAPIKey retrieves a user by their API key.
func (s *UserService) GetByAPIKey(ctx context.Context, apiKey string) (*domain.User, error) {
	hashedKey := s.hashAPIKey(apiKey)
	return s.store.Users().GetByAPIKey(ctx, hashedKey)
}

// RegenerateAPIKey generates a new API key for a user.
func (s *UserService) RegenerateAPIKey(ctx context.Context, userID string) (*domain.APIKeyResponse, error) {
	// Verify user exists
	_, err := s.store.Users().GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Generate new API key
	apiKey, hashedKey, prefix, err := s.generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Update user's API key
	if err := s.store.Users().UpdateAPIKey(ctx, userID, hashedKey, prefix); err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	return &domain.APIKeyResponse{
		APIKey: apiKey,
	}, nil
}

// generateAPIKey generates a new API key and returns the raw key, hashed key, and prefix.
func (s *UserService) generateAPIKey() (raw, hashed, prefix string, err error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", err
	}

	// Create API key with prefix
	raw = "em_" + hex.EncodeToString(bytes)
	prefix = raw[:10] + "..."
	hashed = s.hashAPIKey(raw)

	return raw, hashed, prefix, nil
}

// hashAPIKey creates a SHA-256 hash of an API key.
func (s *UserService) hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
