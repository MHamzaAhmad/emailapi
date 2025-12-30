package domain

import "time"

// ReputationIncident represents a single bounce or complaint incident.
type ReputationIncident struct {
	ID                    string    `json:"id"`
	UserID                string    `json:"user_id"`
	IncidentType          string    `json:"incident_type"` // bounce_hard, bounce_soft, complaint
	MessageID             string    `json:"message_id"`
	RecipientEmailHash    string    `json:"recipient_email_hash"`
	BounceType            string    `json:"bounce_type,omitempty"`
	BounceSubtype         string    `json:"bounce_subtype,omitempty"`
	ComplaintFeedbackType string    `json:"complaint_feedback_type,omitempty"`
	DiagnosticCode        string    `json:"diagnostic_code,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

// IncidentType constants
const (
	IncidentTypeBounceHard = "bounce_hard"
	IncidentTypeBounceSoft = "bounce_soft"
	IncidentTypeComplaint  = "complaint"
)

// UserReputation represents a user's reputation stats.
type UserReputation struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	TotalBounces     int        `json:"total_bounces"`
	HardBounces      int        `json:"hard_bounces"`
	SoftBounces      int        `json:"soft_bounces"`
	Complaints       int        `json:"complaints"`
	Bounces30d       int        `json:"bounces_30d"`
	Complaints30d    int        `json:"complaints_30d"`
	SuspensionScore  float64    `json:"suspension_score"`
	IsFlagged        bool       `json:"is_flagged"`
	FlaggedAt        *time.Time `json:"flagged_at,omitempty"`
	FlaggedReason    string     `json:"flagged_reason,omitempty"`
	IsSuspended      bool       `json:"is_suspended"`
	SuspendedAt      *time.Time `json:"suspended_at,omitempty"`
	SuspendedBy      string     `json:"suspended_by,omitempty"`
	SuspensionReason string     `json:"suspension_reason,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	// Optional user info (populated from joins)
	UserEmail string `json:"user_email,omitempty"`
	UserName  string `json:"user_name,omitempty"`
}

// IncidentStats holds aggregated incident counts.
type IncidentStats struct {
	HardBounces   int `json:"hard_bounces"`
	SoftBounces   int `json:"soft_bounces"`
	Complaints    int `json:"complaints"`
	Bounces30d    int `json:"bounces_30d"`
	Complaints30d int `json:"complaints_30d"`
}

// BounceRecipient represents a bounced email recipient.
type BounceRecipient struct {
	EmailAddress   string `json:"email_address"`
	DiagnosticCode string `json:"diagnostic_code,omitempty"`
}
