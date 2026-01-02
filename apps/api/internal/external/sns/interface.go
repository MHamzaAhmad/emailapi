package sns

//go:generate mockgen -destination=mocks/mock_sns.go -package=mocks github.com/emailapi/api/internal/external/sns VerifierInterface

// VerifierInterface defines the interface for SNS signature verification.
type VerifierInterface interface {
	// VerifySignature validates an SNS message signature.
	// Returns nil if valid, error if invalid or verification fails.
	VerifySignature(
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
	) error
}

// Ensure concrete type implements interface
var _ VerifierInterface = (*Verifier)(nil)
