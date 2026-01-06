package domain

// UserPlan represents a user's subscription plan.
type UserPlan string

const (
	UserPlanFree    UserPlan = "free"
	UserPlanStarter UserPlan = "starter"
	UserPlanGrowth  UserPlan = "growth"
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
	UserPlanStarter: {
		MonthlyLimit: 50000,
		DailyLimit:   -1,
		HardLimit:    false, // Overage allowed
		BillAllUsage: false, // Only overage billed
	},
	UserPlanGrowth: {
		MonthlyLimit: 200000,
		DailyLimit:   -1,
		HardLimit:    false,
		BillAllUsage: false, // Only overage billed, base price covers first 200k
	},
}

// GetConfig returns the configuration for this plan.
func (p UserPlan) GetConfig() PlanConfig {
	if cfg, ok := PlanConfigs[p]; ok {
		return cfg
	}
	return PlanConfigs[UserPlanFree]
}

// Feature represents a plan feature with optional tooltip.
type Feature struct {
	Name    string
	Tooltip string // Markdown/Text for tooltip
}

// PlanInfo contains display information for a plan.
type PlanInfo struct {
	ID                string
	Name              string
	Description       string
	PriceCents        int64
	OveragePriceCents int64
	MonthlyLimit      int64
	DailyLimit        int64
	Features          []Feature
}

// PlanInfos provides display information for each plan.
var PlanInfos = map[UserPlan]PlanInfo{
	UserPlanFree: {
		ID:           "free",
		Name:         "Indie",
		Description:  "Perfect for getting started",
		PriceCents:   0,
		MonthlyLimit: 3000,
		DailyLimit:   100,
		Features: []Feature{
			{Name: "3,000 emails / month"},
			{Name: "100 emails / day"},
			{Name: "Unlimited domains"},
			{
				Name:    "Webhook & onReceive events",
				Tooltip: "**onReceive:** Zero-config, callback-based event stream perfect for coding agents & prototypes.\n\n**Webhooks:** Standard implementation for production apps.\n\nBoth support replies, bounces, complaints & delivery.",
			},
			{
				Name:    "Inbound & Outbound",
				Tooltip: "Every email you send is repliable and it's up to you if you handle the replies or not",
			},
			{Name: "Standard support"},
		},
	},
	UserPlanStarter: {
		ID:                "starter",
		Name:              "Starter",
		Description:       "For growing applications",
		PriceCents:        1200, // $12.00/mo
		OveragePriceCents: 25,   // $0.25 per 1000 emails
		MonthlyLimit:      50000,
		DailyLimit:        -1, // Unlimited
		Features: []Feature{
			{Name: "50,000 emails included"},
			{Name: "Overages: $0.25 / 1K"}, // Hardcoded formatted price
			{Name: "Unlimited daily sending"},
			{Name: "Unlimited domains"},
			{
				Name:    "Webhook & onReceive events",
				Tooltip: "**onReceive:** Zero-config, callback-based event stream perfect for coding agents & prototypes.\n\n**Webhooks:** Standard implementation for production apps.\n\nBoth support replies, bounces, complaints & delivery.",
			},
			{
				Name:    "Inbound & Outbound",
				Tooltip: "Every email you send is repliable and it's up to you if you handle the replies or not",
			},
			{Name: "Priority support"},
		},
	},
	UserPlanGrowth: {
		ID:                "growth",
		Name:              "Growth",
		Description:       "For high volume usage",
		PriceCents:        5000, // $50.00/mo
		OveragePriceCents: 25,   // $0.25 per 1000 emails
		MonthlyLimit:      200000,
		DailyLimit:        -1, // Unlimited
		Features: []Feature{
			{Name: "200,000 emails included"},
			{Name: "Overages: $0.25 / 1K"},
			{Name: "Unlimited daily sending"},
			{Name: "Unlimited domains"},
			{
				Name:    "Webhook & onReceive events",
				Tooltip: "**onReceive:** Zero-config, callback-based event stream perfect for coding agents & prototypes.\n\n**Webhooks:** Standard implementation for production apps.\n\nBoth support replies, bounces, complaints & delivery.",
			},
			{
				Name:    "Inbound & Outbound",
				Tooltip: "Every email you send is repliable and it's up to you if you handle the replies or not",
			},
			{Name: "Enterprise support"},
		},
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
func InitProductMappings(starterProductID, growthProductID string) {
	// Clear existing maps
	ProductPlanMap = make(map[string]UserPlan)
	PlanProductMap = make(map[string]string)

	// Set up mappings from config
	if starterProductID != "" {
		ProductPlanMap[starterProductID] = UserPlanStarter
		PlanProductMap["starter"] = starterProductID
	}
	if growthProductID != "" {
		ProductPlanMap[growthProductID] = UserPlanGrowth
		PlanProductMap["growth"] = growthProductID
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
	return []UserPlan{UserPlanFree, UserPlanStarter, UserPlanGrowth}
}
