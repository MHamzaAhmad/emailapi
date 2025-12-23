package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
)

// ApiKeyHandler implements the Connect ApiKeyServiceHandler.
type ApiKeyHandler struct {
	v1connect.UnimplementedApiKeyServiceHandler
	svc *service.APIKeyService
}

// NewApiKeyHandler creates a new ApiKeyHandler.
func NewApiKeyHandler(svc *service.APIKeyService) *ApiKeyHandler {
	return &ApiKeyHandler{svc: svc}
}

// CreateApiKey handles the CreateApiKey RPC.
func (h *ApiKeyHandler) CreateApiKey(
	ctx context.Context,
	req *connect.Request[v1.CreateApiKeyRequest],
) (*connect.Response[v1.CreateApiKeyResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	domainReq := &domain.CreateAPIKeyRequest{
		Name:        req.Msg.Name,
		Scopes:      toScopes(req.Msg.Scopes),
		Environment: toEnvironment(req.Msg.Environment),
	}
	if req.Msg.ExpiresAt != nil {
		t := req.Msg.ExpiresAt.AsTime()
		domainReq.ExpiresAt = &t
	}

	apiKey, rawKey, err := h.svc.Create(ctx, userID, domainReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&v1.CreateApiKeyResponse{
		ApiKey:  toProtoApiKey(apiKey),
		RawKey:  rawKey,
		Message: "Store this API key securely. It will not be shown again.",
	}), nil
}

// GetApiKey handles the GetApiKey RPC.
func (h *ApiKeyHandler) GetApiKey(
	ctx context.Context,
	req *connect.Request[v1.GetApiKeyRequest],
) (*connect.Response[v1.ApiKey], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	apiKey, err := h.svc.GetByID(ctx, userID, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("API key not found"))
	}

	return connect.NewResponse(toProtoApiKey(apiKey)), nil
}

// ListApiKeys handles the ListApiKeys RPC.
func (h *ApiKeyHandler) ListApiKeys(
	ctx context.Context,
	req *connect.Request[v1.ListApiKeysRequest],
) (*connect.Response[v1.ListApiKeysResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	apiKeys, err := h.svc.List(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoKeys := make([]*v1.ApiKey, len(apiKeys))
	for i, k := range apiKeys {
		protoKeys[i] = toProtoApiKey(k)
	}

	return connect.NewResponse(&v1.ListApiKeysResponse{
		Data: protoKeys,
	}), nil
}

// UpdateApiKey handles the UpdateApiKey RPC.
func (h *ApiKeyHandler) UpdateApiKey(
	ctx context.Context,
	req *connect.Request[v1.UpdateApiKeyRequest],
) (*connect.Response[v1.ApiKey], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	domainReq := &domain.UpdateAPIKeyRequest{}
	if req.Msg.Name != nil {
		domainReq.Name = req.Msg.Name
	}
	if len(req.Msg.Scopes) > 0 {
		scopes := toScopes(req.Msg.Scopes)
		domainReq.Scopes = scopes
	}
	if req.Msg.IsActive != nil {
		domainReq.IsActive = req.Msg.IsActive
	}
	if req.Msg.ExpiresAt != nil {
		t := req.Msg.ExpiresAt.AsTime()
		domainReq.ExpiresAt = &t
	}

	apiKey, err := h.svc.Update(ctx, userID, req.Msg.Id, domainReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(toProtoApiKey(apiKey)), nil
}

// DeleteApiKey handles the DeleteApiKey RPC.
func (h *ApiKeyHandler) DeleteApiKey(
	ctx context.Context,
	req *connect.Request[v1.DeleteApiKeyRequest],
) (*connect.Response[v1.DeleteApiKeyResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	if err := h.svc.Delete(ctx, userID, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("API key not found"))
	}

	return connect.NewResponse(&v1.DeleteApiKeyResponse{}), nil
}

// RevokeApiKey handles the RevokeApiKey RPC.
func (h *ApiKeyHandler) RevokeApiKey(
	ctx context.Context,
	req *connect.Request[v1.RevokeApiKeyRequest],
) (*connect.Response[v1.ApiKey], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	apiKey, err := h.svc.Revoke(ctx, userID, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("API key not found"))
	}

	return connect.NewResponse(toProtoApiKey(apiKey)), nil
}

// ============================================================================
// Proto Conversion Helpers
// ============================================================================

func toProtoApiKey(k *domain.APIKey) *v1.ApiKey {
	proto := &v1.ApiKey{
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

func toScopes(protoScopes []v1.Scope) []domain.Scope {
	scopes := make([]domain.Scope, 0, len(protoScopes))
	for _, s := range protoScopes {
		switch s {
		case v1.Scope_SCOPE_EMAIL_SEND:
			scopes = append(scopes, domain.ScopeEmailSend)
		case v1.Scope_SCOPE_EMAIL_READ:
			scopes = append(scopes, domain.ScopeEmailRead)
		case v1.Scope_SCOPE_DOMAIN_READ:
			scopes = append(scopes, domain.ScopeDomainRead)
		case v1.Scope_SCOPE_DOMAIN_WRITE:
			scopes = append(scopes, domain.ScopeDomainWrite)
		case v1.Scope_SCOPE_APIKEY_READ:
			scopes = append(scopes, domain.ScopeApiKeyRead)
		case v1.Scope_SCOPE_APIKEY_WRITE:
			scopes = append(scopes, domain.ScopeApiKeyWrite)
		case v1.Scope_SCOPE_USER_READ:
			scopes = append(scopes, domain.ScopeUserRead)
		case v1.Scope_SCOPE_USER_WRITE:
			scopes = append(scopes, domain.ScopeUserWrite)
		}
	}
	return scopes
}

func toProtoScopes(scopes []domain.Scope) []v1.Scope {
	protoScopes := make([]v1.Scope, 0, len(scopes))
	for _, s := range scopes {
		switch s {
		case domain.ScopeEmailSend:
			protoScopes = append(protoScopes, v1.Scope_SCOPE_EMAIL_SEND)
		case domain.ScopeEmailRead:
			protoScopes = append(protoScopes, v1.Scope_SCOPE_EMAIL_READ)
		case domain.ScopeDomainRead:
			protoScopes = append(protoScopes, v1.Scope_SCOPE_DOMAIN_READ)
		case domain.ScopeDomainWrite:
			protoScopes = append(protoScopes, v1.Scope_SCOPE_DOMAIN_WRITE)
		case domain.ScopeApiKeyRead:
			protoScopes = append(protoScopes, v1.Scope_SCOPE_APIKEY_READ)
		case domain.ScopeApiKeyWrite:
			protoScopes = append(protoScopes, v1.Scope_SCOPE_APIKEY_WRITE)
		case domain.ScopeUserRead:
			protoScopes = append(protoScopes, v1.Scope_SCOPE_USER_READ)
		case domain.ScopeUserWrite:
			protoScopes = append(protoScopes, v1.Scope_SCOPE_USER_WRITE)
		}
	}
	return protoScopes
}

func toEnvironment(env v1.Environment) domain.Environment {
	switch env {
	case v1.Environment_ENVIRONMENT_LIVE:
		return domain.EnvLive
	case v1.Environment_ENVIRONMENT_DEV:
		return domain.EnvDev
	default:
		return domain.EnvLive
	}
}

func toProtoEnvironment(env domain.Environment) v1.Environment {
	switch env {
	case domain.EnvLive:
		return v1.Environment_ENVIRONMENT_LIVE
	case domain.EnvDev:
		return v1.Environment_ENVIRONMENT_DEV
	default:
		return v1.Environment_ENVIRONMENT_UNSPECIFIED
	}
}
