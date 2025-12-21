-- +goose Up
-- Email archive table for completed emails (moved from PostgreSQL after send/fail)
-- This is the permanent record of emails with 30-day retention

CREATE TABLE IF NOT EXISTS email_archive
(
    id String,
    user_id String,
    from_address String,
    to_addresses Array(String),
    cc_addresses Array(String),
    bcc_addresses Array(String),
    subject String,
    body String,
    html String,
    status String,
    provider_id String,
    attachment_count UInt8 DEFAULT 0,
    metadata String, -- JSON string
    error_message String,
    scheduled_at Nullable(DateTime64(3)),
    sent_at Nullable(DateTime64(3)),
    created_at DateTime64(3),
    archived_at DateTime64(3) DEFAULT now64(3),
    date Date DEFAULT toDate(created_at)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, date, id)
TTL date + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;

-- Update existing emails table TTL to 30 days
-- Note: This requires ALTER TABLE which may take time on large tables
ALTER TABLE emails MODIFY TTL date + INTERVAL 30 DAY DELETE;

-- Materialized view for archive stats by user
CREATE MATERIALIZED VIEW IF NOT EXISTS email_archive_stats_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, date, status)
AS SELECT
    user_id,
    toDate(created_at) as date,
    status,
    count() as email_count
FROM email_archive
GROUP BY user_id, date, status;

-- +goose Down
DROP VIEW IF EXISTS email_archive_stats_mv;
DROP TABLE IF EXISTS email_archive;
-- Revert TTL to 90 days
ALTER TABLE emails MODIFY TTL date + INTERVAL 90 DAY DELETE;
