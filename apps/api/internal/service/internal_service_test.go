package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	svix "github.com/svix/svix-webhooks/go"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/domain"
	repoMocks "github.com/emailapi/api/internal/repository/postgres/mocks"
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
)

func TestInternalService_HandleClerkWebhook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockUserRepo := repoMocks.NewMockUserRepository(ctrl)

	mockStore.EXPECT().Users().Return(mockUserRepo).AnyTimes()

	// Setup UserService with mocked store
	userSvc := NewUserService(mockStore, nil, nil) // no APIKey/Cache needed for Create

	// Svix secrets must be whsec_<base64>
	webhookSecret := "whsec_SGVsbG8gV29ybGQ=" // Valid base64 after prefix

	svc := NewInternalService(InternalServiceConfig{
		UserService:        userSvc,
		ClerkWebhookSecret: webhookSecret,
	})

	ctx := context.Background()

	t.Run("user.created success", func(t *testing.T) {
		payload := map[string]interface{}{
			"type":   "user.created",
			"object": "event",
			"data": map[string]interface{}{
				"id": "user_clerk_123",
				"email_addresses": []map[string]interface{}{
					{
						"id":            "email_1",
						"email_address": "newuser@example.com",
					},
				},
				"primary_email_address_id": "email_1",
				"first_name":               "New",
				"last_name":                "User",
			},
		}

		payloadBytes, _ := json.Marshal(payload)

		// Generate valid headers
		wh, err := svix.NewWebhook(webhookSecret)
		require.NoError(t, err)

		// Manually sign? Svix lib has Verify but no public Sign method usually?
		// Wait, svix-webhooks/go usually provides signing for testing or server side?
		// Actually, standard library usage for testing verification is to use `wh.Sign`.
		// Let's check if `Sign` is available. Checked online: Yes, `Sign(payload string, timestamp time.Time)`

		now := time.Now()
		headers := http.Header{}

		msgID := "msg_123"
		signature, err := wh.Sign(msgID, now, payloadBytes)
		require.NoError(t, err)

		t.Logf("Generated Signature: %s", signature)

		headers.Set("svix-id", msgID)
		headers.Set("svix-timestamp", fmt.Sprint(now.Unix()))

		// Adjust prefix if needed
		sigHeader := signature
		if len(signature) < 3 || signature[:3] != "v1," {
			sigHeader = "v1," + signature
		}
		headers.Set("svix-signature", sigHeader)

		// Mock UserService Expectations
		mockUserRepo.EXPECT().
			GetByEmail(ctx, "newuser@example.com").
			Return(nil, nil)

		mockUserRepo.EXPECT().
			GetByExternalID(ctx, "user_clerk_123").
			Return(nil, nil)

		mockUserRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, u *domain.User) error {
				assert.Equal(t, "newuser@example.com", u.Email)
				assert.Equal(t, "user_clerk_123", *u.ExternalID)
				return nil
			})

		processed, msg, err := svc.HandleClerkWebhook(ctx, payloadBytes, headers)
		require.NoError(t, err)
		assert.True(t, processed)
		assert.Contains(t, msg, "user created")
	})
}
