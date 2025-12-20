package service

// Service aggregates all business logic services.
// This is injected into the Transport layer.
type Service struct {
	store Store

	Email   *EmailService
	User    *UserService
	Webhook *WebhookService
}

// New creates a new Service with the given Store.
func New(store Store) *Service {
	svc := &Service{store: store}
	svc.Email = NewEmailService(store)
	svc.User = NewUserService(store)
	svc.Webhook = NewWebhookService(store)
	return svc
}
