-- +goose Up
-- Email Routing Table: Infinitely retained mapping for reply routing.
-- Stores minimal data (message_id -> user_id + from_email) to route inbound replies.

CREATE TABLE IF NOT EXISTS email_routing (
    -- The Message-ID header we sent (unique per email)
    message_id String,
    -- Our internal email ID
    email_id String,
    -- User who sent the email
    user_id String,
    -- Original "From" address (for translating reply-to back to sender)
    from_email String,
    -- When the email was sent
    sent_at DateTime DEFAULT now(),
    
    -- Partition by year-month for efficient storage, but no TTL (infinite retention)
    INDEX idx_message_id message_id TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(sent_at)
ORDER BY (message_id, sent_at)
SETTINGS index_granularity = 8192;

-- +goose Down
DROP TABLE IF EXISTS email_routing;
