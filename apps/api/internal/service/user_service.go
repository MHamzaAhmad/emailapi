package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/emailapi/api/internal/domain"
)

// UserService handles user business logic.
type UserService struct {
	store  Store
	apiKey *APIKeyService
}

// NewUserService creates a new UserService.
func NewUserService(store Store, apiKey *APIKeyService) *UserService {
	return &UserService{
		store:  store,
		apiKey: apiKey,
	}
}

// Create creates a new user.
func (s *UserService) Create(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, string, error) {
	// Check if user already exists
	existing, _ := s.store.Users().GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, "", fmt.Errorf("user with email already exists")
	}

	// Set default role
	role := req.Role
	if role == "" {
		role = domain.UserRoleMember
	}

	user := &domain.User{
		ID:       uuid.New().String(),
		Email:    req.Email,
		Name:     req.Name,
		Role:     role,
		IsActive: true,
	}

	if err := s.store.Users().Create(ctx, user); err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	// Automatically create an API key for the new user
	createKeyReq := &domain.CreateAPIKeyRequest{
		Name:        "Default Key",
		Scopes:      []domain.Scope{domain.ScopeEmailSend, domain.ScopeEmailRead, domain.ScopeDomainRead, domain.ScopeDomainWrite, domain.ScopeApiKeyRead, domain.ScopeApiKeyWrite, domain.ScopeUserRead, domain.ScopeUserWrite},
		Environment: domain.EnvLive,
	}

	_, rawKey, err := s.apiKey.Create(ctx, user.ID, createKeyReq)
	if err != nil {
		// Log error but don't fail user creation?
		// For now, let's fail it or just return empty key if it fails, but better to fail so user knows something went wrong.
		// Since we just created the user, ideally we should rollback, but for this simple implementation we'll just error out.
		return nil, "", fmt.Errorf("failed to create initial api key: %w", err)
	}

	return user, rawKey, nil
}

// GetByID retrieves a user by ID.
func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.store.Users().GetByID(ctx, id)
}

// GetByEmail retrieves a user by email.
func (s *UserService) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.store.Users().GetByEmail(ctx, email)
}

// Update updates a user.
func (s *UserService) Update(ctx context.Context, id string, req *domain.UpdateUserRequest) (*domain.User, error) {
	user, err := s.store.Users().GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := s.store.Users().Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// Delete deletes a user.
func (s *UserService) Delete(ctx context.Context, id string) error {
	return s.store.Users().Delete(ctx, id)
}

// List retrieves users with pagination.
func (s *UserService) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.store.Users().List(ctx, limit, offset)
}
