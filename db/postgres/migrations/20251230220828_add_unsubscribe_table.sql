-- +goose Up
-- Unsubscribe list for tracking recipient email preferences per user
-- Uses SHA-256 hashes for privacy (never stores raw emails)

CREATE TABLE unsubscribe_list (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_hash TEXT NOT NULL,              -- SHA-256 hash of lowercased recipient email
    source_email_id TEXT,                  -- Email ID that triggered unsubscribe (optional)
    source TEXT NOT NULL DEFAULT 'link',   -- 'link', 'one_click', 'manual'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(user_id, email_hash)            -- One entry per user+recipient pair
);

-- Index for fast lookups by user (for batch checks during sending)
CREATE INDEX idx_unsubscribe_user_id ON unsubscribe_list(user_id);

-- Index for hash lookups (composite with user_id for efficient filtering)
CREATE INDEX idx_unsubscribe_lookup ON unsubscribe_list(user_id, email_hash);

-- +goose Down
DROP TABLE unsubscribe_list;
