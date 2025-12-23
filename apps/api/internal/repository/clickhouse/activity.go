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

// List retrieves activity logs with pagination and optional filters.
func (r *ActivityRepository) List(ctx context.Context, userID string, filters ActivityFilters, limit, offset int) ([]ActivityLog, int, error) {
	// Build query with optional filters
	baseQuery := `
		SELECT 
			toString(id),
			user_id,
			entity_type,
			entity_id,
			action,
			status,
			details,
			metadata,
			toUnixTimestamp64Milli(timestamp)
		FROM activity_logs
		WHERE user_id = ?
	`
	countQuery := `SELECT count(*) FROM activity_logs WHERE user_id = ?`

	args := []interface{}{userID}
	countArgs := []interface{}{userID}

	// Apply optional filters
	if filters.EntityType != "" {
		baseQuery += " AND entity_type = ?"
		countQuery += " AND entity_type = ?"
		args = append(args, filters.EntityType)
		countArgs = append(countArgs, filters.EntityType)
	}

	if filters.Action != "" {
		baseQuery += " AND action = ?"
		countQuery += " AND action = ?"
		args = append(args, filters.Action)
		countArgs = append(countArgs, filters.Action)
	}

	if filters.StartTime != nil {
		baseQuery += " AND timestamp >= fromUnixTimestamp64Milli(?)"
		countQuery += " AND timestamp >= fromUnixTimestamp64Milli(?)"
		args = append(args, *filters.StartTime)
		countArgs = append(countArgs, *filters.StartTime)
	}

	if filters.EndTime != nil {
		baseQuery += " AND timestamp <= fromUnixTimestamp64Milli(?)"
		countQuery += " AND timestamp <= fromUnixTimestamp64Milli(?)"
		args = append(args, *filters.EndTime)
		countArgs = append(countArgs, *filters.EndTime)
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	// Get total count first
	var totalCount int
	row := r.conn.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Execute main query
	rows, err := r.conn.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []ActivityLog
	for rows.Next() {
		var log ActivityLog
		var metadataStr string
		if err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.EntityType,
			&log.EntityID,
			&log.Action,
			&log.Status,
			&log.Details,
			&metadataStr,
			&log.Timestamp,
		); err != nil {
			return nil, 0, err
		}

		// Parse metadata JSON
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &log.Metadata); err != nil {
				log.Metadata = make(map[string]interface{})
			}
		} else {
			log.Metadata = make(map[string]interface{})
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, totalCount, nil
}
