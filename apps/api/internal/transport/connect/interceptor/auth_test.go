package interceptor

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/transport/connect/interceptor/mocks"
)

func TestAuthInterceptor_WrapUnary(t *testing.T) {
	t.Run("public procedure skips auth", func(t *testing.T) {
		interceptor := NewCombinedAuthInterceptor(AuthConfig{})
		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		req := newMockUnaryRequest("/v1.BillingService/GetPlans", nil)
		resp, err := interceptor.WrapUnary(mockHandler)(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("internal service skips auth", func(t *testing.T) {
		interceptor := NewCombinedAuthInterceptor(AuthConfig{})
		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		req := newMockUnaryRequest("/v1.InternalService/SomeMethod", nil)
		resp, err := interceptor.WrapUnary(mockHandler)(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("missing auth header", func(t *testing.T) {
		interceptor := NewCombinedAuthInterceptor(AuthConfig{})
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		_, err := interceptor.WrapUnary(nil)(context.Background(), req)

		require.Error(t, err)
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	})

	t.Run("invalid auth header format", func(t *testing.T) {
		interceptor := NewCombinedAuthInterceptor(AuthConfig{})
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", map[string]string{
			"Authorization": "InvalidFormat",
		})

		_, err := interceptor.WrapUnary(nil)(context.Background(), req)

		require.Error(t, err)
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	})

	t.Run("valid API key auth", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPIKeyService := mocks.NewMockAPIKeyValidator(ctrl)

		interceptor := NewCombinedAuthInterceptor(AuthConfig{
			APIKeyService: mockAPIKeyService,
		})

		apiKey := &domain.APIKey{
			ID:     "key_123",
			UserID: "user_123",
			Scopes: []domain.Scope{domain.ScopeEmailSend},
		}
		user := &domain.User{ID: "user_123"}

		mockAPIKeyService.EXPECT().
			ValidateAndGetUser(gomock.Any(), "sea_live_test_key").
			Return(user, apiKey, nil)

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			assert.Equal(t, "user_123", GetUserID(ctx))
			assert.Equal(t, AuthMethodAPIKey, GetAuthMethod(ctx))
			assert.Equal(t, "key_123", GetAPIKeyID(ctx))
			return connect.NewResponse(&struct{}{}), nil
		}

		req := newMockUnaryRequest("/v1.EmailService/SendEmail", map[string]string{
			"Authorization": "Bearer sea_live_test_key",
		})

		resp, err := interceptor.WrapUnary(mockHandler)(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("API key missing scope", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPIKeyService := mocks.NewMockAPIKeyValidator(ctrl)

		interceptor := NewCombinedAuthInterceptor(AuthConfig{
			APIKeyService: mockAPIKeyService,
		})

		apiKey := &domain.APIKey{
			ID:     "key_123",
			UserID: "user_123",
			Scopes: []domain.Scope{domain.ScopeDomainRead}, // Missing SendEmail scope
		}
		user := &domain.User{ID: "user_123"}

		mockAPIKeyService.EXPECT().
			ValidateAndGetUser(gomock.Any(), "sea_live_test_key").
			Return(user, apiKey, nil)

		req := newMockUnaryRequest("/v1.EmailService/SendEmail", map[string]string{
			"Authorization": "Bearer sea_live_test_key",
		})

		_, err := interceptor.WrapUnary(nil)(context.Background(), req)
		require.Error(t, err)
		assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	})

	t.Run("API key validation error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPIKeyService := mocks.NewMockAPIKeyValidator(ctrl)

		interceptor := NewCombinedAuthInterceptor(AuthConfig{
			APIKeyService: mockAPIKeyService,
		})

		mockAPIKeyService.EXPECT().
			ValidateAndGetUser(gomock.Any(), "sea_live_invalid").
			Return(nil, nil, errors.New("invalid key"))

		req := newMockUnaryRequest("/v1.EmailService/SendEmail", map[string]string{
			"Authorization": "Bearer sea_live_invalid",
		})

		_, err := interceptor.WrapUnary(nil)(context.Background(), req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	})

	t.Run("invalid bearer format", func(t *testing.T) {
		interceptor := NewCombinedAuthInterceptor(AuthConfig{})
		// No space after Bearer
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", map[string]string{
			"Authorization": "Bearertoken",
		})
		_, err := interceptor.WrapUnary(nil)(context.Background(), req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

		// Wrong prefix
		req = newMockUnaryRequest("/v1.EmailService/SendEmail", map[string]string{
			"Authorization": "Basic token",
		})
		_, err = interceptor.WrapUnary(nil)(context.Background(), req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	})

	t.Run("clerk only endpoint rejects api key", func(t *testing.T) {
		interceptor := NewCombinedAuthInterceptor(AuthConfig{})
		// Use a procedure that is NOT internal/public/allowed-for-api-key
		req := newMockUnaryRequest("/v1.TestService/ClerkOnly", map[string]string{
			"Authorization": "Bearer sea_live_key",
		})

		// Ensure we don't panic if it were to succeed (it shouldn't)
		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		// Force it to be treated as API key but rejected by allowlist
		_, err := interceptor.WrapUnary(mockHandler)(context.Background(), req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		assert.Contains(t, err.Error(), "API keys not allowed")
	})

	t.Run("malformed clerk token returns unauthenticated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockKeyValidator := mocks.NewMockAPIKeyValidator(ctrl)

		interceptor := NewCombinedAuthInterceptor(AuthConfig{
			APIKeyService: mockKeyValidator,
		})

		req := newMockUnaryRequest("/v1.EmailService/SendEmail", map[string]string{
			"Authorization": "Bearer header.body.sig", // malformed
		})

		// Malformed token fails Clerk check (assumed), falls back to API key check
		mockKeyValidator.EXPECT().
			ValidateAndGetUser(gomock.Any(), "header.body.sig").
			Return(nil, nil, errors.New("invalid api key"))

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		_, err := interceptor.WrapUnary(mockHandler)(context.Background(), req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	})
}

func TestAuthHelpers(t *testing.T) {
	t.Run("extracts values correctly", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextKeyUserID, "u1")
		ctx = context.WithValue(ctx, ContextKeyAuthMethod, AuthMethodAPIKey)
		ctx = context.WithValue(ctx, ContextKeyAPIKeyID, "ak1")

		assert.Equal(t, "u1", GetUserID(ctx))
		assert.Equal(t, AuthMethodAPIKey, GetAuthMethod(ctx))
		assert.Equal(t, "ak1", GetAPIKeyID(ctx))
	})

	t.Run("returns empty on missing values", func(t *testing.T) {
		ctx := context.Background()
		assert.Empty(t, GetUserID(ctx))
		assert.Empty(t, GetAuthMethod(ctx))
		assert.Empty(t, GetAPIKeyID(ctx))
	})
}
