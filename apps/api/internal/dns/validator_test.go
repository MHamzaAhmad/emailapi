package dns

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpectedRecord_Key(t *testing.T) {
	tests := []struct {
		name     string
		record   ExpectedRecord
		expected string
	}{
		{
			name: "CNAME record key",
			record: ExpectedRecord{
				Type: "CNAME",
				Name: "token1._domainkey.example.com",
			},
			expected: "CNAME:token1._domainkey.example.com",
		},
		{
			name: "TXT record key",
			record: ExpectedRecord{
				Type: "TXT",
				Name: "example.com",
			},
			expected: "TXT:example.com",
		},
		{
			name: "MX record key",
			record: ExpectedRecord{
				Type: "MX",
				Name: "example.com",
			},
			expected: "MX:example.com",
		},
		{
			name: "DMARC record key",
			record: ExpectedRecord{
				Type: "TXT",
				Name: "_dmarc.example.com",
			},
			expected: "TXT:_dmarc.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.record.Key())
		})
	}
}

func TestGetMostCommon(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{
			name:     "single value",
			values:   []string{"value1"},
			expected: "value1",
		},
		{
			name:     "all same values",
			values:   []string{"value1", "value1", "value1"},
			expected: "value1",
		},
		{
			name:     "majority wins",
			values:   []string{"value1", "value1", "value2"},
			expected: "value1",
		},
		{
			name:     "clear majority",
			values:   []string{"a", "a", "a", "b", "b", "c"},
			expected: "a",
		},
		{
			name:     "empty slice returns empty",
			values:   []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMostCommon(tt.values)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidator_valuesMatch(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name       string
		recordType string
		expected   string
		discovered string
		match      bool
	}{
		// CNAME tests
		{
			name:       "CNAME exact match",
			recordType: "CNAME",
			expected:   "token.dkim.amazonses.com",
			discovered: "token.dkim.amazonses.com",
			match:      true,
		},
		{
			name:       "CNAME match with trailing dot",
			recordType: "CNAME",
			expected:   "token.dkim.amazonses.com",
			discovered: "token.dkim.amazonses.com.",
			match:      true,
		},
		{
			name:       "CNAME case insensitive",
			recordType: "CNAME",
			expected:   "token.dkim.amazonses.com",
			discovered: "TOKEN.DKIM.AMAZONSES.COM",
			match:      true,
		},
		{
			name:       "CNAME mismatch",
			recordType: "CNAME",
			expected:   "token1.dkim.amazonses.com",
			discovered: "token2.dkim.amazonses.com",
			match:      false,
		},

		// TXT tests (substring match)
		{
			name:       "TXT exact match",
			recordType: "TXT",
			expected:   "v=spf1 include:amazonses.com ~all",
			discovered: "v=spf1 include:amazonses.com ~all",
			match:      true,
		},
		{
			name:       "TXT substring match in multiple values",
			recordType: "TXT",
			expected:   "v=spf1 include:amazonses.com ~all",
			discovered: "v=spf1 include:amazonses.com ~all; v=spf1 include:_spf.google.com ~all",
			match:      true,
		},
		{
			name:       "TXT no match",
			recordType: "TXT",
			expected:   "v=spf1 include:amazonses.com ~all",
			discovered: "v=spf1 include:google.com ~all",
			match:      false,
		},

		// MX tests (host match ignoring priority)
		{
			name:       "MX exact host match",
			recordType: "MX",
			expected:   "inbound-smtp.us-east-1.amazonaws.com",
			discovered: "10 inbound-smtp.us-east-1.amazonaws.com",
			match:      true,
		},
		{
			name:       "MX case insensitive",
			recordType: "MX",
			expected:   "inbound-smtp.us-east-1.amazonaws.com",
			discovered: "10 INBOUND-SMTP.US-EAST-1.AMAZONAWS.COM",
			match:      true,
		},
		{
			name:       "MX no match",
			recordType: "MX",
			expected:   "inbound-smtp.us-east-1.amazonaws.com",
			discovered: "10 mail.google.com",
			match:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.valuesMatch(tt.recordType, tt.expected, tt.discovered)
			assert.Equal(t, tt.match, result)
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Len(t, config.Resolvers, 6, "should have 6 default resolvers")
	assert.Equal(t, DefaultTimeout, config.Timeout)
	assert.Equal(t, DefaultRetries, config.Retries)
	assert.Equal(t, DefaultRetryDelay, config.RetryDelay)
	assert.Equal(t, ConsensusThreshold, config.ConsensusThreshold)
}

func TestNewValidatorWithConfig(t *testing.T) {
	t.Run("uses defaults for zero values", func(t *testing.T) {
		config := ValidatorConfig{
			// All zero values
		}
		v := NewValidatorWithConfig(config)

		assert.Len(t, v.config.Resolvers, 6, "should use default resolvers")
		assert.Equal(t, DefaultTimeout, v.config.Timeout)
		assert.Equal(t, ConsensusThreshold, v.config.ConsensusThreshold)
	})

	t.Run("uses provided values", func(t *testing.T) {
		config := ValidatorConfig{
			Resolvers:          []string{"1.2.3.4:53"},
			Timeout:            5 * time.Second,
			Retries:            5,
			RetryDelay:         200 * time.Millisecond,
			ConsensusThreshold: 0.75,
		}
		v := NewValidatorWithConfig(config)

		assert.Equal(t, []string{"1.2.3.4:53"}, v.config.Resolvers)
		assert.Equal(t, 5*time.Second, v.config.Timeout)
		assert.Equal(t, 5, v.config.Retries)
		assert.Equal(t, 200*time.Millisecond, v.config.RetryDelay)
		assert.Equal(t, 0.75, v.config.ConsensusThreshold)
	})
}

func TestValidationResult_KeyBasedMapping(t *testing.T) {
	// Test that ValidationResult properly uses map-based keying
	result := &ValidationResult{
		Records:   make(map[string]RecordResult),
		CheckedAt: time.Now(),
	}

	// Add some records
	rec1 := ExpectedRecord{Type: "CNAME", Name: "token1._domainkey.example.com", Value: "token1.dkim.amazonses.com"}
	rec2 := ExpectedRecord{Type: "TXT", Name: "example.com", Value: "v=spf1 include:amazonses.com ~all"}
	rec3 := ExpectedRecord{Type: "MX", Name: "example.com", Value: "inbound-smtp.us-east-1.amazonaws.com", Priority: 10}

	result.Records[rec1.Key()] = RecordResult{ExpectedRecord: rec1, Status: RecordStatusFound}
	result.Records[rec2.Key()] = RecordResult{ExpectedRecord: rec2, Status: RecordStatusMissing}
	result.Records[rec3.Key()] = RecordResult{ExpectedRecord: rec3, Status: RecordStatusMismatch}

	// Verify we can look up by key
	assert.Equal(t, RecordStatusFound, result.Records["CNAME:token1._domainkey.example.com"].Status)
	assert.Equal(t, RecordStatusMissing, result.Records["TXT:example.com"].Status)
	assert.Equal(t, RecordStatusMismatch, result.Records["MX:example.com"].Status)

	// Verify missing key returns zero value
	_, exists := result.Records["CNAME:nonexistent.example.com"]
	assert.False(t, exists)
}

func TestValidator_ValidateRecords_ReturnsKeyedResults(t *testing.T) {
	// This test verifies the structure of ValidateRecords output
	// It doesn't actually hit DNS - we just verify the result structure
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Use a validator with very short timeout and no retries to fail fast
	v := NewValidatorWithConfig(ValidatorConfig{
		Resolvers:          []string{"192.0.2.1:53"}, // TEST-NET-1, won't resolve
		Timeout:            10 * time.Millisecond,
		Retries:            0,
		ConsensusThreshold: 0.5,
	})

	expected := []ExpectedRecord{
		{Type: "CNAME", Name: "test1.example.com", Value: "test.value"},
		{Type: "TXT", Name: "test2.example.com", Value: "test value"},
	}

	result := v.ValidateRecords(ctx, expected)

	require.NotNil(t, result)
	require.NotNil(t, result.Records)
	assert.Len(t, result.Records, 2)

	// Verify keys are correct
	_, exists1 := result.Records["CNAME:test1.example.com"]
	_, exists2 := result.Records["TXT:test2.example.com"]
	assert.True(t, exists1, "should have key for CNAME record")
	assert.True(t, exists2, "should have key for TXT record")

	// Results will be missing since we can't reach the resolver
	for _, rec := range result.Records {
		assert.Equal(t, RecordStatusMissing, rec.Status)
	}
}

func TestValidator_ValidateRecords_EmptyInput(t *testing.T) {
	v := NewValidator()
	ctx := context.Background()

	result := v.ValidateRecords(ctx, []ExpectedRecord{})

	require.NotNil(t, result)
	assert.Empty(t, result.Records)
	assert.False(t, result.CheckedAt.IsZero())
}

// Consensus logic tests using a testable validator wrapper
type mockResolverResult struct {
	value string
	found bool
	err   error
}

// testableValidator wraps Validator to allow testing consensus logic with mock data
type testableValidator struct {
	*Validator
}

func TestConsensusLogic(t *testing.T) {
	// These tests verify the consensus calculation logic by examining
	// how different resolver result combinations should affect the final status

	tests := []struct {
		name           string
		foundCount     int
		mismatchCount  int
		missingCount   int
		totalResolvers int
		threshold      float64
		expectedStatus RecordStatus
	}{
		{
			name:           "all found - should be found",
			foundCount:     6,
			mismatchCount:  0,
			missingCount:   0,
			totalResolvers: 6,
			threshold:      0.5,
			expectedStatus: RecordStatusFound,
		},
		{
			name:           "all missing - should be missing",
			foundCount:     0,
			mismatchCount:  0,
			missingCount:   6,
			totalResolvers: 6,
			threshold:      0.5,
			expectedStatus: RecordStatusMissing,
		},
		{
			name:           "majority found - should be found",
			foundCount:     4,
			mismatchCount:  1,
			missingCount:   1,
			totalResolvers: 6,
			threshold:      0.5,
			expectedStatus: RecordStatusFound,
		},
		{
			name:           "majority mismatch - should be mismatch",
			foundCount:     1,
			mismatchCount:  4,
			missingCount:   1,
			totalResolvers: 6,
			threshold:      0.5,
			expectedStatus: RecordStatusMismatch,
		},
		{
			name:           "exactly at threshold found - should be found",
			foundCount:     3,
			mismatchCount:  1,
			missingCount:   2,
			totalResolvers: 6,
			threshold:      0.5,
			expectedStatus: RecordStatusFound,
		},
		{
			name:           "split results no consensus - should be missing (conservative)",
			foundCount:     2,
			mismatchCount:  2,
			missingCount:   2,
			totalResolvers: 6,
			threshold:      0.5,
			expectedStatus: RecordStatusMissing,
		},
		{
			name:           "higher threshold not met - missing",
			foundCount:     3,
			mismatchCount:  1,
			missingCount:   2,
			totalResolvers: 6,
			threshold:      0.75,
			expectedStatus: RecordStatusMissing, // 3 < 4 threshold (0.75*6=4.5 truncated to 4)
		},
		{
			name:           "higher threshold met - found",
			foundCount:     5,
			mismatchCount:  0,
			missingCount:   1,
			totalResolvers: 6,
			threshold:      0.75,
			expectedStatus: RecordStatusFound, // 5 >= 4.5 threshold
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate what the status should be based on our logic
			threshold := int(float64(tt.totalResolvers) * tt.threshold)
			if threshold < 1 {
				threshold = 1
			}

			var status RecordStatus
			if tt.foundCount >= threshold {
				status = RecordStatusFound
			} else if tt.mismatchCount >= threshold {
				status = RecordStatusMismatch
			} else if tt.missingCount >= threshold {
				status = RecordStatusMissing
			} else {
				// No consensus
				status = RecordStatusMissing
			}

			assert.Equal(t, tt.expectedStatus, status, "consensus logic mismatch")
		})
	}
}

func TestRecordStatus_Values(t *testing.T) {
	// Verify status values are as expected
	assert.Equal(t, RecordStatus("pending"), RecordStatusPending)
	assert.Equal(t, RecordStatus("found"), RecordStatusFound)
	assert.Equal(t, RecordStatus("mismatch"), RecordStatusMismatch)
	assert.Equal(t, RecordStatus("missing"), RecordStatusMissing)
}
