package tinybird

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EmailRouting represents a routing table entry for reply lookups.
type EmailRouting struct {
	MessageID string
	EmailID   string
	UserID    string
	SentAt    time.Time
}

// EmailRepository handles email routing in Tinybird.
type EmailRepository struct {
	client       *Client
	activityRepo *ActivityRepository
}

// NewEmailRepository creates a new EmailRepository.
func NewEmailRepository(client *Client) *EmailRepository {
	return &EmailRepository{
		client:       client,
		activityRepo: NewActivityRepository(client),
	}
}

// InsertRouting adds an entry to the email_routing datasource.
// This is called when an email is successfully sent.
func (r *EmailRepository) InsertRouting(ctx context.Context, messageID, emailID, userID string) error {
	event := map[string]interface{}{
		"message_id": messageID,
		"email_id":   emailID,
		"user_id":    userID,
		"sent_at":    time.Now().Format(time.RFC3339),
	}

	if err := r.client.IngestEvent(ctx, "email_routing", event); err != nil {
		return fmt.Errorf("failed to insert routing entry: %w", err)
	}

	return nil
}

// LookupRouting finds the routing info for a given message_id.
// This is the single source of truth for reply routing.
func (r *EmailRepository) LookupRouting(ctx context.Context, messageID string) (*EmailRouting, error) {
	// Remove domain part if present (e.g., "abc123@amazonses.com" -> "abc123")
	msgId := strings.Split(messageID, "@")[0]

	params := map[string]string{
		"message_id": msgId,
	}

	resp, err := r.client.QueryPipe(ctx, "lookup_routing", params)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup routing: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("routing not found for message ID %s", messageID)
	}

	row := resp.Data[0]

	// Parse sent_at timestamp
	sentAtStr, _ := row["sent_at"].(string)
	sentAt, _ := time.Parse(time.RFC3339, sentAtStr)

	return &EmailRouting{
		MessageID: row["message_id"].(string),
		EmailID:   row["email_id"].(string),
		UserID:    row["user_id"].(string),
		SentAt:    sentAt,
	}, nil
}

// LookupRoutingByEmailID finds the routing info for a given email_id.
// Used for reply threading to resolve email_id → message_id.
func (r *EmailRepository) LookupRoutingByEmailID(ctx context.Context, emailID string) (*EmailRouting, error) {
	params := map[string]string{
		"email_id": emailID,
	}

	resp, err := r.client.QueryPipe(ctx, "lookup_routing_by_email_id", params)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup routing by email_id: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("routing not found for email ID %s", emailID)
	}

	row := resp.Data[0]

	sentAtStr, _ := row["sent_at"].(string)
	sentAt, _ := time.Parse(time.RFC3339, sentAtStr)

	return &EmailRouting{
		MessageID: row["message_id"].(string),
		EmailID:   emailID,
		UserID:    row["user_id"].(string),
		SentAt:    sentAt,
	}, nil
}

// LogActivity logs an activity event to activity_logs datasource.
func (r *EmailRepository) LogActivity(
	ctx context.Context,
	userID, entityType, entityID, action, status, details string,
	metadata map[string]interface{},
) error {
	return r.activityRepo.Log(ctx, userID, entityType, entityID, action, status, details, metadata)
}

// LogEmailEvent logs an email-specific event to activity_logs.
func (r *EmailRepository) LogEmailEvent(
	ctx context.Context,
	userID, emailID, action, status, details string,
	metadata map[string]interface{},
) error {
	return r.activityRepo.Log(ctx, userID, "email", emailID, action, status, details, metadata)
}

// Ping ensures the Tinybird connection is valid.
func (r *EmailRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx)
}

// generateID creates a UUID for activity log entries (matching ClickHouse behavior).
func generateID() string {
	return uuid.New().String()
}
