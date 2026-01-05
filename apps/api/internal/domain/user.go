package domain

import (
	"time"
)

// UserRole represents a user's role in the system.
type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
	UserRoleMember UserRole = "member"
)

// UserPlan represents a user's subscription plan.
type UserPlan string

const (
	UserPlanFree  UserPlan = "free"
	UserPlanScale UserPlan = "scale"
	UserPlanPAYG  UserPlan = "payg"
)

// PlanConfig defines limits and billing behavior per plan.
type PlanConfig struct {
	MonthlyLimit int64 // -1 = unlimited
	DailyLimit   int64 // -1 = no daily limit
	HardLimit    bool  // true = block at limit, false = overage billing
	BillAllUsage bool  // true = ingest all emails to Polar
}

// PlanConfigs maps each plan to its configuration.
var PlanConfigs = map[UserPlan]PlanConfig{
	UserPlanFree: {
		MonthlyLimit: 3000,
		DailyLimit:   100,
		HardLimit:    true,
		BillAllUsage: false,
	},
	UserPlanScale: {
		MonthlyLimit: 50000,
		DailyLimit:   -1,
		HardLimit:    false, // Overage allowed
		BillAllUsage: false, // Only overage billed
	},
	UserPlanPAYG: {
		MonthlyLimit: -1,
		DailyLimit:   -1,
		HardLimit:    false,
		BillAllUsage: true, // All emails billed
	},
}

// GetConfig returns the configuration for this plan.
func (p UserPlan) GetConfig() PlanConfig {
	if cfg, ok := PlanConfigs[p]; ok {
		return cfg
	}
	return PlanConfigs[UserPlanFree]
}

// User represents a user entity in the system.
type User struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	Name            string    `json:"name"`
	Role            UserRole  `json:"role"`
	IsActive        bool      `json:"is_active"`
	ExternalID      *string   `json:"external_id,omitempty"`       // Clerk user ID
	Plan            UserPlan  `json:"plan"`                        // Billing plan
	PolarCustomerID *string   `json:"polar_customer_id,omitempty"` // Polar customer ID
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateUserRequest represents a request to create a new user.
type CreateUserRequest struct {
	Email      string   `json:"email" binding:"required,email"`
	Name       string   `json:"name" binding:"required"`
	Role       UserRole `json:"role,omitempty"`
	ExternalID *string  `json:"external_id,omitempty"` // Clerk user ID (set via webhook)
}

// UpdateUserRequest represents a request to update a user.
type UpdateUserRequest struct {
	Email           *string   `json:"email,omitempty" binding:"omitempty,email"`
	Name            *string   `json:"name,omitempty"`
	Role            *UserRole `json:"role,omitempty"`
	IsActive        *bool     `json:"is_active,omitempty"`
	ExternalID      *string   `json:"external_id,omitempty"`
	Plan            *UserPlan `json:"plan,omitempty"`
	PolarCustomerID *string   `json:"polar_customer_id,omitempty"`
}
