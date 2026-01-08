package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/emailapi/api/internal/domain"
)

const (
	// Default token expiry (30 days)
	defaultTokenExpiry = 30 * 24 * time.Hour
)

// UnsubscribeTokenData contains the data embedded in the unsubscribe token.
type UnsubscribeTokenData struct {
	UserID    string `json:"u"` // API user who sent the email
	Email     string `json:"e"` // Recipient email (plaintext for display, hashed in storage)
	EmailID   string `json:"i"` // Originating email ID (optional)
	ExpiresAt int64  `json:"x"` // Unix timestamp expiry
}

// UnsubscribeTokenService handles secure token encoding and decoding.
// Uses HMAC-SHA256 for fast verification (O(1) without network calls).
type UnsubscribeTokenService struct {
	secret []byte
	expiry time.Duration
}

// NewUnsubscribeTokenService creates a new token service.
func NewUnsubscribeTokenService(secret string) *UnsubscribeTokenService {
	return &UnsubscribeTokenService{
		secret: []byte(secret),
		expiry: defaultTokenExpiry,
	}
}

// Encode creates a URL-safe token from the data.
// Format: base64url(json_payload).base64url(hmac_signature)
// This is lightweight (~200 bytes) and fast to verify.
func (s *UnsubscribeTokenService) Encode(data *UnsubscribeTokenData) (string, error) {
	// Set expiry if not set
	if data.ExpiresAt == 0 {
		data.ExpiresAt = time.Now().Add(s.expiry).Unix()
	}

	// Marshal to JSON (compact)
	payload, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token data: %w", err)
	}

	// Base64url encode payload
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)

	// Compute HMAC-SHA256 signature
	signature := s.computeHMAC(payload)
	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)

	// Combine: payload.signature
	return encodedPayload + "." + encodedSignature, nil
}

// Decode verifies and decodes a token.
// Returns the token data if valid, or an error.
// Verification is O(1) - no network calls, just HMAC computation.
func (s *UnsubscribeTokenService) Decode(tokenStr string) (*UnsubscribeTokenData, error) {
	// Split token
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 2 {
		return nil, domain.ErrInvalidToken
	}

	encodedPayload, encodedSignature := parts[0], parts[1]

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Decode signature
	signature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Verify HMAC signature (constant-time comparison)
	expectedSignature := s.computeHMAC(payload)
	if !hmac.Equal(signature, expectedSignature) {
		return nil, domain.ErrInvalidToken
	}

	// Unmarshal payload
	var data UnsubscribeTokenData
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Check expiry
	if time.Now().Unix() > data.ExpiresAt {
		return nil, domain.ErrExpiredToken
	}

	return &data, nil
}

// computeHMAC computes HMAC-SHA256 of the payload.
func (s *UnsubscribeTokenService) computeHMAC(payload []byte) []byte {
	h := hmac.New(sha256.New, s.secret)
	h.Write(payload)
	return h.Sum(nil)
}

// GenerateLink creates an unsubscribe URL.
func (s *UnsubscribeTokenService) GenerateLink(baseURL, userID, email, emailID string) (string, error) {
	token, err := s.Encode(&UnsubscribeTokenData{
		UserID:  userID,
		Email:   email,
		EmailID: emailID,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/unsubscribe?token=%s", baseURL, token), nil
}
