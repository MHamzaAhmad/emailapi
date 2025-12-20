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

// ApiKeyServer implements the ApiKeyService gRPC server.
type ApiKeyServer struct {
	emailapiv1.UnimplementedApiKeyServiceServer
	svc *service.APIKeyService
}

// NewApiKeyServer creates a new ApiKeyServer.
func NewApiKeyServer(svc *service.APIKeyService) *ApiKeyServer {
	return &ApiKeyServer{svc: svc}
}

// CreateApiKey handles the CreateApiKey RPC.
func (s *ApiKeyServer) CreateApiKey(ctx context.Context, req *emailapiv1.CreateApiKeyRequest) (*emailapiv1.CreateApiKeyResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	domainReq := &domain.CreateAPIKeyRequest{
		Name:        req.Name,
		Scopes:      toScopes(req.Scopes),
		Environment: toEnvironment(req.Environment),
	}
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		domainReq.ExpiresAt = &t
	}

	apiKey, rawKey, err := s.svc.Create(ctx, userID, domainReq)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to create API key: %v", err)
	}

	return &emailapiv1.CreateApiKeyResponse{
		ApiKey:  toProtoApiKey(apiKey),
		RawKey:  rawKey,
		Message: "Store this API key securely. It will not be shown again.",
	}, nil
}

// GetApiKey handles the GetApiKey RPC.
func (s *ApiKeyServer) GetApiKey(ctx context.Context, req *emailapiv1.GetApiKeyRequest) (*emailapiv1.ApiKey, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	apiKey, err := s.svc.GetByID(ctx, userID, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "API key not found")
	}

	return toProtoApiKey(apiKey), nil
}

// ListApiKeys handles the ListApiKeys RPC.
func (s *ApiKeyServer) ListApiKeys(ctx context.Context, req *emailapiv1.ListApiKeysRequest) (*emailapiv1.ListApiKeysResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	apiKeys, err := s.svc.List(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list API keys: %v", err)
	}

	protoKeys := make([]*emailapiv1.ApiKey, len(apiKeys))
	for i, k := range apiKeys {
		protoKeys[i] = toProtoApiKey(k)
	}

	return &emailapiv1.ListApiKeysResponse{
		Data: protoKeys,
	}, nil
}

// UpdateApiKey handles the UpdateApiKey RPC.
func (s *ApiKeyServer) UpdateApiKey(ctx context.Context, req *emailapiv1.UpdateApiKeyRequest) (*emailapiv1.ApiKey, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	domainReq := &domain.UpdateAPIKeyRequest{}
	if req.Name != nil {
		domainReq.Name = req.Name
	}
	if len(req.Scopes) > 0 {
		scopes := toScopes(req.Scopes)
		domainReq.Scopes = scopes
	}
	if req.IsActive != nil {
		domainReq.IsActive = req.IsActive
	}
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		domainReq.ExpiresAt = &t
	}

	apiKey, err := s.svc.Update(ctx, userID, req.Id, domainReq)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to update API key: %v", err)
	}

	return toProtoApiKey(apiKey), nil
}

// DeleteApiKey handles the DeleteApiKey RPC.
func (s *ApiKeyServer) DeleteApiKey(ctx context.Context, req *emailapiv1.DeleteApiKeyRequest) (*emailapiv1.DeleteApiKeyResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if err := s.svc.Delete(ctx, userID, req.Id); err != nil {
		return nil, status.Errorf(codes.NotFound, "API key not found")
	}

	return &emailapiv1.DeleteApiKeyResponse{}, nil
}

// RevokeApiKey handles the RevokeApiKey RPC.
func (s *ApiKeyServer) RevokeApiKey(ctx context.Context, req *emailapiv1.RevokeApiKeyRequest) (*emailapiv1.ApiKey, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	apiKey, err := s.svc.Revoke(ctx, userID, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "API key not found")
	}

	return toProtoApiKey(apiKey), nil
}

// toProtoApiKey converts a domain.APIKey to proto.
func toProtoApiKey(k *domain.APIKey) *emailapiv1.ApiKey {
	proto := &emailapiv1.ApiKey{
		Id:          k.ID,
		UserId:      k.UserID,
		Name:        k.Name,
		KeyPrefix:   k.KeyPrefix,
		Scopes:      toProtoScopes(k.Scopes),
		Environment: toProtoEnvironment(k.Environment),
		IsActive:    k.IsActive,
		CreatedAt:   timestamppb.New(k.CreatedAt),
		UpdatedAt:   timestamppb.New(k.UpdatedAt),
	}

	if k.LastUsedAt != nil {
		proto.LastUsedAt = timestamppb.New(*k.LastUsedAt)
	}
	if k.ExpiresAt != nil {
		proto.ExpiresAt = timestamppb.New(*k.ExpiresAt)
	}

	return proto
}

// toScopes converts proto scopes to domain scopes.
func toScopes(protoScopes []emailapiv1.Scope) []domain.Scope {
	scopes := make([]domain.Scope, 0, len(protoScopes))
	for _, s := range protoScopes {
		switch s {
		case emailapiv1.Scope_SCOPE_EMAIL_SEND:
			scopes = append(scopes, domain.ScopeEmailSend)
		case emailapiv1.Scope_SCOPE_EMAIL_READ:
			scopes = append(scopes, domain.ScopeEmailRead)
		case emailapiv1.Scope_SCOPE_DOMAIN_READ:
			scopes = append(scopes, domain.ScopeDomainRead)
		case emailapiv1.Scope_SCOPE_DOMAIN_WRITE:
			scopes = append(scopes, domain.ScopeDomainWrite)
		case emailapiv1.Scope_SCOPE_APIKEY_READ:
			scopes = append(scopes, domain.ScopeApiKeyRead)
		case emailapiv1.Scope_SCOPE_APIKEY_WRITE:
			scopes = append(scopes, domain.ScopeApiKeyWrite)
		case emailapiv1.Scope_SCOPE_USER_READ:
			scopes = append(scopes, domain.ScopeUserRead)
		case emailapiv1.Scope_SCOPE_USER_WRITE:
			scopes = append(scopes, domain.ScopeUserWrite)
		}
	}
	return scopes
}

// toProtoScopes converts domain scopes to proto scopes.
func toProtoScopes(scopes []domain.Scope) []emailapiv1.Scope {
	protoScopes := make([]emailapiv1.Scope, 0, len(scopes))
	for _, s := range scopes {
		switch s {
		case domain.ScopeEmailSend:
			protoScopes = append(protoScopes, emailapiv1.Scope_SCOPE_EMAIL_SEND)
		case domain.ScopeEmailRead:
			protoScopes = append(protoScopes, emailapiv1.Scope_SCOPE_EMAIL_READ)
		case domain.ScopeDomainRead:
			protoScopes = append(protoScopes, emailapiv1.Scope_SCOPE_DOMAIN_READ)
		case domain.ScopeDomainWrite:
			protoScopes = append(protoScopes, emailapiv1.Scope_SCOPE_DOMAIN_WRITE)
		case domain.ScopeApiKeyRead:
			protoScopes = append(protoScopes, emailapiv1.Scope_SCOPE_APIKEY_READ)
		case domain.ScopeApiKeyWrite:
			protoScopes = append(protoScopes, emailapiv1.Scope_SCOPE_APIKEY_WRITE)
		case domain.ScopeUserRead:
			protoScopes = append(protoScopes, emailapiv1.Scope_SCOPE_USER_READ)
		case domain.ScopeUserWrite:
			protoScopes = append(protoScopes, emailapiv1.Scope_SCOPE_USER_WRITE)
		}
	}
	return protoScopes
}

// toEnvironment converts proto environment to domain environment.
func toEnvironment(env emailapiv1.Environment) domain.Environment {
	switch env {
	case emailapiv1.Environment_ENVIRONMENT_LIVE:
		return domain.EnvLive
	case emailapiv1.Environment_ENVIRONMENT_DEV:
		return domain.EnvDev
	default:
		return domain.EnvLive
	}
}

// toProtoEnvironment converts domain environment to proto environment.
func toProtoEnvironment(env domain.Environment) emailapiv1.Environment {
	switch env {
	case domain.EnvLive:
		return emailapiv1.Environment_ENVIRONMENT_LIVE
	case domain.EnvDev:
		return emailapiv1.Environment_ENVIRONMENT_DEV
	default:
		return emailapiv1.Environment_ENVIRONMENT_UNSPECIFIED
	}
}
