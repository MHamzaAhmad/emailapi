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

// ApiKeyHandler implements the Connect ApiKeyServiceHandler.
type ApiKeyHandler struct {
	emailapiv1connect.UnimplementedApiKeyServiceHandler
	svc *service.APIKeyService
}

// NewApiKeyHandler creates a new ApiKeyHandler.
func NewApiKeyHandler(svc *service.APIKeyService) *ApiKeyHandler {
	return &ApiKeyHandler{svc: svc}
}

// CreateApiKey handles the CreateApiKey RPC.
func (h *ApiKeyHandler) CreateApiKey(
	ctx context.Context,
	req *connect.Request[emailapiv1.CreateApiKeyRequest],
) (*connect.Response[emailapiv1.CreateApiKeyResponse], error) {
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

	return connect.NewResponse(&emailapiv1.CreateApiKeyResponse{
		ApiKey:  toProtoApiKey(apiKey),
		RawKey:  rawKey,
		Message: "Store this API key securely. It will not be shown again.",
	}), nil
}

// GetApiKey handles the GetApiKey RPC.
func (h *ApiKeyHandler) GetApiKey(
	ctx context.Context,
	req *connect.Request[emailapiv1.GetApiKeyRequest],
) (*connect.Response[emailapiv1.ApiKey], error) {
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
	req *connect.Request[emailapiv1.ListApiKeysRequest],
) (*connect.Response[emailapiv1.ListApiKeysResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	apiKeys, err := h.svc.List(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoKeys := make([]*emailapiv1.ApiKey, len(apiKeys))
	for i, k := range apiKeys {
		protoKeys[i] = toProtoApiKey(k)
	}

	return connect.NewResponse(&emailapiv1.ListApiKeysResponse{
		Data: protoKeys,
	}), nil
}

// UpdateApiKey handles the UpdateApiKey RPC.
func (h *ApiKeyHandler) UpdateApiKey(
	ctx context.Context,
	req *connect.Request[emailapiv1.UpdateApiKeyRequest],
) (*connect.Response[emailapiv1.ApiKey], error) {
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
	req *connect.Request[emailapiv1.DeleteApiKeyRequest],
) (*connect.Response[emailapiv1.DeleteApiKeyResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	if err := h.svc.Delete(ctx, userID, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("API key not found"))
	}

	return connect.NewResponse(&emailapiv1.DeleteApiKeyResponse{}), nil
}

// RevokeApiKey handles the RevokeApiKey RPC.
func (h *ApiKeyHandler) RevokeApiKey(
	ctx context.Context,
	req *connect.Request[emailapiv1.RevokeApiKeyRequest],
) (*connect.Response[emailapiv1.ApiKey], error) {
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
