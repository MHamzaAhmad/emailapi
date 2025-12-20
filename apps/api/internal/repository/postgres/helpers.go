package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/emailapi/api/internal/domain"
)

// toPgText converts a string to pgtype.Text.
func toPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// toPgTextPtr converts a *string to pgtype.Text.
func toPgTextPtr(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// fromPgText converts pgtype.Text to string.
func fromPgText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// fromPgTextPtr converts pgtype.Text to *string.
func fromPgTextPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// toPgTimestamp converts *time.Time to pgtype.Timestamptz.
func toPgTimestamp(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// toPgTimestampNow returns the current time as pgtype.Timestamptz.
func toPgTimestampNow() pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now(), Valid: true}
}

// fromPgTimestamp converts pgtype.Timestamptz to *time.Time.
func fromPgTimestamp(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// toPgInt4 converts *int to pgtype.Int4.
func toPgInt4(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

// fromPgInt4 converts pgtype.Int4 to *int.
func fromPgInt4(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

// fromPgInt4Value converts pgtype.Int4 to int (0 if null).
func fromPgInt4Value(i pgtype.Int4) int {
	if !i.Valid {
		return 0
	}
	return int(i.Int32)
}

// toPgInt4Value converts int to pgtype.Int4.
func toPgInt4Value(i int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(i), Valid: true}
}

// fromPgTextValue converts pgtype.Text to string (empty if null).
func fromPgTextValue(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// toMetadata converts map[string]string to domain.Metadata.
func toMetadata(m map[string]string) domain.Metadata {
	if m == nil {
		return nil
	}
	result := make(domain.Metadata, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// toPgTimestampFromTime converts time.Time to pgtype.Timestamptz.
func toPgTimestampFromTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
