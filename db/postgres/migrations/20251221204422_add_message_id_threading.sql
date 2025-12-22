-- +goose Up
-- Add message_id and threading support to emails table

ALTER TABLE emails ADD COLUMN message_id TEXT;
ALTER TABLE emails ADD COLUMN in_reply_to TEXT;

-- Index for looking up emails by SES message_id (for threading/inbound matching)
CREATE INDEX idx_emails_message_id ON emails(message_id) WHERE message_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_emails_message_id;
ALTER TABLE emails DROP COLUMN IF EXISTS in_reply_to;
ALTER TABLE emails DROP COLUMN IF EXISTS message_id;
