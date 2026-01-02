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
				// Build map-based results keyed by Type:Name
				records := make(map[string]dns.RecordResult)
				for _, exp := range expected {
					records[exp.Key()] = dns.RecordResult{
						ExpectedRecord:  exp,
						Status:          dns.RecordStatusFound,
						DiscoveredValue: exp.Value,
					}
				}
				return &dns.ValidationResult{
					Records:   records,
					CheckedAt: time.Now(),
				}
			})

		result, err := svc.Verify(ctx, userID, domainID)
		require.NoError(t, err)
		assert.True(t, result.WasRefreshed)
		assert.True(t, result.Domain.Summary.CanSend)
	})
}

func TestDomainService_Verify_Cooldown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockDomainRepo := repoMocks.NewMockDomainRepository(ctrl)
	mockSES := sesMocks.NewMockClient(ctrl)

	mockStore.EXPECT().Domains().Return(mockDomainRepo).AnyTimes()

	svc := NewDomainService(mockStore, mockSES, nil, nil, nil, nil, "us-east-1", "config-set")
	ctx := context.Background()
	userID := "user_1"
	domainID := "dom_1"

	t.Run("respects cooldown period", func(t *testing.T) {
		// LastCheckedAt is very recent (within cooldown)
		recentCheck := time.Now().Add(-10 * time.Second) // Only 10s ago
		storedDomain := &domain.SendingDomain{
			ID:            domainID,
			UserID:        userID,
			Domain:        "example.com",
			DkimTokens:    []string{"token1"},
			LastCheckedAt: &recentCheck,
		}

		mockDomainRepo.EXPECT().
			GetByID(ctx, domainID).
			Return(storedDomain, nil)

		// No SES call should be made due to cooldown

		result, err := svc.Verify(ctx, userID, domainID)
		require.NoError(t, err)
		assert.False(t, result.WasRefreshed)
		assert.NotNil(t, result.NextRetryAt)
		assert.Contains(t, result.Message, "Rate limited")
	})

	t.Run("allows verify after cooldown expires", func(t *testing.T) {
		mockStore2 := serviceMocks.NewMockStore(ctrl)
		mockDomainRepo2 := repoMocks.NewMockDomainRepository(ctrl)
		mockSES2 := sesMocks.NewMockClient(ctrl)
		mockDNS := dnsMocks.NewMockValidatorInterface(ctrl)

		mockStore2.EXPECT().Domains().Return(mockDomainRepo2).AnyTimes()

		svc2 := NewDomainService(mockStore2, mockSES2, mockDNS, nil, nil, nil, "us-east-1", "config-set")

		// LastCheckedAt is beyond cooldown
		oldCheck := time.Now().Add(-35 * time.Second) // 35s ago, beyond 30s cooldown
		storedDomain := &domain.SendingDomain{
			ID:            domainID,
			UserID:        userID,
			Domain:        "example.com",
			DkimTokens:    []string{"token1"},
			LastCheckedAt: &oldCheck,
		}

		mockDomainRepo2.EXPECT().
			GetByID(ctx, domainID).
			Return(storedDomain, nil)

		mockSES2.EXPECT().
			GetEmailIdentity(ctx, "example.com").
			Return(&ses.IdentityResult{
				VerifiedForSendingStatus: true,
				DkimStatus:               "SUCCESS",
				DkimTokens:               []string{"token1"},
			}, nil)

		mockDomainRepo2.EXPECT().
			Update(ctx, gomock.Any()).
			Return(nil).Times(2)

		mockDNS.EXPECT().
			ValidateRecords(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, expected []dns.ExpectedRecord) *dns.ValidationResult {
				records := make(map[string]dns.RecordResult)
				for _, exp := range expected {
					records[exp.Key()] = dns.RecordResult{
						ExpectedRecord:  exp,
						Status:          dns.RecordStatusFound,
						DiscoveredValue: exp.Value,
					}
				}
				return &dns.ValidationResult{Records: records, CheckedAt: time.Now()}
			})

		result, err := svc2.Verify(ctx, userID, domainID)
		require.NoError(t, err)
		assert.True(t, result.WasRefreshed)
	})
}

func TestDomainService_Verify_CacheInvalidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockCache := serviceMocks.NewMockCache(ctrl)
	mockSES := sesMocks.NewMockClient(ctrl)
	mockDNS := dnsMocks.NewMockValidatorInterface(ctrl)
	mockDomainRepo := repoMocks.NewMockDomainRepository(ctrl)
	mockDomainCache := redisMocks.NewMockDomainCacheInterface(ctrl)

	mockStore.EXPECT().Domains().Return(mockDomainRepo).AnyTimes()
	mockCache.EXPECT().Domain().Return(mockDomainCache).AnyTimes()

	svc := NewDomainService(mockStore, mockSES, mockDNS, mockCache, nil, nil, "us-east-1", "config-set")
	ctx := context.Background()
	userID := "user_1"
	domainID := "dom_1"
	domainName := "example.com"

	t.Run("invalidates cache after verify", func(t *testing.T) {
		oldCheck := time.Now().Add(-35 * time.Second)
		storedDomain := &domain.SendingDomain{
			ID:            domainID,
			UserID:        userID,
			Domain:        domainName,
			DkimTokens:    []string{"token1"},
			LastCheckedAt: &oldCheck,
		}

		mockDomainRepo.EXPECT().
			GetByID(ctx, domainID).
			Return(storedDomain, nil)

		mockSES.EXPECT().
			GetEmailIdentity(ctx, domainName).
			Return(&ses.IdentityResult{
				VerifiedForSendingStatus: true,
				DkimStatus:               "SUCCESS",
				DkimTokens:               []string{"token1"},
			}, nil)

		mockDomainRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(nil).Times(2)

		mockDNS.EXPECT().
			ValidateRecords(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, expected []dns.ExpectedRecord) *dns.ValidationResult {
				records := make(map[string]dns.RecordResult)
				for _, exp := range expected {
					records[exp.Key()] = dns.RecordResult{
						ExpectedRecord:  exp,
						Status:          dns.RecordStatusFound,
						DiscoveredValue: exp.Value,
					}
				}
				return &dns.ValidationResult{Records: records, CheckedAt: time.Now()}
			})

		// Key assertion: cache is invalidated after verify
		mockDomainCache.EXPECT().
			InvalidateAll(ctx, domainID, userID).
			Return(nil)

		result, err := svc.Verify(ctx, userID, domainID)
		require.NoError(t, err)
		assert.True(t, result.WasRefreshed)
	})
}

func TestDomainService_Verify_KeyBasedRecordMapping(t *testing.T) {
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

	t.Run("correctly maps records by key with mixed statuses", func(t *testing.T) {
		oldCheck := time.Now().Add(-35 * time.Second)
		storedDomain := &domain.SendingDomain{
			ID:             domainID,
			UserID:         userID,
			Domain:         domainName,
			DkimTokens:     []string{"token1", "token2"},
			MailFromDomain: "mail.example.com",
			LastCheckedAt:  &oldCheck,
		}

		mockDomainRepo.EXPECT().
			GetByID(ctx, domainID).
			Return(storedDomain, nil)

		mockSES.EXPECT().
			GetEmailIdentity(ctx, domainName).
			Return(&ses.IdentityResult{
				VerifiedForSendingStatus: true,
				DkimStatus:               "SUCCESS",
				DkimTokens:               []string{"token1", "token2"},
				MailFromDomain:           "mail.example.com",
				MailFromStatus:           "SUCCESS",
			}, nil)

		mockDomainRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(nil).Times(2)

		// Return mixed statuses - key-based mapping ensures correct assignment
		mockDNS.EXPECT().
			ValidateRecords(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, expected []dns.ExpectedRecord) *dns.ValidationResult {
				records := make(map[string]dns.RecordResult)
				for _, exp := range expected {
					var status dns.RecordStatus
					// Assign different statuses based on record type
					switch {
					case exp.Type == "CNAME": // DKIM
						status = dns.RecordStatusFound
					case exp.Type == "TXT" && exp.Name == domainName: // SPF
						status = dns.RecordStatusMissing
					case exp.Type == "TXT" && exp.Name == "_dmarc."+domainName: // DMARC
						status = dns.RecordStatusMismatch
					case exp.Type == "MX" && exp.Name == domainName: // MX Inbound
						status = dns.RecordStatusFound
					default: // MailFrom records
						status = dns.RecordStatusFound
					}
					records[exp.Key()] = dns.RecordResult{
						ExpectedRecord:  exp,
						Status:          status,
						DiscoveredValue: exp.Value,
					}
				}
				return &dns.ValidationResult{Records: records, CheckedAt: time.Now()}
			})

		result, err := svc.Verify(ctx, userID, domainID)
		require.NoError(t, err)

		// Verify records have correct statuses from key-based lookup
		records := result.Domain.Records
		require.NotNil(t, records)

		// DKIM records should be found
		for _, dkim := range records.DkimRecords {
			assert.Equal(t, domain.RecordStatusFound, dkim.Status, "DKIM should be found")
		}

		// SPF should be missing
		require.NotNil(t, records.SpfRecord)
		assert.Equal(t, domain.RecordStatusMissing, records.SpfRecord.Status, "SPF should be missing")

		// DMARC should be mismatch
		require.NotNil(t, records.DmarcRecord)
		assert.Equal(t, domain.RecordStatusMismatch, records.DmarcRecord.Status, "DMARC should be mismatch")

		// MX should be found
		require.NotEmpty(t, records.MxRecords)
		assert.Equal(t, domain.RecordStatusFound, records.MxRecords[0].Status, "MX should be found")

		// MailFrom records should be found
		for _, mf := range records.MailFromRecords {
			assert.Equal(t, domain.RecordStatusFound, mf.Status, "MailFrom should be found")
		}
	})
}

func TestDomainService_Verify_StatusTransitions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name            string
		verifiedSending bool
		allRecordsFound bool
		expectedStatus  domain.DomainStatus
		expectedCanSend bool
	}{
		{
			name:            "all verified - DomainStatusReady",
			verifiedSending: true,
			allRecordsFound: true,
			expectedStatus:  domain.DomainStatusReady,
			expectedCanSend: true,
		},
		{
			name:            "verified but records pending - DomainStatusDegraded",
			verifiedSending: true,
			allRecordsFound: false,
			expectedStatus:  domain.DomainStatusDegraded,
			expectedCanSend: true,
		},
		{
			name:            "not verified but some records - DomainStatusVerifying",
			verifiedSending: false,
			allRecordsFound: false,
			expectedStatus:  domain.DomainStatusVerifying,
			expectedCanSend: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			oldCheck := time.Now().Add(-35 * time.Second)
			storedDomain := &domain.SendingDomain{
				ID:            domainID,
				UserID:        userID,
				Domain:        domainName,
				DkimTokens:    []string{"token1"},
				LastCheckedAt: &oldCheck,
			}

			mockDomainRepo.EXPECT().
				GetByID(ctx, domainID).
				Return(storedDomain, nil)

			mockSES.EXPECT().
				GetEmailIdentity(ctx, domainName).
				Return(&ses.IdentityResult{
					VerifiedForSendingStatus: tt.verifiedSending,
					DkimStatus:               "SUCCESS",
					DkimTokens:               []string{"token1"},
				}, nil)

			mockDomainRepo.EXPECT().
				Update(ctx, gomock.Any()).
				Return(nil).Times(2)

			mockDNS.EXPECT().
				ValidateRecords(ctx, gomock.Any()).
				DoAndReturn(func(_ context.Context, expected []dns.ExpectedRecord) *dns.ValidationResult {
					records := make(map[string]dns.RecordResult)
					for _, exp := range expected {
						status := dns.RecordStatusMissing
						if tt.allRecordsFound {
							status = dns.RecordStatusFound
						} else if exp.Type == "CNAME" {
							// At least DKIM is found for "verifying" status
							status = dns.RecordStatusFound
						}
						records[exp.Key()] = dns.RecordResult{
							ExpectedRecord:  exp,
							Status:          status,
							DiscoveredValue: exp.Value,
						}
					}
					return &dns.ValidationResult{Records: records, CheckedAt: time.Now()}
				})

			result, err := svc.Verify(ctx, userID, domainID)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, result.Domain.Status, "domain status mismatch")
			assert.Equal(t, tt.expectedCanSend, result.Domain.Summary.CanSend, "canSend mismatch")
		})
	}
}

func TestDomainService_Verify_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockDomainRepo := repoMocks.NewMockDomainRepository(ctrl)

	mockStore.EXPECT().Domains().Return(mockDomainRepo).AnyTimes()

	svc := NewDomainService(mockStore, nil, nil, nil, nil, nil, "us-east-1", "config-set")
	ctx := context.Background()

	t.Run("returns error for unauthorized user", func(t *testing.T) {
		storedDomain := &domain.SendingDomain{
			ID:     "dom_1",
			UserID: "other_user", // Different user
			Domain: "example.com",
		}

		mockDomainRepo.EXPECT().
			GetByID(ctx, "dom_1").
			Return(storedDomain, nil)

		_, err := svc.Verify(ctx, "user_1", "dom_1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
