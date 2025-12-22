package sns

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Verifier validates AWS SNS message signatures.
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
	messageType string, // "Notification", "SubscriptionConfirmation", etc.
	message string, // The complete JSON message
	messageID string,
	timestamp string,
	topicArn string,
	subscribeURL string, // Only for SubscriptionConfirmation
	subject string, // Only for Notification with subject
) error {
	// Validate certificate URL is from AWS
	if err := v.validateCertURL(signingCertURL); err != nil {
		return fmt.Errorf("invalid signing cert URL: %w", err)
	}

	// Get or fetch certificate
	cert, err := v.getCertificate(signingCertURL)
	if err != nil {
		return fmt.Errorf("failed to get certificate: %w", err)
	}

	// Build the string to sign based on message type
	stringToSign := v.buildStringToSign(messageType, messageID, timestamp, topicArn, message, subscribeURL, subject)

	// Decode signature
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify based on signature version
	switch signatureVersion {
	case "1":
		// SHA1 with RSA
		err = cert.CheckSignature(x509.SHA1WithRSA, []byte(stringToSign), sig)
	case "2":
		// SHA256 with RSA
		err = cert.CheckSignature(x509.SHA256WithRSA, []byte(stringToSign), sig)
	default:
		return fmt.Errorf("unsupported signature version: %s", signatureVersion)
	}

	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// validateCertURL ensures the certificate URL is from AWS SNS.
func (v *Verifier) validateCertURL(certURL string) error {
	parsed, err := url.Parse(certURL)
	if err != nil {
		return err
	}

	// Must be HTTPS
	if parsed.Scheme != "https" {
		return errors.New("certificate URL must use HTTPS")
	}

	// Must be from amazonaws.com
	if !strings.HasSuffix(parsed.Host, ".amazonaws.com") {
		return errors.New("certificate URL must be from amazonaws.com")
	}

	return nil
}

// getCertificate fetches and caches the signing certificate.
func (v *Verifier) getCertificate(certURL string) (*x509.Certificate, error) {
	// Check cache first
	if cached, ok := v.certCache.Load(certURL); ok {
		return cached.(*x509.Certificate), nil
	}

	// Fetch certificate
	resp, err := v.client.Get(certURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch certificate: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse PEM
	block, _ := pem.Decode(body)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	// Cache the certificate
	v.certCache.Store(certURL, cert)

	return cert, nil
}

// buildStringToSign creates the canonical string for signature verification.
func (v *Verifier) buildStringToSign(
	messageType string,
	messageID string,
	timestamp string,
	topicArn string,
	message string,
	subscribeURL string,
	subject string,
) string {
	var sb strings.Builder

	sb.WriteString("Message\n")
	sb.WriteString(message)
	sb.WriteString("\n")

	sb.WriteString("MessageId\n")
	sb.WriteString(messageID)
	sb.WriteString("\n")

	if subject != "" && messageType == "Notification" {
		sb.WriteString("Subject\n")
		sb.WriteString(subject)
		sb.WriteString("\n")
	}

	if messageType == "SubscriptionConfirmation" || messageType == "UnsubscribeConfirmation" {
		sb.WriteString("SubscribeURL\n")
		sb.WriteString(subscribeURL)
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

	return sb.String()
}
