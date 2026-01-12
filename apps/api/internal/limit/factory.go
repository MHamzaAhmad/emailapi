package limit

// EmailConfig holds configuration for the email limit engine.
type EmailConfig struct {
	ReputationCache ReputationCacheInterface
	CreditCache     CreditCacheInterface
	FreeDailyLimit  int64 // Default: 100
}

// NewEmailEngine creates a LimitEngine configured for email sending.
// Runs suspension, daily, and monthly checks in parallel.
func NewEmailEngine(cfg EmailConfig) *Engine {
	var checkers []Checker

	// 1. Suspension checker (highest priority)
	if cfg.ReputationCache != nil {
		checkers = append(checkers, NewSuspensionChecker(cfg.ReputationCache))
	}

	// 2. Daily quota checker
	if cfg.CreditCache != nil {
		dailyLimit := cfg.FreeDailyLimit
		if dailyLimit <= 0 {
			dailyLimit = 100
		}
		checkers = append(checkers, NewDailyQuotaChecker(DailyQuotaConfig{
			CreditCache: cfg.CreditCache,
			RepCache:    cfg.ReputationCache,
			FreeLimit:   dailyLimit,
		}))
	}

	// 3. Monthly quota checker (Polar credit balance)
	if cfg.CreditCache != nil {
		checkers = append(checkers, NewMonthlyQuotaChecker(cfg.CreditCache))
	}

	return NewEngine(checkers...)
}
