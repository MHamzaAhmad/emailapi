package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
)

// AdminHandler implements the Connect AdminServiceHandler.
type AdminHandler struct {
	v1connect.UnimplementedAdminServiceHandler
	svc *service.AdminService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(svc *service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// ListUsers handles the ListUsers RPC.
func (h *AdminHandler) ListUsers(
	ctx context.Context,
	req *connect.Request[v1.ListAdminUsersRequest],
) (*connect.Response[v1.ListAdminUsersResponse], error) {
	limit := int(req.Msg.PageSize)
	if limit <= 0 {
		limit = 20
	}
	offset := int(req.Msg.Offset)

	users, total, err := h.svc.ListUsers(ctx, limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoUsers := make([]*v1.AdminUser, len(users))
	for i, u := range users {
		protoUsers[i] = &v1.AdminUser{
			Id:          u.User.ID,
			Email:       u.User.Email,
			Name:        u.User.Name,
			Role:        string(u.User.Role),
			IsActive:    u.User.IsActive,
			IsSuspended: u.IsSuspended,
			IsFlagged:   u.IsFlagged,
			CreatedAt:   timestamppb.New(u.User.CreatedAt),
			UpdatedAt:   timestamppb.New(u.User.UpdatedAt),
		}
	}

	return connect.NewResponse(&v1.ListAdminUsersResponse{
		Users:      protoUsers,
		TotalCount: int32(total),
	}), nil
}

// ListFlaggedUsers handles the ListFlaggedUsers RPC.
func (h *AdminHandler) ListFlaggedUsers(
	ctx context.Context,
	req *connect.Request[v1.ListFlaggedUsersRequest],
) (*connect.Response[v1.ListFlaggedUsersResponse], error) {
	limit := int(req.Msg.PageSize)
	if limit <= 0 {
		limit = 50
	}
	offset := int(req.Msg.Offset)

	users, total, err := h.svc.ListFlaggedUsers(ctx, limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoUsers := make([]*v1.FlaggedUser, len(users))
	for i, u := range users {
		flaggedUser := &v1.FlaggedUser{
			UserId:          u.UserID,
			Email:           u.UserEmail,
			Name:            u.UserName,
			TotalBounces:    int32(u.TotalBounces),
			HardBounces:     int32(u.HardBounces),
			SoftBounces:     int32(u.SoftBounces),
			Complaints:      int32(u.Complaints),
			Bounces_30D:     int32(u.Bounces30d),
			Complaints_30D:  int32(u.Complaints30d),
			SuspensionScore: u.SuspensionScore,
			FlaggedReason:   u.FlaggedReason,
		}
		if u.FlaggedAt != nil {
			flaggedUser.FlaggedAt = timestamppb.New(*u.FlaggedAt)
		}
		protoUsers[i] = flaggedUser
	}

	return connect.NewResponse(&v1.ListFlaggedUsersResponse{
		Users:      protoUsers,
		TotalCount: int32(total),
	}), nil
}

// GetUserDetails handles the GetUserDetails RPC.
func (h *AdminHandler) GetUserDetails(
	ctx context.Context,
	req *connect.Request[v1.GetUserDetailsRequest],
) (*connect.Response[v1.UserDetails], error) {
	if req.Msg.UserId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user_id is required"))
	}

	user, err := h.svc.GetUserDetails(ctx, req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	details := &v1.UserDetails{
		Id:              user.ID,
		Email:           user.Email,
		Name:            user.Name,
		Role:            string(user.Role),
		IsActive:        user.IsActive,
		TotalBounces:    int32(user.TotalBounces),
		HardBounces:     int32(user.HardBounces),
		SoftBounces:     int32(user.SoftBounces),
		Complaints:      int32(user.Complaints),
		Bounces_30D:     int32(user.Bounces30d),
		Complaints_30D:  int32(user.Complaints30d),
		SuspensionScore: user.SuspensionScore,
		IsFlagged:       user.IsFlagged,
		IsSuspended:     user.IsSuspended,
	}

	if user.FlaggedReason != nil {
		details.FlaggedReason = user.FlaggedReason
	}
	if user.SuspendedBy != nil {
		details.SuspendedBy = user.SuspendedBy
	}
	if user.SuspensionReason != nil {
		details.SuspensionReason = user.SuspensionReason
	}

	return connect.NewResponse(details), nil
}

// SuspendUser handles the SuspendUser RPC.
func (h *AdminHandler) SuspendUser(
	ctx context.Context,
	req *connect.Request[v1.SuspendUserRequest],
) (*connect.Response[v1.SuspendUserResponse], error) {
	if req.Msg.UserId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user_id is required"))
	}
	if req.Msg.Reason == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("reason is required"))
	}

	adminID := interceptor.GetUserID(ctx)
	if adminID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin not authenticated"))
	}

	if err := h.svc.SuspendUser(ctx, req.Msg.UserId, adminID, req.Msg.Reason); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.SuspendUserResponse{
		Message: "User suspended successfully",
	}), nil
}

// UnsuspendUser handles the UnsuspendUser RPC.
func (h *AdminHandler) UnsuspendUser(
	ctx context.Context,
	req *connect.Request[v1.UnsuspendUserRequest],
) (*connect.Response[v1.UnsuspendUserResponse], error) {
	if req.Msg.UserId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user_id is required"))
	}

	adminID := interceptor.GetUserID(ctx)
	if adminID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin not authenticated"))
	}

	if err := h.svc.UnsuspendUser(ctx, req.Msg.UserId, adminID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.UnsuspendUserResponse{
		Message: "User unsuspended successfully",
	}), nil
}
