package domain

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

// PlanInfo contains display information for a plan.
type PlanInfo struct {
	ID           string
	Name         string
	Description  string
	PriceCents   int64
	MonthlyLimit int64
	DailyLimit   int64
}

// PlanInfos provides display information for each plan.
var PlanInfos = map[UserPlan]PlanInfo{
	UserPlanFree: {
		ID:           "free",
		Name:         "Free",
		Description:  "Perfect for getting started",
		PriceCents:   0,
		MonthlyLimit: 3000,
		DailyLimit:   100,
	},
	UserPlanScale: {
		ID:           "scale",
		Name:         "Scale",
		Description:  "For growing applications",
		PriceCents:   2900, // $29/mo
		MonthlyLimit: 50000,
		DailyLimit:   -1, // Unlimited
	},
	UserPlanPAYG: {
		ID:           "payg",
		Name:         "Pay As You Go",
		Description:  "Usage-based pricing",
		PriceCents:   0,  // Usage-based
		MonthlyLimit: -1, // Unlimited
		DailyLimit:   -1, // Unlimited
	},
}

// GetInfo returns the display information for this plan.
func (p UserPlan) GetInfo() PlanInfo {
	if info, ok := PlanInfos[p]; ok {
		return info
	}
	return PlanInfos[UserPlanFree]
}

// ProductPlanMap maps Polar product IDs to plan types.
// Initialized via InitProductMappings at startup.
var ProductPlanMap = map[string]UserPlan{}

// PlanProductMap maps plan IDs to Polar product IDs.
// Initialized via InitProductMappings at startup.
var PlanProductMap = map[string]string{}

// InitProductMappings configures the product-plan mappings from config.
// Call this at application startup with values from config.
func InitProductMappings(scaleProductID, paygProductID string) {
	// Clear existing maps
	ProductPlanMap = make(map[string]UserPlan)
	PlanProductMap = make(map[string]string)

	// Set up mappings from config
	if scaleProductID != "" {
		ProductPlanMap[scaleProductID] = UserPlanScale
		PlanProductMap["scale"] = scaleProductID
	}
	if paygProductID != "" {
		ProductPlanMap[paygProductID] = UserPlanPAYG
		PlanProductMap["payg"] = paygProductID
	}
}

// GetPlanFromProductID returns the plan for a Polar product ID.
func GetPlanFromProductID(productID string) UserPlan {
	if plan, ok := ProductPlanMap[productID]; ok {
		return plan
	}
	return UserPlanFree
}

// GetProductIDFromPlanID returns the Polar product ID for a plan ID string.
func GetProductIDFromPlanID(planID string) (string, bool) {
	productID, ok := PlanProductMap[planID]
	return productID, ok
}

// AllPlans returns all plan types in display order.
func AllPlans() []UserPlan {
	return []UserPlan{UserPlanFree, UserPlanScale, UserPlanPAYG}
}
