package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// PendingAttachmentTTL is the time-to-live for pending attachment entries.
// Set to 10 minutes to allow for scan completion with some buffer.
const PendingAttachmentTTL = 10 * time.Minute

// PendingAttachmentData contains all data needed to resume email sending
// after attachment scanning is complete.
type PendingAttachmentData struct {
	EmailID                string            `json:"email_id"`
	UserID                 string            `json:"user_id"`
	From                   string            `json:"from"`
	To                     []string          `json:"to"`
	Cc                     []string          `json:"cc,omitempty"`
	Bcc                    []string          `json:"bcc,omitempty"`
	Subject                string            `json:"subject"`
	Body                   string            `json:"body,omitempty"`
	HTML                   string            `json:"html,omitempty"`
	InReplyTo              string            `json:"in_reply_to,omitempty"`
	References             []string          `json:"references,omitempty"`
	Metadata               map[string]string `json:"metadata,omitempty"`
	ScheduledAt            *time.Time        `json:"scheduled_at,omitempty"`
	UnsubscribeBaseURL     string            `json:"unsubscribe_base_url,omitempty"`
	UnsubscribeTokenSecret string            `json:"unsubscribe_token_secret,omitempty"`
	// AttachmentKeys contains all S3 keys for attachments belonging to this email.
	// Multiple attachments map to the same email data.
	AttachmentKeys []AttachmentKeyInfo `json:"attachment_keys"`
	// PendingCount is decremented as each attachment scan completes.
	// When it reaches 0, all attachments are scanned.
	PendingCount int `json:"pending_count"`
}

// AttachmentKeyInfo contains S3 key and metadata for an attachment.
type AttachmentKeyInfo struct {
	S3Key       string `json:"s3_key"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

// PendingAttachmentCache manages pending attachment email data in Redis.
// Keys are stored as: pending_attachment:{email_id}
// A reverse lookup exists: attachment_email:{s3_key} -> email_id
type PendingAttachmentCache struct {
	client *Client
}

// NewPendingAttachmentCache creates a new PendingAttachmentCache.
func NewPendingAttachmentCache(client *Client) *PendingAttachmentCache {
	return &PendingAttachmentCache{client: client}
}

func (c *PendingAttachmentCache) emailKey(emailID string) string {
	return fmt.Sprintf("pending_attachment:%s", emailID)
}

func (c *PendingAttachmentCache) s3KeyLookup(s3Key string) string {
	return fmt.Sprintf("attachment_email:%s", s3Key)
}

// Store saves pending attachment data and creates reverse lookups for each S3 key.
func (c *PendingAttachmentCache) Store(ctx context.Context, data *PendingAttachmentData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal pending attachment data: %w", err)
	}

	// Store main entry
	emailKey := c.emailKey(data.EmailID)
	if err := c.client.Set(ctx, emailKey, string(jsonData), PendingAttachmentTTL); err != nil {
		return fmt.Errorf("failed to store pending attachment: %w", err)
	}

	// Create reverse lookups for each S3 key
	for _, att := range data.AttachmentKeys {
		lookupKey := c.s3KeyLookup(att.S3Key)
		if err := c.client.Set(ctx, lookupKey, data.EmailID, PendingAttachmentTTL); err != nil {
			// Log but don't fail - we can fall back to other lookup methods
			fmt.Printf("Warning: failed to store S3 key lookup %s: %v\n", att.S3Key, err)
		}
	}

	return nil
}

// GetByEmailID retrieves pending attachment data by email ID.
func (c *PendingAttachmentCache) GetByEmailID(ctx context.Context, emailID string) (*PendingAttachmentData, error) {
	key := c.emailKey(emailID)
	jsonData, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if jsonData == "" {
		return nil, nil // Not found
	}

	var data PendingAttachmentData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pending attachment data: %w", err)
	}
	return &data, nil
}

// GetByS3Key retrieves pending attachment data by S3 key (uses reverse lookup).
func (c *PendingAttachmentCache) GetByS3Key(ctx context.Context, s3Key string) (*PendingAttachmentData, error) {
	lookupKey := c.s3KeyLookup(s3Key)
	emailID, err := c.client.Get(ctx, lookupKey)
	if err != nil {
		return nil, err
	}
	if emailID == "" {
		return nil, nil // Not found
	}

	return c.GetByEmailID(ctx, emailID)
}

// Delete removes pending attachment data and all reverse lookups.
func (c *PendingAttachmentCache) Delete(ctx context.Context, emailID string) error {
	// First get the data to find all S3 keys
	data, err := c.GetByEmailID(ctx, emailID)
	if err != nil {
		return err
	}
	if data == nil {
		return nil // Already deleted
	}

	// Delete reverse lookups
	for _, att := range data.AttachmentKeys {
		lookupKey := c.s3KeyLookup(att.S3Key)
		_ = c.client.Del(ctx, lookupKey) // Best effort
	}

	// Delete main entry
	return c.client.Del(ctx, c.emailKey(emailID))
}

// MarkAttachmentScanned decrements the pending count for an email.
// Returns the updated data and whether all attachments are now scanned.
func (c *PendingAttachmentCache) MarkAttachmentScanned(ctx context.Context, s3Key string) (*PendingAttachmentData, bool, error) {
	data, err := c.GetByS3Key(ctx, s3Key)
	if err != nil {
		return nil, false, err
	}
	if data == nil {
		return nil, false, nil // Not found - might be expired or already processed
	}

	// Decrement pending count
	data.PendingCount--
	allScanned := data.PendingCount <= 0

	// Update the entry
	if err := c.Store(ctx, data); err != nil {
		return nil, false, err
	}

	// Delete the S3 key lookup (this attachment is done)
	lookupKey := c.s3KeyLookup(s3Key)
	_ = c.client.Del(ctx, lookupKey)

	return data, allScanned, nil
}
