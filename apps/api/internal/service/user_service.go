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
	cache  Cache
}

// NewUserService creates a new UserService.
func NewUserService(store Store, apiKey *APIKeyService, cache Cache) *UserService {
	return &UserService{
		store:  store,
		apiKey: apiKey,
		cache:  cache,
	}
}

// Create creates a new user.
func (s *UserService) Create(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, error) {
	// Check if user already exists by email
	existing, _ := s.store.Users().GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, fmt.Errorf("user with email already exists")
	}

	// Check if user exists by external_id (Clerk ID)
	if req.ExternalID != nil && *req.ExternalID != "" {
		existingExt, _ := s.store.Users().GetByExternalID(ctx, *req.ExternalID)
		if existingExt != nil {
			return nil, fmt.Errorf("user with external_id already exists")
		}
	}

	// Set default role
	role := req.Role
	if role == "" {
		role = domain.UserRoleMember
	}

	user := &domain.User{
		ID:         uuid.New().String(),
		Email:      req.Email,
		Name:       req.Name,
		Role:       role,
		IsActive:   true,
		ExternalID: req.ExternalID,
	}

	if err := s.store.Users().Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetByID retrieves a user by ID.
func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.store.Users().GetByID(ctx, id)
}

// GetByEmail retrieves a user by email.
func (s *UserService) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.store.Users().GetByEmail(ctx, email)
}

// GetByExternalID retrieves a user by their Clerk external ID.
// Returns the internal user ID and error.
// Uses cache-first approach for performance.
func (s *UserService) GetByExternalID(ctx context.Context, externalID string) (string, error) {
	// Try cache first
	if s.cache != nil {
		if cached, _ := s.cache.User().GetByExternalID(ctx, externalID); cached != nil {
			return cached.ID, nil
		}
	}

	// Cache miss - query DB
	user, err := s.store.Users().GetByExternalID(ctx, externalID)
	if err != nil {
		return "", err
	}

	// Cache the result
	if s.cache != nil {
		_ = s.cache.User().SetByExternalID(ctx, user)
	}

	return user.ID, nil
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
	if req.ExternalID != nil {
		user.ExternalID = req.ExternalID
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
