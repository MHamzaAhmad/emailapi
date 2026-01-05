package domain

import "time"

// AdminUser represents a user with suspension/flag status for admin listing.
type AdminUser struct {
	*User
	IsSuspended bool `json:"is_suspended"`
	IsFlagged   bool `json:"is_flagged"`
}

// UserWithReputation represents a user with full reputation data.
type UserWithReputation struct {
	// User fields
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	Name            string    `json:"name"`
	Role            UserRole  `json:"role"`
	IsActive        bool      `json:"is_active"`
	ExternalID      *string   `json:"external_id,omitempty"`
	Plan            UserPlan  `json:"plan"`
	PolarCustomerID *string   `json:"polar_customer_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Reputation fields
	TotalBounces     int        `json:"total_bounces"`
	HardBounces      int        `json:"hard_bounces"`
	SoftBounces      int        `json:"soft_bounces"`
	Complaints       int        `json:"complaints"`
	Bounces30d       int        `json:"bounces_30d"`
	Complaints30d    int        `json:"complaints_30d"`
	SuspensionScore  float64    `json:"suspension_score"`
	IsFlagged        bool       `json:"is_flagged"`
	FlaggedAt        *time.Time `json:"flagged_at,omitempty"`
	FlaggedReason    *string    `json:"flagged_reason,omitempty"`
	IsSuspended      bool       `json:"is_suspended"`
	SuspendedAt      *time.Time `json:"suspended_at,omitempty"`
	SuspendedBy      *string    `json:"suspended_by,omitempty"`
	SuspensionReason *string    `json:"suspension_reason,omitempty"`
}
