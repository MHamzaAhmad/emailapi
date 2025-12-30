package service

import (
	"context"
	"fmt"

	"github.com/emailapi/api/internal/domain"
)

// AdminService handles admin-only user management operations.
type AdminService struct {
	store      Store
	user       *UserService
	reputation *ReputationService
}

// NewAdminService creates a new AdminService.
func NewAdminService(store Store, user *UserService, reputation *ReputationService) *AdminService {
	return &AdminService{
		store:      store,
		user:       user,
		reputation: reputation,
	}
}

// ListUsers returns a paginated list of all users with their reputation status.
func (s *AdminService) ListUsers(ctx context.Context, limit, offset int) ([]*domain.AdminUser, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	users, err := s.store.Users().ListWithReputation(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	total, err := s.store.Users().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	return users, total, nil
}

// ListFlaggedUsers returns users flagged for review due to reputation issues.
func (s *AdminService) ListFlaggedUsers(ctx context.Context, limit, offset int) ([]*domain.UserReputation, int, error) {
	return s.reputation.ListFlaggedUsers(ctx, limit, offset)
}

// GetUserDetails returns full user details including reputation stats.
func (s *AdminService) GetUserDetails(ctx context.Context, userID string) (*domain.UserWithReputation, error) {
	result, err := s.store.Users().GetWithReputation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return result, nil
}

// SuspendUser suspends a user account.
func (s *AdminService) SuspendUser(ctx context.Context, userID, adminID, reason string) error {
	// Verify target user exists
	_, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	return s.reputation.SuspendUser(ctx, userID, adminID, reason)
}

// UnsuspendUser reinstates a suspended user account.
func (s *AdminService) UnsuspendUser(ctx context.Context, userID, adminID string) error {
	// Verify target user exists
	_, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	return s.reputation.UnsuspendUser(ctx, userID, adminID)
}
