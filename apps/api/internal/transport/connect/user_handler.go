package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/emailapiv1connect"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
)

// UserHandler implements the Connect UserServiceHandler.
type UserHandler struct {
	emailapiv1connect.UnimplementedUserServiceHandler
	svc *service.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// CreateUser handles the CreateUser RPC.
func (h *UserHandler) CreateUser(
	ctx context.Context,
	req *connect.Request[emailapiv1.CreateUserRequest],
) (*connect.Response[emailapiv1.CreateUserResponse], error) {
	domainReq := &domain.CreateUserRequest{
		Email: req.Msg.Email,
		Name:  req.Msg.Name,
		Role:  toRole(req.Msg.Role),
	}

	user, apiKey, err := h.svc.Create(ctx, domainReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&emailapiv1.CreateUserResponse{
		User:    toProtoUser(user),
		Message: "User created successfully.",
		ApiKey:  apiKey,
	}), nil
}

// GetCurrentUser handles the GetCurrentUser RPC.
func (h *UserHandler) GetCurrentUser(
	ctx context.Context,
	req *connect.Request[emailapiv1.GetCurrentUserRequest],
) (*connect.Response[emailapiv1.User], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	user, err := h.svc.GetByID(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}

	return connect.NewResponse(toProtoUser(user)), nil
}

// UpdateUser handles the UpdateUser RPC.
func (h *UserHandler) UpdateUser(
	ctx context.Context,
	req *connect.Request[emailapiv1.UpdateUserRequest],
) (*connect.Response[emailapiv1.User], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	// Only allow users to update their own profile
	if req.Msg.Id != userID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot update another user"))
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
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(toProtoUser(user)), nil
}

// ListUsers handles the ListUsers RPC.
func (h *UserHandler) ListUsers(
	ctx context.Context,
	req *connect.Request[emailapiv1.ListUsersRequest],
) (*connect.Response[emailapiv1.ListUsersResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	limit := int(req.Msg.PageSize)
	if limit <= 0 {
		limit = 20
	}
	offset := int(req.Msg.Offset)

	users, err := h.svc.List(ctx, limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoUsers := make([]*emailapiv1.User, len(users))
	for i, u := range users {
		protoUsers[i] = toProtoUser(u)
	}

	return connect.NewResponse(&emailapiv1.ListUsersResponse{
		Users:      protoUsers,
		TotalCount: int32(len(protoUsers)),
	}), nil
}

// ============================================================================
// Proto Conversion Helpers
// ============================================================================

func toRole(r emailapiv1.UserRole) domain.UserRole {
	switch r {
	case emailapiv1.UserRole_USER_ROLE_ADMIN:
		return domain.UserRoleAdmin
	case emailapiv1.UserRole_USER_ROLE_MEMBER:
		return domain.UserRoleMember
	default:
		return domain.UserRoleMember
	}
}

func toProtoUserRole(r domain.UserRole) emailapiv1.UserRole {
	switch r {
	case domain.UserRoleAdmin:
		return emailapiv1.UserRole_USER_ROLE_ADMIN
	case domain.UserRoleMember:
		return emailapiv1.UserRole_USER_ROLE_MEMBER
	default:
		return emailapiv1.UserRole_USER_ROLE_UNSPECIFIED
	}
}

func toProtoUser(u *domain.User) *emailapiv1.User {
	user := &emailapiv1.User{
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
