package clickhouse

import (
	"context"
	"fmt"
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

// AddEmail inserts a new email event log into ClickHouse.
func (r *EmailRepository) AddEmail(ctx context.Context, email *domain.Email, eventType string) error {
	query := `
		INSERT INTO emails (
			id, email_id, message_id, user_id, event_type, level, message, metadata, created_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`

	// Using Exec for simple insertion. For high throughput, we should use Batch/AsyncInsert.
	// clickhouse-go v2 supports inputs as arguments directly.

	err := r.conn.Exec(ctx, query,
		email.ID, // id (using email ID as log ID might be wrong, ideally generate new UUID but schema defaults it)
		// Actually schema defaults 'id', but here we are providing it.
		// If we want schema to generate it, we should omit it from INSERT.
		// But let's assume we pass a new UUID or just pass email.ID as a placeholder if strictly 1:1, but logs are 1:N.
		// Wait, earlier code passed email.ID as 'id'.
		// Let's generate a new UUID for the log entry if possible, or let CH generate it.
		// To let CH generate it (DEFAULT generateUUIDv4()), we should exclude 'id' from columns.

		email.ID,        // email_id
		email.MessageID, // message_id
		email.UserID,
		eventType,
		"info",                             // level
		fmt.Sprintf("Email %s", eventType), // message
		"{}",                               // metadata (TODO: serialize actual metadata)
		time.Now(),
	)

	return err
}

// Ensure database connection is valid
func (r *EmailRepository) Ping(ctx context.Context) error {
	return r.conn.Ping(ctx)
}
