-- ClickHouse schema for high-volume email event logs
-- This table is optimized for analytics and time-series queries

CREATE TABLE IF NOT EXISTS email_logs
(
    id UUID DEFAULT generateUUIDv4(),
    email_id String,
    user_id String,
    event_type Enum8(
        'sent' = 1,
        'delivered' = 2,
        'opened' = 3,
        'clicked' = 4,
        'bounced' = 5,
        'failed' = 6
    ),
    level Enum8('info' = 1, 'warning' = 2, 'error' = 3),
    message String,
    metadata String, -- JSON string
    ip_address String,
    user_agent String,
    timestamp DateTime64(3) DEFAULT now64(3),
    
    -- For efficient time-based queries
    date Date DEFAULT toDate(timestamp)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, date, timestamp)
TTL date + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

-- Materialized view for event counts by user
CREATE MATERIALIZED VIEW IF NOT EXISTS email_events_by_user_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, date, event_type)
AS SELECT
    user_id,
    toDate(timestamp) as date,
    event_type,
    count() as event_count
FROM email_logs
GROUP BY user_id, date, event_type;

-- Materialized view for daily email statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS daily_email_stats_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, event_type)
AS SELECT
    toDate(timestamp) as date,
    event_type,
    count() as event_count
FROM email_logs
GROUP BY date, event_type;
