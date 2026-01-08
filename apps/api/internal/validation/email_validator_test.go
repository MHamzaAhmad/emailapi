package validation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/validation/mocks"
)

func TestEmailValidator_ValidateSendEmail(t *testing.T) {
	t.Run("happy path - all validations pass", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDomainChecker := mocks.NewMockDomainChecker(ctrl)
		mockSuppression := mocks.NewMockSuppressionChecker(ctrl)
		mockMXCache := mocks.NewMockMXCache(ctrl)
		mockReputation := mocks.NewMockReputationChecker(ctrl)

		// Domain is owned and verified
		mockDomainChecker.EXPECT().
			GetVerifiedDomainForSending(gomock.Any(), "user_1", "example.com").
			Return(&domain.SendingDomain{VerifiedForSending: true}, nil)

		// No suppressed emails
		mockSuppression.EXPECT().
			CheckBatch(gomock.Any(), gomock.Any()).
			Return([]string{}, nil)

		// Reputation check passes
		mockReputation.EXPECT().
			CheckSendPermission(gomock.Any(), "user_1").
			Return(nil)

		// MX records exist (cached)
		hasMX := true
		mockMXCache.EXPECT().
			HasMX(gomock.Any(), gomock.Any()).
			Return(&hasMX, nil).
			AnyTimes()

		v := NewEmailValidator(mockDomainChecker, mockSuppression, nil, mockMXCache, mockReputation, "", nil)
		err := v.ValidateSendEmail(context.Background(), "user_1", "sender@example.com", []string{"recipient@gmail.com"}, nil, nil, "Hello", "")

		assert.NoError(t, err)
	})

	t.Run("reputation suspended blocks sending", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockReputation := mocks.NewMockReputationChecker(ctrl)

		// User is suspended
		mockReputation.EXPECT().
			CheckSendPermission(gomock.Any(), "user_1").
			Return(errors.New("account suspended"))

		v := NewEmailValidator(nil, nil, nil, nil, mockReputation, "", nil)
		err := v.ValidateSendEmail(context.Background(), "user_1", "sender@example.com", []string{"recipient@gmail.com"}, nil, nil, "", "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "suspended")
	})

	t.Run("sandbox recipient mismatch", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserCache := mocks.NewMockUserCache(ctrl)
		mockReputation := mocks.NewMockReputationChecker(ctrl)

		// Reputation check passes
		mockReputation.EXPECT().
			CheckSendPermission(gomock.Any(), "user_1").
			Return(nil)

		// User has verified email
		mockUserCache.EXPECT().
			GetByID(gomock.Any(), "user_1").
			Return(&domain.User{Email: "user@verified.com"}, nil)

		v := NewEmailValidator(nil, nil, nil, nil, mockReputation, "sandbox.example.com", mockUserCache)
		// Sending from sandbox to a different email than user's verified email
		err := v.ValidateSendEmail(context.Background(), "user_1", "sender@sandbox.example.com", []string{"other@gmail.com"}, nil, nil, "", "")

		require.Error(t, err)
		ve, ok := err.(*ValidationErrors)
		require.True(t, ok)
		assert.Equal(t, ErrCodeSandboxRestriction, ve.Errors[0].Code)
	})

	t.Run("sandbox recipient matches user email", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserCache := mocks.NewMockUserCache(ctrl)
		mockReputation := mocks.NewMockReputationChecker(ctrl)
		mockDomainChecker := mocks.NewMockDomainChecker(ctrl)
		mockMXCache := mocks.NewMockMXCache(ctrl)

		// Reputation check passes
		mockReputation.EXPECT().
			CheckSendPermission(gomock.Any(), "user_1").
			Return(nil)

		// User has verified email
		mockUserCache.EXPECT().
			GetByID(gomock.Any(), "user_1").
			Return(&domain.User{Email: "user@verified.com"}, nil)

		// MX records exist
		hasMX := true
		mockMXCache.EXPECT().
			HasMX(gomock.Any(), gomock.Any()).
			Return(&hasMX, nil).
			AnyTimes()

		v := NewEmailValidator(mockDomainChecker, nil, nil, mockMXCache, mockReputation, "sandbox.example.com", mockUserCache)
		// Sending from sandbox to user's own verified email
		err := v.ValidateSendEmail(context.Background(), "user_1", "sender@sandbox.example.com", []string{"user@verified.com"}, nil, nil, "", "")

		assert.NoError(t, err)
	})

	t.Run("domain not owned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDomainChecker := mocks.NewMockDomainChecker(ctrl)
		mockMXCache := mocks.NewMockMXCache(ctrl)
		mockReputation := mocks.NewMockReputationChecker(ctrl)

		// Reputation check passes
		mockReputation.EXPECT().
			CheckSendPermission(gomock.Any(), "user_1").
			Return(nil)

		// Domain not found
		mockDomainChecker.EXPECT().
			GetVerifiedDomainForSending(gomock.Any(), "user_1", "notowned.com").
			Return(nil, errors.New("domain not found"))

		// MX records exist
		hasMX := true
		mockMXCache.EXPECT().
			HasMX(gomock.Any(), gomock.Any()).
			Return(&hasMX, nil).
			AnyTimes()

		v := NewEmailValidator(mockDomainChecker, nil, nil, mockMXCache, mockReputation, "", nil)
		err := v.ValidateSendEmail(context.Background(), "user_1", "sender@notowned.com", []string{"recipient@gmail.com"}, nil, nil, "", "")

		require.Error(t, err)
		ve, ok := err.(*ValidationErrors)
		require.True(t, ok)
		assert.Equal(t, ErrCodeDomainNotOwned, ve.Errors[0].Code)
	})

	t.Run("domain not verified for sending", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDomainChecker := mocks.NewMockDomainChecker(ctrl)
		mockMXCache := mocks.NewMockMXCache(ctrl)
		mockReputation := mocks.NewMockReputationChecker(ctrl)

		// Reputation check passes
		mockReputation.EXPECT().
			CheckSendPermission(gomock.Any(), "user_1").
			Return(nil)

		// Domain owned but not verified for sending
		mockDomainChecker.EXPECT().
			GetVerifiedDomainForSending(gomock.Any(), "user_1", "unverified.com").
			Return(&domain.SendingDomain{VerifiedForSending: false}, nil)

		// MX records exist
		hasMX := true
		mockMXCache.EXPECT().
			HasMX(gomock.Any(), gomock.Any()).
			Return(&hasMX, nil).
			AnyTimes()

		v := NewEmailValidator(mockDomainChecker, nil, nil, mockMXCache, mockReputation, "", nil)
		err := v.ValidateSendEmail(context.Background(), "user_1", "sender@unverified.com", []string{"recipient@gmail.com"}, nil, nil, "", "")

		require.Error(t, err)
		ve, ok := err.(*ValidationErrors)
		require.True(t, ok)
		assert.Equal(t, ErrCodeDomainNotVerified, ve.Errors[0].Code)
	})

	t.Run("suppressed recipients", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDomainChecker := mocks.NewMockDomainChecker(ctrl)
		mockSuppression := mocks.NewMockSuppressionChecker(ctrl)
		mockMXCache := mocks.NewMockMXCache(ctrl)
		mockReputation := mocks.NewMockReputationChecker(ctrl)

		// Reputation check passes
		mockReputation.EXPECT().
			CheckSendPermission(gomock.Any(), "user_1").
			Return(nil)

		// Domain verified
		mockDomainChecker.EXPECT().
			GetVerifiedDomainForSending(gomock.Any(), "user_1", "example.com").
			Return(&domain.SendingDomain{VerifiedForSending: true}, nil)

		// One recipient is suppressed
		mockSuppression.EXPECT().
			CheckBatch(gomock.Any(), gomock.Any()).
			Return([]string{"suppressed_hash"}, nil)

		// MX records exist
		hasMX := true
		mockMXCache.EXPECT().
			HasMX(gomock.Any(), gomock.Any()).
			Return(&hasMX, nil).
			AnyTimes()

		v := NewEmailValidator(mockDomainChecker, mockSuppression, nil, mockMXCache, mockReputation, "", nil)
		err := v.ValidateSendEmail(context.Background(), "user_1", "sender@example.com", []string{"bounced@gmail.com"}, nil, nil, "", "")

		require.Error(t, err)
		ve, ok := err.(*ValidationErrors)
		require.True(t, ok)
		assert.Equal(t, ErrCodeEmailSuppressed, ve.Errors[0].Code)
	})
}

func TestValidateSender(t *testing.T) {
	t.Run("invalid email syntax", func(t *testing.T) {
		v := &EmailValidator{}
		err := v.validateSender(context.Background(), "user_1", "invalid-email")
		require.NotNil(t, err)
		assert.Equal(t, ErrCodeInvalidSyntax, err.Code)
	})

	t.Run("sandbox domain bypasses ownership check", func(t *testing.T) {
		v := &EmailValidator{sandboxDomain: "sandbox.test.com"}
		err := v.validateSender(context.Background(), "user_1", "sender@sandbox.test.com")
		assert.Nil(t, err)
	})
}

func TestFormatField(t *testing.T) {
	assert.Equal(t, "to[0]", formatField("to", 0))
	assert.Equal(t, "cc[1]", formatField("CC", 1))
	assert.Equal(t, "bcc[2]", formatField("BCC", 2))
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		email    string
		expected string
	}{
		{"user@example.com", "example.com"},
		{"USER@EXAMPLE.COM", "example.com"},
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractDomain(tt.email))
		})
	}
}
