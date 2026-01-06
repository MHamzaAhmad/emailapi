package ses

// IdentityResult contains the result of an email identity operation.
type IdentityResult struct {
	// Whether the identity is verified and can send emails.
	VerifiedForSendingStatus bool

	// DKIM verification status: PENDING, SUCCESS, FAILED, TEMPORARY_FAILURE, NOT_STARTED.
	DkimStatus string

	// DKIM tokens for CNAME records (3 tokens).
	DkimTokens []string

	// Custom MAIL FROM domain if configured.
	MailFromDomain string

	// MAIL FROM status: PENDING, SUCCESS, FAILED, TEMPORARY_FAILURE.
	MailFromStatus string
}
