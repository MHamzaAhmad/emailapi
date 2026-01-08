package validation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/external/webrisk"
	webriskMocks "github.com/emailapi/api/internal/external/webrisk/mocks"
	redisMocks "github.com/emailapi/api/internal/repository/redis/mocks"
)

func TestBodyValidator_ValidateURLs(t *testing.T) {
	t.Run("no URLs returns nil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := redisMocks.NewMockWebRiskCacheInterface(ctrl)
		mockWebRisk := webriskMocks.NewMockClient(ctrl)

		v := NewBodyValidator(mockCache, mockWebRisk)
		err := v.ValidateURLs(context.Background(), "Hello world, this is plain text", "<p>HTML without URLs</p>")

		assert.NoError(t, err)
	})

	t.Run("URLs not in cache returns nil (safe)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := redisMocks.NewMockWebRiskCacheInterface(ctrl)
		mockWebRisk := webriskMocks.NewMockClient(ctrl)

		// URLs not flagged in cache
		mockCache.EXPECT().
			CheckURLPrefixes(gomock.Any(), gomock.Any()).
			Return([]bool{false, false}, nil)

		v := NewBodyValidator(mockCache, mockWebRisk)
		err := v.ValidateURLs(context.Background(), "Check out https://example.com and https://google.com", "")

		assert.NoError(t, err)
	})

	t.Run("URLs in cache but cleared by API", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := redisMocks.NewMockWebRiskCacheInterface(ctrl)
		mockWebRisk := webriskMocks.NewMockClient(ctrl)

		// URL flagged in cache (potential threat)
		mockCache.EXPECT().
			CheckURLPrefixes(gomock.Any(), gomock.Any()).
			Return([]bool{true}, nil)

		// But API says it's safe
		mockWebRisk.EXPECT().
			Lookup(gomock.Any(), gomock.Any()).
			Return([]webrisk.ThreatMatch{}, nil)

		v := NewBodyValidator(mockCache, mockWebRisk)
		err := v.ValidateURLs(context.Background(), "Check out https://suspicious.com", "")

		assert.NoError(t, err)
	})

	t.Run("URLs flagged by Web Risk API", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := redisMocks.NewMockWebRiskCacheInterface(ctrl)
		mockWebRisk := webriskMocks.NewMockClient(ctrl)

		// URL flagged in cache
		mockCache.EXPECT().
			CheckURLPrefixes(gomock.Any(), gomock.Any()).
			Return([]bool{true}, nil)

		// API confirms threat
		mockWebRisk.EXPECT().
			Lookup(gomock.Any(), gomock.Any()).
			Return([]webrisk.ThreatMatch{
				{URL: "https://malware.com", ThreatType: webrisk.ThreatTypeMalware},
			}, nil)

		v := NewBodyValidator(mockCache, mockWebRisk)
		err := v.ValidateURLs(context.Background(), "Check out https://malware.com", "")

		require.Error(t, err)
		ve, ok := err.(*ValidationErrors)
		require.True(t, ok)
		assert.Len(t, ve.Errors, 1)
		assert.Equal(t, v1.ErrorCode_ERROR_CODE_UNSAFE_URL, ve.Errors[0].Code)
	})

	t.Run("cache error falls back to API check", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := redisMocks.NewMockWebRiskCacheInterface(ctrl)
		mockWebRisk := webriskMocks.NewMockClient(ctrl)

		// Cache error
		mockCache.EXPECT().
			CheckURLPrefixes(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("cache unavailable"))

		// Falls back to API (returns safe)
		mockWebRisk.EXPECT().
			Lookup(gomock.Any(), gomock.Any()).
			Return([]webrisk.ThreatMatch{}, nil)

		v := NewBodyValidator(mockCache, mockWebRisk)
		err := v.ValidateURLs(context.Background(), "https://example.com", "")

		assert.NoError(t, err)
	})

	t.Run("nil webrisk client skips API verification", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := redisMocks.NewMockWebRiskCacheInterface(ctrl)

		// Cache error triggers fallback, but no webrisk client
		mockCache.EXPECT().
			CheckURLPrefixes(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("cache error"))

		v := NewBodyValidator(mockCache, nil)
		err := v.ValidateURLs(context.Background(), "https://example.com", "")

		assert.NoError(t, err)
	})

	t.Run("API error is gracefully handled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := redisMocks.NewMockWebRiskCacheInterface(ctrl)
		mockWebRisk := webriskMocks.NewMockClient(ctrl)

		// URL flagged in cache
		mockCache.EXPECT().
			CheckURLPrefixes(gomock.Any(), gomock.Any()).
			Return([]bool{true}, nil)

		// API error - should not block
		mockWebRisk.EXPECT().
			Lookup(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("API unavailable"))

		v := NewBodyValidator(mockCache, mockWebRisk)
		err := v.ValidateURLs(context.Background(), "https://suspicious.com", "")

		assert.NoError(t, err) // Graceful degradation
	})
}

func TestExtractURLsFromText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"simple URL", "Visit https://example.com today!", 1},
		{"multiple URLs", "See https://a.com and https://b.com", 2},
		{"URL with query params", "Go to https://api.com/path?foo=bar", 1},
		{"trailing punctuation cleaned", "Click https://example.com.", 1},
		{"no URLs", "Just plain text here", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls := extractURLsFromText(tt.text)
			assert.Len(t, urls, tt.expected)
		})
	}
}

func TestExtractURLsFromHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected int
	}{
		{"href attribute", `<a href="https://example.com">Click</a>`, 1},
		{"src attribute", `<img src="https://cdn.com/img.png">`, 1},
		{"multiple links", `<a href="https://a.com">A</a><a href="https://b.com">B</a>`, 2},
		{"inline text URL", `<p>Visit https://inline.com today</p>`, 1},
		{"no URLs", `<p>Hello World</p>`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls := extractURLsFromHTML(tt.html)
			assert.GreaterOrEqual(t, len(urls), tt.expected)
		})
	}
}

func TestDeduplicate(t *testing.T) {
	urls := []string{"https://a.com", "https://b.com", "https://a.com", "https://c.com", "https://b.com"}
	result := deduplicate(urls)
	assert.Len(t, result, 3)
}

func TestComputeHashPrefix(t *testing.T) {
	// Hash prefix should be 4 bytes = 8 hex chars
	prefix := computeHashPrefix("https://example.com")
	assert.Len(t, prefix, 8)
}
