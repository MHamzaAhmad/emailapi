-- +goose Up
-- Suppression list for bounced/complained emails
-- Stores SHA-256 hashes, never raw emails

CREATE TABLE suppression_list (
    email_hash TEXT PRIMARY KEY,          -- SHA-256 hash of lowercased email
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,                 -- 'bounce_hard', 'bounce_soft', 'complaint'
    bounce_type TEXT,                     -- 'Permanent', 'Transient', etc.
    source_message_id TEXT,               -- SES Message-ID that triggered this
    expires_at TIMESTAMPTZ,               -- NULL = permanent (hard bounce/complaint)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for cleanup job to find expired entries
CREATE INDEX idx_suppression_expires ON suppression_list(expires_at) 
WHERE expires_at IS NOT NULL;

-- Index for listing by user
CREATE INDEX idx_suppression_user_id ON suppression_list(user_id);

-- +goose Down
DROP TABLE suppression_list;
