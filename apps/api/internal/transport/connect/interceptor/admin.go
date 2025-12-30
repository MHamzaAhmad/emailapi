package interceptor

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	"github.com/emailapi/api/internal/domain"
)

// UserRoleLookup interface for looking up user roles.
type UserRoleLookup interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

// AdminConfig holds configuration for the admin interceptor.
type AdminConfig struct {
	UserLookup UserRoleLookup
}

// adminProcedures defines which procedures require admin role.
var adminProcedures = map[string]bool{
	"/v1.AdminService/ListUsers":        true,
	"/v1.AdminService/ListFlaggedUsers": true,
	"/v1.AdminService/GetUserDetails":   true,
	"/v1.AdminService/SuspendUser":      true,
	"/v1.AdminService/UnsuspendUser":    true,
}

// adminInterceptor implements role-based access control for admin routes.
type adminInterceptor struct {
	cfg AdminConfig
}

// NewAdminInterceptor creates an interceptor that enforces admin role for designated routes.
func NewAdminInterceptor(cfg AdminConfig) connect.Interceptor {
	return &adminInterceptor{cfg: cfg}
}

// WrapUnary implements connect.Interceptor for unary calls.
func (a *adminInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		procedure := req.Spec().Procedure

		// Check if this is an admin procedure
		if !adminProcedures[procedure] {
			return next(ctx, req)
		}

		// Verify admin role
		if err := a.verifyAdminRole(ctx); err != nil {
			return nil, err
		}

		return next(ctx, req)
	}
}

// WrapStreamingClient implements connect.Interceptor (no-op for server).
func (a *adminInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor for server streaming calls.
func (a *adminInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		procedure := conn.Spec().Procedure

		// Check if this is an admin procedure
		if !adminProcedures[procedure] {
			return next(ctx, conn)
		}

		// Verify admin role
		if err := a.verifyAdminRole(ctx); err != nil {
			return err
		}

		return next(ctx, conn)
	}
}

// verifyAdminRole checks if the authenticated user has admin role.
func (a *adminInterceptor) verifyAdminRole(ctx context.Context) error {
	userID := GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	user, err := a.cfg.UserLookup.GetByID(ctx, userID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.New("failed to lookup user"))
	}

	if user.Role != domain.UserRoleAdmin {
		return connect.NewError(connect.CodePermissionDenied, errors.New("admin access required"))
	}

	return nil
}

// IsAdminProcedure returns true if the procedure requires admin role.
func IsAdminProcedure(procedure string) bool {
	return adminProcedures[procedure]
}

// GetAdminUserID is a helper to extract and validate admin user from context.
// Returns the user ID if the user is an admin, otherwise returns an error.
func GetAdminUserID(ctx context.Context, userLookup UserRoleLookup) (string, error) {
	userID := GetUserID(ctx)
	if userID == "" {
		return "", errors.New("user not authenticated")
	}

	user, err := userLookup.GetByID(ctx, userID)
	if err != nil {
		return "", errors.New("failed to lookup user")
	}

	if user.Role != domain.UserRoleAdmin {
		return "", errors.New("admin access required")
	}

	return userID, nil
}

// AddAdminProcedure allows dynamically adding admin procedures.
// Useful for extending admin routes at runtime or in tests.
func AddAdminProcedure(procedure string) {
	adminProcedures[procedure] = true
}

// RemoveAdminProcedure removes a procedure from admin protection.
func RemoveAdminProcedure(procedure string) {
	delete(adminProcedures, procedure)
}

// ListAdminProcedures returns all currently protected admin procedures.
func ListAdminProcedures() []string {
	procedures := make([]string, 0, len(adminProcedures))
	for p := range adminProcedures {
		procedures = append(procedures, p)
	}
	return procedures
}

// IsAdminRoute checks if a path is an admin route (starts with AdminService).
func IsAdminRoute(path string) bool {
	return strings.Contains(path, "/v1.AdminService/")
}
