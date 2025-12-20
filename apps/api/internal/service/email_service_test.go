package service_test

import (
	"context"
	"testing"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/repository"
	"github.com/emailapi/api/internal/service"
)

// MockStore implements service.Store for testing.
type MockStore struct {
	emails   *MockEmailRepository
	users    *MockUserRepository
	webhooks *MockWebhookRepository
	domains  *MockDomainRepository
}

func NewMockStore() *MockStore {
	return &MockStore{
		emails:   &MockEmailRepository{},
		users:    &MockUserRepository{},
		webhooks: &MockWebhookRepository{},
		domains:  &MockDomainRepository{},
	}
}

func (m *MockStore) Emails() repository.EmailRepository     { return m.emails }
func (m *MockStore) Users() repository.UserRepository       { return m.users }
func (m *MockStore) Webhooks() repository.WebhookRepository { return m.webhooks }
func (m *MockStore) Domains() repository.DomainRepository   { return m.domains }
func (m *MockStore) Close()                                 {}

// MockEmailRepository implements repository.EmailRepository for testing.
type MockEmailRepository struct {
	emails []*domain.Email
}

func (m *MockEmailRepository) Create(ctx context.Context, email *domain.Email) error {
	m.emails = append(m.emails, email)
	return nil
}

func (m *MockEmailRepository) GetByID(ctx context.Context, id string) (*domain.Email, error) {
	for _, e := range m.emails {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}

func (m *MockEmailRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Email, error) {
	var result []*domain.Email
	for _, e := range m.emails {
		if e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *MockEmailRepository) Update(ctx context.Context, email *domain.Email) error {
	for i, e := range m.emails {
		if e.ID == email.ID {
			m.emails[i] = email
			return nil
		}
	}
	return nil
}

func (m *MockEmailRepository) UpdateStatus(ctx context.Context, id string, status domain.EmailStatus) error {
	for _, e := range m.emails {
		if e.ID == id {
			e.Status = status
			return nil
		}
	}
	return nil
}

// MockUserRepository implements repository.UserRepository for testing.
type MockUserRepository struct {
	users []*domain.User
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	m.users = append(m.users, user)
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*domain.User, error) {
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *MockUserRepository) UpdateAPIKey(ctx context.Context, id, hashedKey, prefix string) error {
	return nil
}

// MockWebhookRepository implements repository.WebhookRepository for testing.
type MockWebhookRepository struct{}

func (m *MockWebhookRepository) Create(ctx context.Context, webhook *domain.Webhook) error {
	return nil
}
func (m *MockWebhookRepository) GetByID(ctx context.Context, id string) (*domain.Webhook, error) {
	return nil, nil
}
func (m *MockWebhookRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Webhook, error) {
	return nil, nil
}
func (m *MockWebhookRepository) GetActiveByEvent(ctx context.Context, eventType domain.WebhookEventType) ([]*domain.Webhook, error) {
	return nil, nil
}
func (m *MockWebhookRepository) Update(ctx context.Context, webhook *domain.Webhook) error {
	return nil
}
func (m *MockWebhookRepository) Delete(ctx context.Context, id string) error { return nil }
func (m *MockWebhookRepository) CreateDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error {
	return nil
}
func (m *MockWebhookRepository) GetDeliveriesByWebhookID(ctx context.Context, webhookID string, limit int) ([]*domain.WebhookDelivery, error) {
	return nil, nil
}

// MockDomainRepository implements repository.DomainRepository for testing.
type MockDomainRepository struct {
	domains []*domain.SendingDomain
}

func (m *MockDomainRepository) Create(ctx context.Context, d *domain.SendingDomain) error {
	m.domains = append(m.domains, d)
	return nil
}

func (m *MockDomainRepository) GetByID(ctx context.Context, id string) (*domain.SendingDomain, error) {
	for _, d := range m.domains {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, nil
}

func (m *MockDomainRepository) GetByDomainName(ctx context.Context, userID, domainName string) (*domain.SendingDomain, error) {
	for _, d := range m.domains {
		if d.UserID == userID && d.Domain == domainName {
			return d, nil
		}
	}
	return nil, nil
}

func (m *MockDomainRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.SendingDomain, error) {
	var result []*domain.SendingDomain
	for _, d := range m.domains {
		if d.UserID == userID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *MockDomainRepository) Update(ctx context.Context, d *domain.SendingDomain) error {
	for i, existing := range m.domains {
		if existing.ID == d.ID {
			m.domains[i] = d
			return nil
		}
	}
	return nil
}

func (m *MockDomainRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func TestEmailService_Send(t *testing.T) {
	store := NewMockStore()
	svc := service.New(store)

	ctx := context.Background()
	req := &domain.SendEmailRequest{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test body content",
	}

	resp, err := svc.Email.Send(ctx, "user-123", req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp.ID == "" {
		t.Error("expected email ID to be set")
	}

	if resp.Status != domain.EmailStatusSent {
		t.Errorf("expected status %s, got %s", domain.EmailStatusSent, resp.Status)
	}

	// Verify email was stored
	if len(store.emails.emails) != 1 {
		t.Errorf("expected 1 email in store, got %d", len(store.emails.emails))
	}
}

func TestEmailService_Send_ValidationError(t *testing.T) {
	store := NewMockStore()
	svc := service.New(store)

	ctx := context.Background()

	tests := []struct {
		name string
		req  *domain.SendEmailRequest
	}{
		{
			name: "missing recipients",
			req: &domain.SendEmailRequest{
				From:    "sender@example.com",
				To:      []string{},
				Subject: "Test",
				Body:    "Body",
			},
		},
		{
			name: "missing subject",
			req: &domain.SendEmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "",
				Body:    "Body",
			},
		},
		{
			name: "missing body",
			req: &domain.SendEmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Subject",
				Body:    "",
				HTML:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Email.Send(ctx, "user-123", tt.req)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestEmailService_List(t *testing.T) {
	store := NewMockStore()
	svc := service.New(store)

	ctx := context.Background()

	// Send some emails
	for i := 0; i < 3; i++ {
		req := &domain.SendEmailRequest{
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Subject: "Test",
			Body:    "Body",
		}
		_, _ = svc.Email.Send(ctx, "user-123", req)
	}

	emails, err := svc.Email.List(ctx, "user-123", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(emails) != 3 {
		t.Errorf("expected 3 emails, got %d", len(emails))
	}
}
