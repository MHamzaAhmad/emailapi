package tinybird

// AnalyticsAggregator implements service.Analytics using Tinybird.
type AnalyticsAggregator struct {
	email    *EmailRepository
	activity *ActivityRepository
}

// NewAnalyticsAggregator creates a new AnalyticsAggregator.
func NewAnalyticsAggregator(email *EmailRepository, activity *ActivityRepository) *AnalyticsAggregator {
	return &AnalyticsAggregator{
		email:    email,
		activity: activity,
	}
}

// Email returns the email repository.
func (a *AnalyticsAggregator) Email() EmailRepositoryInterface {
	return a.email
}

// Activity returns the activity repository.
func (a *AnalyticsAggregator) Activity() ActivityRepositoryInterface {
	return a.activity
}
