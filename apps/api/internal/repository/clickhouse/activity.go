package clickhouse

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ActivityRepository handles generic activity logging to ClickHouse.
// This is for all system events: email events, domain changes, API key operations, etc.
type ActivityRepository struct {
	conn driver.Conn
}

// NewActivityRepository creates a new ActivityRepository.
func NewActivityRepository(conn driver.Conn) *ActivityRepository {
	return &ActivityRepository{conn: conn}
}

// Log inserts a generic activity log entry into ClickHouse.
func (r *ActivityRepository) Log(ctx context.Context, userID, entityType, entityID, action, status, details string, metadata map[string]interface{}) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO activity_logs (
			user_id, entity_type, entity_id, action, status, details, metadata, timestamp
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?
		)
	`

	return r.conn.Exec(ctx, query,
		userID,
		entityType,
		entityID,
		action,
		status,
		details,
		string(metadataJSON),
		time.Now(),
	)
}

// LogEmail is a convenience method for logging email-related activities.
func (r *ActivityRepository) LogEmail(ctx context.Context, userID, emailID, action, status, details string, metadata map[string]interface{}) error {
	return r.Log(ctx, userID, "email", emailID, action, status, details, metadata)
}

// LogDomain is a convenience method for logging domain-related activities.
func (r *ActivityRepository) LogDomain(ctx context.Context, userID, domainID, action, status, details string) error {
	return r.Log(ctx, userID, "domain", domainID, action, status, details, nil)
}

// LogAPIKey is a convenience method for logging API key-related activities.
func (r *ActivityRepository) LogAPIKey(ctx context.Context, userID, apiKeyID, action, status, details string) error {
	return r.Log(ctx, userID, "api_key", apiKeyID, action, status, details, nil)
}

// Ping ensures the database connection is valid.
func (r *ActivityRepository) Ping(ctx context.Context) error {
	return r.conn.Ping(ctx)
}
