package connect

import (
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/domain"
)

// ToConnectError converts a domain.AppError to a connect.Error with details.
// If the error is not an AppError, it wraps it as an internal error.
func ToConnectError(err error) *connect.Error {
	if err == nil {
		return nil
	}

	appErr, ok := domain.IsAppError(err)
	if !ok {
		// Wrap unknown errors as internal
		return connect.NewError(connect.CodeInternal, err)
	}

	// Map our error codes to gRPC codes
	grpcCode := mapToGRPCCode(appErr.Code)
	connectErr := connect.NewError(grpcCode, errors.New(appErr.Message))

	// Add structured ErrorDetail for client-side parsing
	detail := &v1.ErrorDetail{
		Code:     appErr.Code,
		Message:  appErr.Message,
		Field:    appErr.Field,
		Metadata: appErr.Metadata,
	}

	if detailAny, err := connect.NewErrorDetail(detail); err == nil {
		connectErr.AddDetail(detailAny)
	}

	return connectErr
}

// mapToGRPCCode maps our error code ranges to appropriate gRPC status codes.
func mapToGRPCCode(code v1.ErrorCode) connect.Code {
	codeInt := int32(code)
	switch {
	case codeInt >= 100 && codeInt < 200:
		return connect.CodeUnauthenticated
	case codeInt >= 200 && codeInt < 300:
		return connect.CodePermissionDenied
	case codeInt >= 300 && codeInt < 400:
		return connect.CodeInvalidArgument
	case codeInt >= 400 && codeInt < 410:
		return connect.CodeNotFound
	case codeInt >= 410 && codeInt < 500:
		return connect.CodeAlreadyExists
	case codeInt >= 500 && codeInt < 600:
		return connect.CodeFailedPrecondition
	case codeInt >= 600 && codeInt < 700:
		return connect.CodeResourceExhausted
	case codeInt >= 700 && codeInt < 800:
		return connect.CodeInvalidArgument // Attachment errors are validation
	case codeInt >= 800 && codeInt < 900:
		return connect.CodeUnauthenticated // Token errors
	default:
		return connect.CodeInternal
	}
}

// HandleError is a convenience wrapper for handler functions.
// Use at the end of handlers: return nil, connect.HandleError(err)
func HandleError(err error) error {
	if err == nil {
		return nil
	}
	return ToConnectError(err)
}
