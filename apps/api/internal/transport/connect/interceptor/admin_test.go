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

func TestAdminInterceptor_WrapUnary(t *testing.T) {
	t.Run("non-admin procedure skips check", func(t *testing.T) {
		interceptor := NewAdminInterceptor(AdminConfig{})
		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)
		resp, err := interceptor.WrapUnary(mockHandler)(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("admin procedure requires admin role", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserLookup := mocks.NewMockUserRoleLookup(ctrl)
		interceptor := NewAdminInterceptor(AdminConfig{
			UserLookup: mockUserLookup,
		})

		ctx := newTestContext("admin_user")
		req := newMockUnaryRequest("/v1.AdminService/ListUsers", nil)

		mockUserLookup.EXPECT().
			GetByID(ctx, "admin_user").
			Return(&domain.User{
				ID:   "admin_user",
				Role: domain.UserRoleAdmin,
			}, nil)

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		resp, err := interceptor.WrapUnary(mockHandler)(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("non-admin user denied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserLookup := mocks.NewMockUserRoleLookup(ctrl)
		interceptor := NewAdminInterceptor(AdminConfig{
			UserLookup: mockUserLookup,
		})

		ctx := newTestContext("regular_user")
		req := newMockUnaryRequest("/v1.AdminService/ListUsers", nil)

		mockUserLookup.EXPECT().
			GetByID(ctx, "regular_user").
			Return(&domain.User{
				ID:   "regular_user",
				Role: domain.UserRoleMember,
			}, nil)

		_, err := interceptor.WrapUnary(nil)(ctx, req)
		require.Error(t, err)
		assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	})

	t.Run("unauthenticated user denied", func(t *testing.T) {
		interceptor := NewAdminInterceptor(AdminConfig{})

		// Context without user ID
		ctx := context.Background()
		req := newMockUnaryRequest("/v1.AdminService/ListUsers", nil)

		_, err := interceptor.WrapUnary(nil)(ctx, req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	})

	t.Run("user lookup error returns internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserLookup := mocks.NewMockUserRoleLookup(ctrl)
		interceptor := NewAdminInterceptor(AdminConfig{
			UserLookup: mockUserLookup,
		})

		ctx := newTestContext("user_error")
		req := newMockUnaryRequest("/v1.AdminService/ListUsers", nil)

		mockUserLookup.EXPECT().
			GetByID(ctx, "user_error").
			Return(nil, errors.New("db error"))

		_, err := interceptor.WrapUnary(nil)(ctx, req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})
}

func TestAdminHelpers(t *testing.T) {
	t.Run("dynamic admin procedures", func(t *testing.T) {
		proc := "/v1.TestService/AdminOnly"

		// 1. Initially false
		assert.False(t, IsAdminProcedure(proc))

		// 2. Add
		AddAdminProcedure(proc)
		assert.True(t, IsAdminProcedure(proc))

		// 3. Remove
		RemoveAdminProcedure(proc)
		assert.False(t, IsAdminProcedure(proc))
	})

	t.Run("IsAdminRoute checks prefix", func(t *testing.T) {
		assert.True(t, IsAdminRoute("/v1.AdminService/ListUsers"))
		assert.False(t, IsAdminRoute("/v1.EmailService/SendEmail"))
	})

	t.Run("GetAdminUserID returns error if no user ID", func(t *testing.T) {
		ctx := context.Background()
		// userLookup can be nil here because it fails before usage
		uid, err := GetAdminUserID(ctx, nil)
		require.Error(t, err)
		assert.Empty(t, uid)
	})

	t.Run("GetAdminUserID checks RBAC via lookup if needed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockUserLookup := mocks.NewMockUserRoleLookup(ctrl)

		ctx := newTestContext("admin_id")

		mockUserLookup.EXPECT().
			GetByID(ctx, "admin_id").
			Return(&domain.User{
				ID:   "admin_id",
				Role: domain.UserRoleAdmin,
			}, nil)

		uid, err := GetAdminUserID(ctx, mockUserLookup)
		require.NoError(t, err)
		assert.Equal(t, "admin_id", uid)
	})

	t.Run("ListAdminProcedures returns list", func(t *testing.T) {
		procs := ListAdminProcedures()
		assert.Contains(t, procs, "/v1.AdminService/ListUsers")
	})
}
