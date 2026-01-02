package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/dns"
	dnsMocks "github.com/emailapi/api/internal/dns/mocks"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/ses"
	sesMocks "github.com/emailapi/api/internal/external/ses/mocks"
	repoMocks "github.com/emailapi/api/internal/repository/postgres/mocks"
	redisMocks "github.com/emailapi/api/internal/repository/redis/mocks"
	tinybirdMocks "github.com/emailapi/api/internal/repository/tinybird/mocks"
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
	validationMocks "github.com/emailapi/api/internal/validation/mocks"
)

func TestDomainService_Add(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockCache := serviceMocks.NewMockCache(ctrl)
	mockAnalytics := serviceMocks.NewMockAnalytics(ctrl)
	mockSES := sesMocks.NewMockClient(ctrl)
	mockReputation := validationMocks.NewMockReputationChecker(ctrl)
	mockDomainRepo := repoMocks.NewMockDomainRepository(ctrl)
	mockDomainCache := redisMocks.NewMockDomainCacheInterface(ctrl)
	mockActivityRepo := tinybirdMocks.NewMockActivityRepositoryInterface(ctrl)

	mockStore.EXPECT().Domains().Return(mockDomainRepo).AnyTimes()
	mockCache.EXPECT().Domain().Return(mockDomainCache).AnyTimes()
	mockAnalytics.EXPECT().Activity().Return(mockActivityRepo).AnyTimes()

	// Use nil DNS validator for Add tests as it's not used
	svc := NewDomainService(mockStore, mockSES, nil, mockCache, mockAnalytics, mockReputation, "us-east-1", "config-set")
	ctx := context.Background()
	userID := "user_1"
	domainName := "example.com"

	t.Run("success", func(t *testing.T) {
		mockReputation.EXPECT().
			CheckSendPermission(ctx, userID).
			Return(nil)

		mockDomainRepo.EXPECT().
			GetByDomainName(ctx, userID, domainName).
			Return(nil, nil)

		mockSES.EXPECT().
			CreateEmailIdentity(ctx, domainName).
			Return(&ses.IdentityResult{
				VerifiedForSendingStatus: false,
				DkimTokens:               []string{"token1", "token2"},
				DkimStatus:               "PENDING",
			}, nil)

		mockSES.EXPECT().
			PutEmailIdentityMailFromAttributes(ctx, domainName, "mail.example.com").
			Return(nil)

		mockSES.EXPECT().
			PutEmailIdentityConfigurationSetAttributes(ctx, domainName, "config-set").
			Return(nil)

		mockDomainRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, d *domain.SendingDomain) error {
				assert.Equal(t, domainName, d.Domain)
				assert.Equal(t, userID, d.UserID)
				assert.Equal(t, 2, len(d.DkimTokens))
				return nil
			})

		mockDomainCache.EXPECT().
			InvalidateByUserID(ctx, userID).
			Return(nil)

		mockActivityRepo.EXPECT().
			LogDomain(ctx, userID, gomock.Any(), "create", "success", gomock.Any()).
			Return(nil)

		d, err := svc.Add(ctx, userID, domainName)
		require.NoError(t, err)
		assert.Equal(t, domainName, d.Domain)
	})
}

func TestDomainService_Verify(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockSES := sesMocks.NewMockClient(ctrl)
	mockDNS := dnsMocks.NewMockValidatorInterface(ctrl)
	mockDomainRepo := repoMocks.NewMockDomainRepository(ctrl)

	mockStore.EXPECT().Domains().Return(mockDomainRepo).AnyTimes()

	svc := NewDomainService(mockStore, mockSES, mockDNS, nil, nil, nil, "us-east-1", "config-set")
	ctx := context.Background()
	userID := "user_1"
	domainID := "dom_1"
	domainName := "example.com"

	t.Run("success", func(t *testing.T) {
		now := time.Now()
		storedDomain := &domain.SendingDomain{
			ID:            domainID,
			UserID:        userID,
			Domain:        domainName,
			DkimTokens:    []string{"token1"},
			LastCheckedAt: &now, // Just checked, but Verify forces check? Check cooldown.
		}
		// VerifyCooldown is 30s. If we set LastCheckedAt to Now - 31s, it should allow.
		lastChecked := now.Add(-31 * time.Second)
		storedDomain.LastCheckedAt = &lastChecked

		mockDomainRepo.EXPECT().
			GetByID(ctx, domainID).
			Return(storedDomain, nil)

		// Refresh from SES
		mockSES.EXPECT().
			GetEmailIdentity(ctx, domainName).
			Return(&ses.IdentityResult{
				VerifiedForSendingStatus: true,
				DkimStatus:               "SUCCESS",
				DkimTokens:               []string{"token1"},
			}, nil)

		mockDomainRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(nil).Times(2) // Once for refresh, once for status update after DNS

		// DNS Validation
		mockDNS.EXPECT().
			ValidateRecords(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, expected []dns.ExpectedRecord) *dns.ValidationResult {
				assert.NotEmpty(t, expected)
				return &dns.ValidationResult{
					Records: []dns.RecordResult{
						{Status: dns.RecordStatusFound}, // DKIM
						{Status: dns.RecordStatusFound}, // SPF
						{Status: dns.RecordStatusFound}, // DMARC
						{Status: dns.RecordStatusFound}, // MX
						// MailFrom records if any... storedDomain has none by default
					},
					CheckedAt: time.Now(),
				}
			})

		result, err := svc.Verify(ctx, userID, domainID)
		require.NoError(t, err)
		assert.True(t, result.WasRefreshed)
		assert.True(t, result.Domain.Summary.CanSend)
	})
}
