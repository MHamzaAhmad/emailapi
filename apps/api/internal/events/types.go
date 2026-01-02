package events

// EventType represents the type of SES/inbound email event.
type EventType string

const (
	// SES sending events (from Configuration Set)
	EventTypeBounce    EventType = "Bounce"
	EventTypeComplaint EventType = "Complaint"
	EventTypeDelivery  EventType = "Delivery"
	EventTypeSend      EventType = "Send"
	EventTypeReject    EventType = "Reject"
	EventTypeOpen      EventType = "Open"
	EventTypeClick     EventType = "Click"

	// Inbound email events (from Receipt Rules)
	EventTypeReceived EventType = "Received"
)

// EventEnvelope is the outer wrapper for all SES events from EventBridge.
// This matches the EventBridge event format.
type EventEnvelope struct {
	Version    string      `json:"version"`
	ID         string      `json:"id"`
	DetailType string      `json:"detail-type"`
	Source     string      `json:"source"`
	Account    string      `json:"account"`
	Time       string      `json:"time"`
	Region     string      `json:"region"`
	Detail     interface{} `json:"detail"`
}

// SNSNotification wraps SES events when delivered via SNS->SQS.
type SNSNotification struct {
	Type             string `json:"Type"`
	MessageId        string `json:"MessageId"`
	TopicArn         string `json:"TopicArn"`
	Message          string `json:"Message"` // Contains the actual SES event JSON
	Timestamp        string `json:"Timestamp"`
	SignatureVersion string `json:"SignatureVersion"`
}

// SESEvent is the common structure for SES sending events.
type SESEvent struct {
	EventType string     `json:"eventType"`
	Mail      MailInfo   `json:"mail"`
	Bounce    *Bounce    `json:"bounce,omitempty"`
	Complaint *Complaint `json:"complaint,omitempty"`
	Delivery  *Delivery  `json:"delivery,omitempty"`
}

// MailInfo contains information about the sent email.
type MailInfo struct {
	Timestamp        string            `json:"timestamp"`
	MessageId        string            `json:"messageId"`
	Source           string            `json:"source"`
	SourceArn        string            `json:"sourceArn"`
	SendingAccountId string            `json:"sendingAccountId"`
	Destination      []string          `json:"destination"`
	HeadersTruncated bool              `json:"headersTruncated"`
	Headers          []Header          `json:"headers"`
	CommonHeaders    CommonHeaders     `json:"commonHeaders"`
	Tags             map[string]string `json:"tags"`
}

// Header represents an email header.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CommonHeaders contains common email headers.
type CommonHeaders struct {
	From      []string `json:"from"`
	To        []string `json:"to"`
	MessageId string   `json:"messageId"`
	Subject   string   `json:"subject"`
}

// Bounce contains bounce-specific information.
type Bounce struct {
	BounceType        string             `json:"bounceType"`
	BounceSubType     string             `json:"bounceSubType"`
	BouncedRecipients []BouncedRecipient `json:"bouncedRecipients"`
	Timestamp         string             `json:"timestamp"`
	FeedbackId        string             `json:"feedbackId"`
	ReportingMTA      string             `json:"reportingMTA"`
}

// BouncedRecipient contains information about a bounced recipient.
type BouncedRecipient struct {
	EmailAddress   string `json:"emailAddress"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	DiagnosticCode string `json:"diagnosticCode"`
}

// Complaint contains complaint-specific information.
type Complaint struct {
	ComplainedRecipients  []ComplainedRecipient `json:"complainedRecipients"`
	Timestamp             string                `json:"timestamp"`
	FeedbackId            string                `json:"feedbackId"`
	ComplaintSubType      string                `json:"complaintSubType"`
	ComplaintFeedbackType string                `json:"complaintFeedbackType"`
}

// ComplainedRecipient contains information about a complained recipient.
type ComplainedRecipient struct {
	EmailAddress string `json:"emailAddress"`
}

// Delivery contains delivery-specific information.
type Delivery struct {
	Timestamp            string   `json:"timestamp"`
	ProcessingTimeMillis int64    `json:"processingTimeMillis"`
	Recipients           []string `json:"recipients"`
	SmtpResponse         string   `json:"smtpResponse"`
	ReportingMTA         string   `json:"reportingMTA"`
}

// InboundEmailNotification is the SES inbound email notification structure.
type InboundEmailNotification struct {
	NotificationType string          `json:"notificationType"` // "Received"
	Mail             InboundMailInfo `json:"mail"`
	Receipt          InboundReceipt  `json:"receipt"`
	Content          string          `json:"content,omitempty"` // Raw email if included
}

// InboundMailInfo contains information about the received email.
type InboundMailInfo struct {
	Timestamp        string        `json:"timestamp"`
	MessageId        string        `json:"messageId"`
	Source           string        `json:"source"`
	Destination      []string      `json:"destination"`
	HeadersTruncated bool          `json:"headersTruncated"`
	CommonHeaders    CommonHeaders `json:"commonHeaders"`
}

// InboundReceipt contains receipt processing information.
type InboundReceipt struct {
	Timestamp            string        `json:"timestamp"`
	ProcessingTimeMillis int64         `json:"processingTimeMillis"`
	Recipients           []string      `json:"recipients"`
	SpamVerdict          VerdictInfo   `json:"spamVerdict"`
	VirusVerdict         VerdictInfo   `json:"virusVerdict"`
	SPFVerdict           VerdictInfo   `json:"spfVerdict"`
	DKIMVerdict          VerdictInfo   `json:"dkimVerdict"`
	DMARCVerdict         VerdictInfo   `json:"dmarcVerdict"`
	Action               ReceiptAction `json:"action"`
}

// VerdictInfo contains a scan verdict.
type VerdictInfo struct {
	Status string `json:"status"` // "PASS", "FAIL", "GRAY", "PROCESSING_FAILED"
}

// ReceiptAction describes what action was taken on the email.
type ReceiptAction struct {
	Type       string `json:"type"` // "S3", "SNS", "Lambda", etc.
	TopicArn   string `json:"topicArn,omitempty"`
	BucketName string `json:"bucketName,omitempty"`
	ObjectKey  string `json:"objectKey,omitempty"`
}

// S3EventDetail is the EventBridge detail for S3 object events.
// This is sent when EventBridge notifications are enabled on an S3 bucket.
type S3EventDetail struct {
	Version         string       `json:"version"`
	Bucket          S3BucketInfo `json:"bucket"`
	Object          S3ObjectInfo `json:"object"`
	RequestID       string       `json:"request-id"`
	Requester       string       `json:"requester"`
	SourceIPAddress string       `json:"source-ip-address"`
	Reason          string       `json:"reason,omitempty"`
}

// S3BucketInfo contains S3 bucket information.
type S3BucketInfo struct {
	Name string `json:"name"`
}

// S3ObjectInfo contains S3 object information.
type S3ObjectInfo struct {
	Key       string `json:"key"`
	Size      int64  `json:"size"`
	ETag      string `json:"etag"`
	Sequencer string `json:"sequencer"`
}
