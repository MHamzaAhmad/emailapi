package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/service"
)

// UserServer implements the UserService gRPC server.
type UserServer struct {
	emailapiv1.UnimplementedUserServiceServer
	svc *service.UserService
}

// NewUserServer creates a new UserServer.
func NewUserServer(svc *service.UserService) *UserServer {
	return &UserServer{svc: svc}
}

// CreateUser handles the CreateUser RPC.
func (s *UserServer) CreateUser(ctx context.Context, req *emailapiv1.CreateUserRequest) (*emailapiv1.CreateUserResponse, error) {
	domainReq := &domain.CreateUserRequest{
		Email: req.Email,
		Name:  req.Name,
		Role:  toRole(req.Role),
	}

	user, err := s.svc.Create(ctx, domainReq)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to create user: %v", err)
	}

	return &emailapiv1.CreateUserResponse{
		User:    toProtoUser(user),
		Message: "User created successfully. Create an API key to authenticate.",
	}, nil
}

// GetCurrentUser handles the GetCurrentUser RPC.
func (s *UserServer) GetCurrentUser(ctx context.Context, req *emailapiv1.GetCurrentUserRequest) (*emailapiv1.User, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	user, err := s.svc.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	return toProtoUser(user), nil
}

// UpdateUser handles the UpdateUser RPC.
func (s *UserServer) UpdateUser(ctx context.Context, req *emailapiv1.UpdateUserRequest) (*emailapiv1.User, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Only allow users to update their own profile (or admins)
	if req.Id != userID {
		return nil, status.Error(codes.PermissionDenied, "cannot update another user")
	}

	domainReq := &domain.UpdateUserRequest{}
	if req.Email != nil {
		domainReq.Email = req.Email
	}
	if req.Name != nil {
		domainReq.Name = req.Name
	}
	if req.Role != nil {
		role := toRole(*req.Role)
		domainReq.Role = &role
	}
	if req.IsActive != nil {
		domainReq.IsActive = req.IsActive
	}

	user, err := s.svc.Update(ctx, req.Id, domainReq)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to update user: %v", err)
	}

	return toProtoUser(user), nil
}

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
	return &emailapiv1.User{
		Id:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      toProtoUserRole(u.Role),
		IsActive:  u.IsActive,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}
