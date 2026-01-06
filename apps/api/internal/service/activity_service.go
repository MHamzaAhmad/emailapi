package service

import (
	"context"

	tbrepo "github.com/emailapi/api/internal/repository/tinybird"
)

// ActivityService handles activity log business logic.
type ActivityService struct {
	analytics Analytics
}

// NewActivityService creates a new ActivityService.
func NewActivityService(analytics Analytics) *ActivityService {
	return &ActivityService{
		analytics: analytics,
	}
}

// ListFilters defines optional filters for listing activity logs.
type ListFilters struct {
	EntityType *string
	Action     *string
	StartTime  *int64
	EndTime    *int64
}

// List retrieves activity logs for a user with pagination and filters.
// userID is extracted from authenticated context in the transport layer.
func (s *ActivityService) List(ctx context.Context, userID string, filters ListFilters, limit, offset int) ([]tbrepo.ActivityLog, int, error) {
	// Apply default limits
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	repoFilters := tbrepo.ActivityFilters{}
	if filters.EntityType != nil {
		repoFilters.EntityType = *filters.EntityType
	}
	if filters.Action != nil {
		repoFilters.Action = *filters.Action
	}
	if filters.StartTime != nil {
		repoFilters.StartTime = filters.StartTime
	}
	if filters.EndTime != nil {
		repoFilters.EndTime = filters.EndTime
	}

	if s.analytics == nil {
		return nil, 0, nil
	}

	return s.analytics.Activity().List(ctx, userID, repoFilters, limit, offset)
}
