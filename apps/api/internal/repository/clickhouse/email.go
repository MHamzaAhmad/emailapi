package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// EmailRepository handles email routing and activity logging in ClickHouse.
type EmailRepository struct {
	conn         driver.Conn
	activityRepo *ActivityRepository
}

// NewEmailRepository creates a new EmailRepository.
func NewEmailRepository(conn driver.Conn) *EmailRepository {
	return &EmailRepository{
		conn:         conn,
		activityRepo: NewActivityRepository(conn),
	}
}

// EmailRouting represents a routing table entry for reply lookups.
type EmailRouting struct {
	MessageID string
	EmailID   string
	UserID    string
	SentAt    time.Time
}

// InsertRouting adds an entry to the email_routing table.
// This is called when an email is successfully sent.
func (r *EmailRepository) InsertRouting(ctx context.Context, messageID, emailID, userID string) error {
	query := `
		INSERT INTO email_routing (message_id, email_id, user_id, sent_at)
		VALUES (?, ?, ?, ?)
	`

	err := r.conn.Exec(ctx, query, messageID, emailID, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert routing entry: %w", err)
	}

	return nil
}

// LookupRouting finds the routing info for a given message_id.
// This is the single source of truth for reply routing.
func (r *EmailRepository) LookupRouting(ctx context.Context, messageID string) (*EmailRouting, error) {
	query := `
		SELECT message_id, email_id, user_id, sent_at
		FROM email_routing
		WHERE message_id = ?
		LIMIT 1
	`

	// Remove domain part if present (e.g., "abc123@amazonses.com" -> "abc123")
	msgId := strings.Split(messageID, "@")[0]
	row := r.conn.QueryRow(ctx, query, msgId)

	var routing EmailRouting
	if err := row.Scan(
		&routing.MessageID,
		&routing.EmailID,
		&routing.UserID,
		&routing.SentAt,
	); err != nil {
		return nil, fmt.Errorf("routing not found for message ID %s: %w", messageID, err)
	}

	return &routing, nil
}

// LogActivity logs an activity event to activity_logs table.
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

// Ping ensures the database connection is valid.
func (r *EmailRepository) Ping(ctx context.Context) error {
	return r.conn.Ping(ctx)
}
