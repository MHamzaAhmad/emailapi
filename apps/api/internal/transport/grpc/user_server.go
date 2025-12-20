package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

	user, apiKey, err := s.svc.Create(ctx, domainReq)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to create user: %v", err)
	}

	return &emailapiv1.CreateUserResponse{
		User:    toProtoUser(user),
		ApiKey:  apiKey,
		Message: "Store this API key securely. It will not be shown again.",
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

// RegenerateAPIKey handles the RegenerateAPIKey RPC.
func (s *UserServer) RegenerateAPIKey(ctx context.Context, req *emailapiv1.RegenerateAPIKeyRequest) (*emailapiv1.RegenerateAPIKeyResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	resp, err := s.svc.RegenerateAPIKey(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to regenerate API key: %v", err)
	}

	return &emailapiv1.RegenerateAPIKeyResponse{
		ApiKey:  resp.APIKey,
		Message: "Store this API key securely. It will not be shown again.",
	}, nil
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
		Id:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		Role:         toProtoUserRole(u.Role),
		ApiKeyPrefix: u.APIKeyPrefix,
		IsActive:     u.IsActive,
	}
}
