-- +goose Up
-- Add message_id and threading support to email_archive table

ALTER TABLE email_archive ADD COLUMN message_id String AFTER provider_id;
ALTER TABLE email_archive ADD COLUMN in_reply_to String AFTER message_id;

-- +goose Down
ALTER TABLE email_archive DROP COLUMN IF EXISTS in_reply_to;
ALTER TABLE email_archive DROP COLUMN IF EXISTS message_id;
