-- +goose Up
-- Consolidated ClickHouse schema: activity_logs + email_routing

-- Activity logs for analytics and debugging (90 day retention)
CREATE TABLE IF NOT EXISTS activity_logs (
    id UUID DEFAULT generateUUIDv4(),
    user_id String,
    entity_type LowCardinality(String), -- 'email', 'domain', 'api_key', 'webhook'
    entity_id String,
    action LowCardinality(String),      -- 'sent', 'delivered', 'bounced', 'created', 'deleted', 'reply_received'
    status LowCardinality(String),      -- 'success', 'failed', 'pending'
    details String,                     -- Human readable message
    metadata String,                    -- JSON payload
    ip_address String,
    user_agent String,
    timestamp DateTime64(3) DEFAULT now64(3),
    date Date DEFAULT toDate(timestamp)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, entity_type, date, timestamp)
TTL date + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

-- Email routing table for reply mapping (no TTL - infinite retention)
-- Maps message_id -> user_id for webhook routing
CREATE TABLE IF NOT EXISTS email_routing (
    message_id String,
    email_id String,
    user_id String,
    sent_at DateTime DEFAULT now(),
    INDEX idx_message_id message_id TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(sent_at)
ORDER BY (message_id, sent_at)
SETTINGS index_granularity = 8192;

-- +goose Down
DROP TABLE IF EXISTS email_routing;
DROP TABLE IF EXISTS activity_logs;
