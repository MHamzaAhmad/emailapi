package errors

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/validation"
)

func TestToConnectError_ValidationErrors(t *testing.T) {
	t.Run("handles multiple validation errors", func(t *testing.T) {
		ve := validation.NewValidationErrors()
		ve.Add(validation.InvalidSyntaxError("to[0]", "invalid@"))
		ve.Add(validation.NoMXRecordsError("to[1]", "test@nodomain.fake", "nodomain.fake"))

		connectErr := ToConnectError(ve)

		assert.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
		// First error message is used as the main message
		assert.Contains(t, connectErr.Message(), "email address has invalid syntax")

		// Both errors should be in details
		require.Len(t, connectErr.Details(), 2)

		// Verify first detail
		detail1 := &v1.ErrorDetail{}
		err := proto.Unmarshal(connectErr.Details()[0].Bytes(), detail1)
		require.NoError(t, err)
		assert.Equal(t, v1.ErrorCode_ERROR_CODE_INVALID_EMAIL_SYNTAX, detail1.Code)
		assert.Equal(t, "to[0]", detail1.Field)
		assert.Equal(t, "invalid@", detail1.Metadata["email"])

		// Verify second detail
		detail2 := &v1.ErrorDetail{}
		err = proto.Unmarshal(connectErr.Details()[1].Bytes(), detail2)
		require.NoError(t, err)
		assert.Equal(t, v1.ErrorCode_ERROR_CODE_NO_MX_RECORDS, detail2.Code)
		assert.Equal(t, "to[1]", detail2.Field)
		assert.Equal(t, "nodomain.fake", detail2.Metadata["domain"])
	})

	t.Run("handles empty validation errors", func(t *testing.T) {
		ve := validation.NewValidationErrors()

		connectErr := ToConnectError(ve)

		assert.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
		assert.Equal(t, "validation failed", connectErr.Message())
		assert.Empty(t, connectErr.Details())
	})

	t.Run("handles single AppError", func(t *testing.T) {
		appErr := domain.ErrDomainNotVerified.Clone().WithField("from").WithMeta("domain", "example.com")

		connectErr := ToConnectError(appErr)

		assert.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
		require.Len(t, connectErr.Details(), 1)

		detail := &v1.ErrorDetail{}
		err := proto.Unmarshal(connectErr.Details()[0].Bytes(), detail)
		require.NoError(t, err)
		assert.Equal(t, v1.ErrorCode_ERROR_CODE_DOMAIN_NOT_VERIFIED, detail.Code)
		assert.Equal(t, "from", detail.Field)
		assert.Equal(t, "example.com", detail.Metadata["domain"])
	})

	t.Run("handles nil error", func(t *testing.T) {
		connectErr := ToConnectError(nil)
		assert.Nil(t, connectErr)
	})
}
