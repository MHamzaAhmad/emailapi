package domain

import (
	"time"
)

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
)

// LogEventType represents the type of event being logged.
type LogEventType string

const (
	LogEventSent      LogEventType = "sent"
	LogEventDelivered LogEventType = "delivered"
	LogEventOpened    LogEventType = "opened"
	LogEventClicked   LogEventType = "clicked"
	LogEventBounced   LogEventType = "bounced"
	LogEventFailed    LogEventType = "failed"
)

// Log represents an email event log entry.
// Stored in ClickHouse for high-volume analytics.
type Log struct {
	ID        string       `json:"id"`
	EmailID   string       `json:"email_id"`
	UserID    string       `json:"user_id"`
	EventType LogEventType `json:"event_type"`
	Level     LogLevel     `json:"level"`
	Message   string       `json:"message,omitempty"`
	Metadata  Metadata     `json:"metadata,omitempty"`
	IPAddress string       `json:"ip_address,omitempty"`
	UserAgent string       `json:"user_agent,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

// LogFilter represents filters for querying logs.
type LogFilter struct {
	EmailID   string       `json:"email_id,omitempty"`
	UserID    string       `json:"user_id,omitempty"`
	EventType LogEventType `json:"event_type,omitempty"`
	Level     LogLevel     `json:"level,omitempty"`
	StartTime *time.Time   `json:"start_time,omitempty"`
	EndTime   *time.Time   `json:"end_time,omitempty"`
	Limit     int          `json:"limit,omitempty"`
	Offset    int          `json:"offset,omitempty"`
}
