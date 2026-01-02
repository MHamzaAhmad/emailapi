package service

//go:generate mockgen -destination=mocks/mock_analytics.go -package=mocks github.com/emailapi/api/internal/service Analytics

import (
	tbrepo "github.com/emailapi/api/internal/repository/tinybird"
)

// Analytics aggregates all Tinybird repository access.
// This is injected into services for testability.
type Analytics interface {
	Email() tbrepo.EmailRepositoryInterface
	Activity() tbrepo.ActivityRepositoryInterface
}
