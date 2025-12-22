-- +goose Up
-- Add reply_received event type to the emails table Enum

-- Note: ClickHouse doesn't support ALTER MODIFY COLUMN for Enum expansion easily.
-- We need to recreate using ALTER MODIFY to add the new enum value.
ALTER TABLE emails MODIFY COLUMN event_type Enum8(
    'sent' = 1,
    'delivered' = 2,
    'opened' = 3,
    'clicked' = 4,
    'bounced' = 5,
    'failed' = 6,
    'reply_received' = 7
);

-- +goose Down
ALTER TABLE emails MODIFY COLUMN event_type Enum8(
    'sent' = 1,
    'delivered' = 2,
    'opened' = 3,
    'clicked' = 4,
    'bounced' = 5,
    'failed' = 6
);
