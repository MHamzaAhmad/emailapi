package clickhouse

//go:generate mockgen -destination=mocks/mock_clickhouse.go -package=mocks github.com/emailapi/api/internal/repository/clickhouse EmailRepositoryInterface,ActivityRepositoryInterface

import (
	"context"
)

// EmailRepositoryInterface defines the interface for ClickHouse email operations.
type EmailRepositoryInterface interface {
	// InsertRouting adds an entry to the email_routing table.
	InsertRouting(ctx context.Context, messageID, emailID, userID string) error

	// LookupRouting finds the routing info for a given message_id.
	LookupRouting(ctx context.Context, messageID string) (*EmailRouting, error)

	// LogActivity logs an activity event to activity_logs table.
	LogActivity(ctx context.Context, userID, entityType, entityID, action, status, details string, metadata map[string]interface{}) error

	// LogEmailEvent logs an email-specific event to activity_logs.
	LogEmailEvent(ctx context.Context, userID, emailID, action, status, details string, metadata map[string]interface{}) error

	// Ping ensures the database connection is valid.
	Ping(ctx context.Context) error
}

// ActivityFilters defines optional filters for querying activity logs.
type ActivityFilters struct {
	EntityType string // Optional: filter by entity type (email, domain, api_key)
	Action     string // Optional: filter by action
	StartTime  *int64 // Optional: Unix timestamp for start range
	EndTime    *int64 // Optional: Unix timestamp for end range
}

// ActivityLog represents a single activity log entry from ClickHouse.
type ActivityLog struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Action     string                 `json:"action"`
	Status     string                 `json:"status"`
	Details    string                 `json:"details"`
	Metadata   map[string]interface{} `json:"metadata"`
	Timestamp  int64                  `json:"timestamp"` // Unix milliseconds
}

// ActivityRepositoryInterface defines the interface for ClickHouse activity operations.
type ActivityRepositoryInterface interface {
	// Log inserts a generic activity log entry into ClickHouse.
	Log(ctx context.Context, userID, entityType, entityID, action, status, details string, metadata map[string]interface{}) error

	// LogEmail is a convenience method for logging email-related activities.
	LogEmail(ctx context.Context, userID, emailID, action, status, details string, metadata map[string]interface{}) error

	// LogDomain is a convenience method for logging domain-related activities.
	LogDomain(ctx context.Context, userID, domainID, action, status, details string) error

	// LogAPIKey is a convenience method for logging API key-related activities.
	LogAPIKey(ctx context.Context, userID, apiKeyID, action, status, details string) error

	// List retrieves activity logs with pagination and optional filters.
	// Returns logs, total count, and error.
	List(ctx context.Context, userID string, filters ActivityFilters, limit, offset int) ([]ActivityLog, int, error)

	// Ping ensures the database connection is valid.
	Ping(ctx context.Context) error
}

// Ensure concrete types implement interfaces
var _ EmailRepositoryInterface = (*EmailRepository)(nil)
var _ ActivityRepositoryInterface = (*ActivityRepository)(nil)
