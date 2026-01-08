package connect

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/domain"

	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
	transporterrors "github.com/emailapi/api/internal/transport/errors"

)

// UserHandler implements the Connect UserServiceHandler.
type UserHandler struct {
	v1connect.UnimplementedUserServiceHandler
	svc    *service.UserService
	repSvc *service.ReputationService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *service.UserService, repSvc *service.ReputationService) *UserHandler {
	return &UserHandler{svc: svc, repSvc: repSvc}
}

// CreateUser handles the CreateUser RPC.
func (h *UserHandler) CreateUser(
	ctx context.Context,
	req *connect.Request[v1.CreateUserRequest],
) (*connect.Response[v1.CreateUserResponse], error) {
	domainReq := &domain.CreateUserRequest{
		Email: req.Msg.Email,
		Name:  req.Msg.Name,
		Role:  toRole(req.Msg.Role),
	}

	user, err := h.svc.Create(ctx, domainReq)
	if err != nil {
		return nil, transporterrors.ToConnectError(err)
	}

	return connect.NewResponse(&v1.CreateUserResponse{
		User:    toProtoUser(user),
		Message: "User created successfully.",
	}), nil
}

// GetCurrentUser handles the GetCurrentUser RPC.
func (h *UserHandler) GetCurrentUser(
	ctx context.Context,
	req *connect.Request[v1.GetCurrentUserRequest],
) (*connect.Response[v1.User], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, transporterrors.ToConnectError(domain.ErrUnauthenticated)
	}

	user, err := h.svc.GetByID(ctx, userID)
	if err != nil {
		return nil, transporterrors.ToConnectError(domain.ErrUserNotFound)
	}

	return connect.NewResponse(toProtoUser(user)), nil
}

// UpdateUser handles the UpdateUser RPC.
func (h *UserHandler) UpdateUser(
	ctx context.Context,
	req *connect.Request[v1.UpdateUserRequest],
) (*connect.Response[v1.User], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, transporterrors.ToConnectError(domain.ErrUnauthenticated)
	}

	// Only allow users to update their own profile
	if req.Msg.Id != userID {
		return nil, transporterrors.ToConnectError(domain.ErrPermissionDenied)
	}

	domainReq := &domain.UpdateUserRequest{}
	if req.Msg.Email != nil {
		domainReq.Email = req.Msg.Email
	}
	if req.Msg.Name != nil {
		domainReq.Name = req.Msg.Name
	}
	if req.Msg.Role != nil {
		role := toRole(*req.Msg.Role)
		domainReq.Role = &role
	}
	if req.Msg.IsActive != nil {
		domainReq.IsActive = req.Msg.IsActive
	}

	user, err := h.svc.Update(ctx, req.Msg.Id, domainReq)
	if err != nil {
		return nil, transporterrors.ToConnectError(err)
	}

	return connect.NewResponse(toProtoUser(user)), nil
}

// ListUsers handles the ListUsers RPC.
func (h *UserHandler) ListUsers(
	ctx context.Context,
	req *connect.Request[v1.ListUsersRequest],
) (*connect.Response[v1.ListUsersResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, transporterrors.ToConnectError(domain.ErrUnauthenticated)
	}

	limit := int(req.Msg.PageSize)
	if limit <= 0 {
		limit = 20
	}
	offset := int(req.Msg.Offset)

	users, err := h.svc.List(ctx, limit, offset)
	if err != nil {
		return nil, transporterrors.ToConnectError(err)
	}

	protoUsers := make([]*v1.User, len(users))
	for i, u := range users {
		protoUsers[i] = toProtoUser(u)
	}

	return connect.NewResponse(&v1.ListUsersResponse{
		Users:      protoUsers,
		TotalCount: int32(len(protoUsers)),
	}), nil
}

// GetSuspensionStatus returns the current user's suspension status for FE banner.
func (h *UserHandler) GetSuspensionStatus(
	ctx context.Context,
	req *connect.Request[v1.GetSuspensionStatusRequest],
) (*connect.Response[v1.SuspensionStatus], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, transporterrors.ToConnectError(domain.ErrUnauthenticated)
	}

	// Get reputation status
	rep, err := h.repSvc.GetUserReputation(ctx, userID)
	if err != nil {
		// No reputation record = not suspended/flagged
		return connect.NewResponse(&v1.SuspensionStatus{
			IsSuspended: false,
			IsFlagged:   false,
		}), nil
	}

	status := &v1.SuspensionStatus{
		IsSuspended: rep.IsSuspended,
		IsFlagged:   rep.IsFlagged,
	}

	if rep.SuspensionReason != "" {
		status.SuspensionReason = &rep.SuspensionReason
	}
	if rep.SuspendedAt != nil {
		status.SuspendedAt = timestamppb.New(*rep.SuspendedAt)
	}
	if rep.FlaggedReason != "" {
		status.FlaggedReason = &rep.FlaggedReason
	}

	return connect.NewResponse(status), nil
}

// ============================================================================
// Proto Conversion Helpers
// ============================================================================

func toRole(r v1.UserRole) domain.UserRole {
	switch r {
	case v1.UserRole_USER_ROLE_ADMIN:
		return domain.UserRoleAdmin
	case v1.UserRole_USER_ROLE_MEMBER:
		return domain.UserRoleMember
	default:
		return domain.UserRoleMember
	}
}

func toProtoUserRole(r domain.UserRole) v1.UserRole {
	switch r {
	case domain.UserRoleAdmin:
		return v1.UserRole_USER_ROLE_ADMIN
	case domain.UserRoleMember:
		return v1.UserRole_USER_ROLE_MEMBER
	default:
		return v1.UserRole_USER_ROLE_UNSPECIFIED
	}
}

func toProtoUser(u *domain.User) *v1.User {
	user := &v1.User{
		Id:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      toProtoUserRole(u.Role),
		IsActive:  u.IsActive,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
	if u.ExternalID != nil {
		user.ExternalId = u.ExternalID
	}
	return user
}
