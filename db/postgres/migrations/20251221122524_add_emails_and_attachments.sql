-- +goose Up

-- Emails table for mutable email state tracking
-- Emails are stored here until sent/failed, then archived to ClickHouse and deleted
CREATE TABLE emails (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_address TEXT NOT NULL,
    to_addresses TEXT[] NOT NULL,
    cc_addresses TEXT[],
    bcc_addresses TEXT[],
    subject TEXT NOT NULL,
    body TEXT,
    html TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    provider_id TEXT,
    metadata JSONB DEFAULT '{}',
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_emails_user_id ON emails(user_id);
CREATE INDEX idx_emails_status ON emails(status);
CREATE INDEX idx_emails_created_at ON emails(created_at);
CREATE INDEX idx_emails_scheduled_at ON emails(scheduled_at) WHERE scheduled_at IS NOT NULL;

-- Email attachments table
CREATE TABLE email_attachments (
    id TEXT PRIMARY KEY,
    email_id TEXT NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    s3_key TEXT NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    scan_status TEXT NOT NULL DEFAULT 'pending',
    scan_checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_attachments_email_id ON email_attachments(email_id);
CREATE INDEX idx_email_attachments_scan_status ON email_attachments(scan_status);
CREATE INDEX idx_email_attachments_s3_key ON email_attachments(s3_key);

-- +goose Down
DROP TABLE IF EXISTS email_attachments;
DROP TABLE IF EXISTS emails;
