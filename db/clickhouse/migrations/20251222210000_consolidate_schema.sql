-- +goose Up
-- Consolidate ClickHouse schema: Generic activity_logs + emails archive

-- 1. Create new generic activity_logs table
CREATE TABLE IF NOT EXISTS activity_logs (
    id UUID DEFAULT generateUUIDv4(),
    user_id String,
    entity_type LowCardinality(String), -- 'email', 'domain', 'api_key', 'webhook'
    entity_id String,
    action LowCardinality(String),      -- 'sent', 'delivered', 'bounced', 'created', 'deleted'
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

-- 2. Rename email_archive → emails (primary archive table)
RENAME TABLE email_archive TO emails;

-- 3. Update emails TTL to 90 days
ALTER TABLE emails MODIFY TTL date + INTERVAL 90 DAY DELETE;

-- 4. Drop old materialized views that depended on old emails table
DROP VIEW IF EXISTS email_events_by_user_mv;
DROP VIEW IF EXISTS daily_email_stats_mv;
DROP VIEW IF EXISTS email_archive_stats_mv;

-- +goose Down
-- Rollback: Restore original schema

-- Recreate materialized views (simplified - would need full definitions in production)
-- CREATE MATERIALIZED VIEW IF NOT EXISTS email_archive_stats_mv ...

-- Rename emails back to email_archive
RENAME TABLE emails TO email_archive;

-- Restore TTL
ALTER TABLE email_archive MODIFY TTL date + INTERVAL 30 DAY DELETE;

-- Drop activity_logs
DROP TABLE IF EXISTS activity_logs;
