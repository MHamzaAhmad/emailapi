package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/emailapi/api/internal/domain"
)

type EmailRepository struct {
	conn driver.Conn
}

func NewEmailRepository(conn driver.Conn) *EmailRepository {
	return &EmailRepository{conn: conn}
}

// AddEmailEvent inserts a new email event log into ClickHouse.
func (r *EmailRepository) AddEmailEvent(ctx context.Context, email *domain.Email, eventType string) error {
	metadataJSON, err := json.Marshal(email.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO emails (
			email_id, message_id, user_id, event_type, level, message, metadata, timestamp
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?
		)
	`

	level := "info"
	if eventType == "failed" || eventType == "bounced" {
		level = "error"
	}

	err = r.conn.Exec(ctx, query,
		email.ID,
		email.MessageID,
		email.UserID,
		eventType,
		level,
		fmt.Sprintf("Email %s", eventType),
		string(metadataJSON),
		time.Now(),
	)

	return err
}

// ArchiveEmail archives a completed email from PostgreSQL to ClickHouse.
// This is called after an email reaches a terminal state (sent, failed, bounced).
func (r *EmailRepository) ArchiveEmail(ctx context.Context, email *domain.Email) error {
	metadataJSON, err := json.Marshal(email.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO email_archive (
			id, user_id, from_address, to_addresses, cc_addresses, bcc_addresses,
			subject, body, html, status, provider_id, attachment_count, metadata,
			error_message, scheduled_at, sent_at, created_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`

	var scheduledAt, sentAt *time.Time
	if email.ScheduledAt != nil {
		scheduledAt = email.ScheduledAt
	}
	if email.SentAt != nil {
		sentAt = email.SentAt
	}

	err = r.conn.Exec(ctx, query,
		email.ID,
		email.UserID,
		email.From,
		email.To,
		nullableStringSlice(email.Cc),
		nullableStringSlice(email.Bcc),
		email.Subject,
		email.Body,
		email.HTML,
		string(email.Status),
		email.ProviderID,
		uint8(len(email.Attachments)),
		string(metadataJSON),
		email.ErrorMessage,
		scheduledAt,
		sentAt,
		email.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to archive email: %w", err)
	}

	return nil
}

// GetEmailEvents retrieves event history for an email.
func (r *EmailRepository) GetEmailEvents(ctx context.Context, emailID string) ([]*domain.EmailEvent, error) {
	query := `
		SELECT email_id, message_id, user_id, event_type, level, message, metadata, timestamp
		FROM emails
		WHERE email_id = ?
		ORDER BY timestamp DESC
	`

	rows, err := r.conn.Query(ctx, query, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to query email events: %w", err)
	}
	defer rows.Close()

	var events []*domain.EmailEvent
	for rows.Next() {
		var event domain.EmailEvent
		var eventType string
		var level string
		if err := rows.Scan(
			&event.EmailID,
			&event.MessageID,
			&event.UserID,
			&eventType,
			&level,
			&event.Message,
			&event.Metadata,
			&event.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan email event: %w", err)
		}
		event.EventType = eventType
		event.Level = level
		events = append(events, &event)
	}

	return events, nil
}

// GetUserEmailStats retrieves aggregated stats for a user over the last N days.
func (r *EmailRepository) GetUserEmailStats(ctx context.Context, userID string, days int) (*domain.EmailStats, error) {
	query := `
		SELECT 
			countIf(event_type = 'sent') as total_sent,
			countIf(event_type = 'delivered') as total_delivered,
			countIf(event_type = 'bounced') as total_bounced,
			countIf(event_type = 'failed') as total_failed
		FROM emails
		WHERE user_id = ? AND date >= today() - ?
	`

	row := r.conn.QueryRow(ctx, query, userID, days)

	var stats domain.EmailStats
	if err := row.Scan(
		&stats.TotalSent,
		&stats.TotalDelivered,
		&stats.TotalBounced,
		&stats.TotalFailed,
	); err != nil {
		return nil, fmt.Errorf("failed to get user email stats: %w", err)
	}

	return &stats, nil
}

// Ping ensures the database connection is valid.
func (r *EmailRepository) Ping(ctx context.Context) error {
	return r.conn.Ping(ctx)
}

// nullableStringSlice converts a slice to a format suitable for ClickHouse Array.
func nullableStringSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	// Clean up any empty strings
	var result []string
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			result = append(result, v)
		}
	}
	if result == nil {
		return []string{}
	}
	return result
}
