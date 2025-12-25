package tinybird

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// ActivityFilters defines optional filters for querying activity logs.
type ActivityFilters struct {
	EntityType string // Optional: filter by entity type (email, domain, api_key)
	Action     string // Optional: filter by action
	StartTime  *int64 // Optional: Unix timestamp for start range
	EndTime    *int64 // Optional: Unix timestamp for end range
}

// ActivityLog represents a single activity log entry.
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

// ActivityRepository handles generic activity logging to Tinybird.
type ActivityRepository struct {
	client *Client
}

// NewActivityRepository creates a new ActivityRepository.
func NewActivityRepository(client *Client) *ActivityRepository {
	return &ActivityRepository{client: client}
}

// Log inserts a generic activity log entry into Tinybird.
func (r *ActivityRepository) Log(ctx context.Context, userID, entityType, entityID, action, status, details string, metadata map[string]interface{}) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	event := map[string]interface{}{
		"id":          uuid.New().String(),
		"user_id":     userID,
		"entity_type": entityType,
		"entity_id":   entityID,
		"action":      action,
		"status":      status,
		"details":     details,
		"metadata":    string(metadataJSON),
		"timestamp":   time.Now().Format("2006-01-02 15:04:05.000"),
	}

	if err := r.client.IngestEvent(ctx, "activity_logs", event); err != nil {
		return fmt.Errorf("failed to log activity: %w", err)
	}

	return nil
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

// Ping ensures the Tinybird connection is valid.
func (r *ActivityRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx)
}

// List retrieves activity logs with pagination and optional filters.
func (r *ActivityRepository) List(ctx context.Context, userID string, filters ActivityFilters, limit, offset int) ([]ActivityLog, int, error) {
	params := map[string]string{
		"user_id": userID,
		"limit":   strconv.Itoa(limit),
		"offset":  strconv.Itoa(offset),
	}

	if filters.EntityType != "" {
		params["entity_type"] = filters.EntityType
	}
	if filters.Action != "" {
		params["action"] = filters.Action
	}
	if filters.StartTime != nil {
		params["start_time"] = strconv.FormatInt(*filters.StartTime, 10)
	}
	if filters.EndTime != nil {
		params["end_time"] = strconv.FormatInt(*filters.EndTime, 10)
	}

	// Query list endpoint
	listResp, err := r.client.QueryPipe(ctx, "list_activity", params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list activity: %w", err)
	}

	// Query count endpoint (same pipe, different node - accessed via query param)
	countParams := make(map[string]string)
	for k, v := range params {
		countParams[k] = v
	}
	// Note: Tinybird uses node query param to select which node to execute
	// The count node returns total count
	countResp, err := r.client.QueryPipe(ctx, "list_activity__count", params)
	if err != nil {
		// If count fails, we can still return results with unknown total
		countResp = &PipeResponse{Data: []map[string]interface{}{{"total": float64(0)}}}
	}

	// Parse list results
	var logs []ActivityLog
	for _, row := range listResp.Data {
		log := ActivityLog{
			ID:         getString(row, "id"),
			UserID:     getString(row, "user_id"),
			EntityType: getString(row, "entity_type"),
			EntityID:   getString(row, "entity_id"),
			Action:     getString(row, "action"),
			Status:     getString(row, "status"),
			Details:    getString(row, "details"),
			Timestamp:  getInt64(row, "timestamp"),
		}

		// Parse metadata JSON
		metadataStr := getString(row, "metadata")
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &log.Metadata); err != nil {
				log.Metadata = make(map[string]interface{})
			}
		} else {
			log.Metadata = make(map[string]interface{})
		}

		logs = append(logs, log)
	}

	// Parse total count
	totalCount := 0
	if len(countResp.Data) > 0 {
		if total, ok := countResp.Data[0]["total"].(float64); ok {
			totalCount = int(total)
		}
	}

	return logs, totalCount, nil
}

// getString safely extracts a string from the row map.
func getString(row map[string]interface{}, key string) string {
	if v, ok := row[key].(string); ok {
		return v
	}
	return ""
}

// getInt64 safely extracts an int64 from the row map.
func getInt64(row map[string]interface{}, key string) int64 {
	switch v := row[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}
