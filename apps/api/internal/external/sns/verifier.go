package sns

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Verifier validates AWS SNS message signatures following AWS documentation.
// https://docs.aws.amazon.com/sns/latest/dg/sns-verify-signature-of-message.html
type Verifier struct {
	certCache sync.Map // map[string]*x509.Certificate
	client    *http.Client
}

// NewVerifier creates a new SNS signature verifier.
func NewVerifier() *Verifier {
	return &Verifier{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// VerifySignature validates an SNS message signature.
// Returns nil if valid, error if invalid or verification fails.
func (v *Verifier) VerifySignature(
	signingCertURL string,
	signature string,
	signatureVersion string,
	messageType string,
	message string,
	messageID string,
	timestamp string,
	topicArn string,
	subscribeURL string,
	subject string,
	token string,
) error {
	// Validate the signing certificate URL
	if !v.isValidSigningCertURL(signingCertURL) {
		return fmt.Errorf("invalid signing certificate URL: %s", signingCertURL)
	}

	// Download and cache the certificate
	cert, err := v.getCertificate(signingCertURL)
	if err != nil {
		return fmt.Errorf("failed to get certificate: %w", err)
	}

	// Build the string to sign based on message type
	stringToSign := v.buildStringToSign(messageType, messageID, message, subject, timestamp, topicArn, subscribeURL, token)

	// Decode the base64 signature
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify the signature
	if err := v.verifySignature(stringToSign, sig, cert, signatureVersion); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// isValidSigningCertURL validates that the certificate URL is from AWS SNS.
func (v *Verifier) isValidSigningCertURL(certURL string) bool {
	u, err := url.Parse(certURL)
	if err != nil {
		return false
	}

	// Must be HTTPS
	if u.Scheme != "https" {
		return false
	}

	// Must be from AWS (amazonaws.com or amazon.com)
	return strings.HasSuffix(u.Host, ".amazonaws.com") ||
		strings.HasSuffix(u.Host, ".amazon.com")
}

// getCertificate downloads and caches the X.509 certificate.
func (v *Verifier) getCertificate(certURL string) (*x509.Certificate, error) {
	// Check cache first
	if cached, ok := v.certCache.Load(certURL); ok {
		return cached.(*x509.Certificate), nil
	}

	// Download certificate
	resp, err := v.client.Get(certURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch certificate: status %d", resp.StatusCode)
	}

	certData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse PEM
	block, _ := pem.Decode(certData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	// Cache the certificate
	v.certCache.Store(certURL, cert)

	return cert, nil
}

// buildStringToSign constructs the canonical string to sign per AWS documentation.
func (v *Verifier) buildStringToSign(
	messageType string,
	messageID string,
	message string,
	subject string,
	timestamp string,
	topicArn string,
	subscribeURL string,
	token string,
) string {
	var sb strings.Builder

	switch messageType {
	case "Notification":
		// For Notification messages
		sb.WriteString("Message\n")
		sb.WriteString(message)
		sb.WriteString("\n")

		sb.WriteString("MessageId\n")
		sb.WriteString(messageID)
		sb.WriteString("\n")

		// Subject is optional - only include if present
		if subject != "" {
			sb.WriteString("Subject\n")
			sb.WriteString(subject)
			sb.WriteString("\n")
		}

		sb.WriteString("Timestamp\n")
		sb.WriteString(timestamp)
		sb.WriteString("\n")

		sb.WriteString("TopicArn\n")
		sb.WriteString(topicArn)
		sb.WriteString("\n")

		sb.WriteString("Type\n")
		sb.WriteString(messageType)
		sb.WriteString("\n")

	case "SubscriptionConfirmation", "UnsubscribeConfirmation":
		// For confirmation messages
		sb.WriteString("Message\n")
		sb.WriteString(message)
		sb.WriteString("\n")

		sb.WriteString("MessageId\n")
		sb.WriteString(messageID)
		sb.WriteString("\n")

		sb.WriteString("SubscribeURL\n")
		sb.WriteString(subscribeURL)
		sb.WriteString("\n")

		sb.WriteString("Timestamp\n")
		sb.WriteString(timestamp)
		sb.WriteString("\n")

		// Token is required for confirmation messages
		if token != "" {
			sb.WriteString("Token\n")
			sb.WriteString(token)
			sb.WriteString("\n")
		}

		sb.WriteString("TopicArn\n")
		sb.WriteString(topicArn)
		sb.WriteString("\n")

		sb.WriteString("Type\n")
		sb.WriteString(messageType)
		sb.WriteString("\n")
	}

	return sb.String()
}

// verifySignature verifies the signature using the public key from the certificate.
func (v *Verifier) verifySignature(stringToSign string, signature []byte, cert *x509.Certificate, signatureVersion string) error {
	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificate does not contain RSA public key")
	}

	var hash crypto.Hash
	var hashed []byte

	switch signatureVersion {
	case "1":
		// SHA1 with RSA (legacy)
		hash = crypto.SHA1
		h := sha1.Sum([]byte(stringToSign))
		hashed = h[:]
	case "2":
		// SHA256 with RSA (recommended)
		hash = crypto.SHA256
		h := sha256.Sum256([]byte(stringToSign))
		hashed = h[:]
	default:
		return fmt.Errorf("unsupported signature version: %s", signatureVersion)
	}

	return rsa.VerifyPKCS1v15(publicKey, hash, hashed, signature)
}
