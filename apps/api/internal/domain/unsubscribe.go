package domain

import "time"

// UnsubscribeSource indicates how the unsubscribe was triggered.
type UnsubscribeSource string

const (
	// UnsubscribeSourceLink is from web form (user clicked link and confirmed).
	UnsubscribeSourceLink UnsubscribeSource = "link"
	// UnsubscribeSourceOneClick is from RFC 8058 one-click header (email client button).
	UnsubscribeSourceOneClick UnsubscribeSource = "one_click"
	// UnsubscribeSourceManual is from admin/API action.
	UnsubscribeSourceManual UnsubscribeSource = "manual"
)

// UnsubscribeEntry represents an unsubscribed recipient for a user.
type UnsubscribeEntry struct {
	ID            string
	UserID        string
	EmailHash     string // SHA-256 hash of lowercased recipient email
	SourceEmailID string // Email that triggered unsubscribe (optional)
	Source        UnsubscribeSource
	CreatedAt     time.Time
}
