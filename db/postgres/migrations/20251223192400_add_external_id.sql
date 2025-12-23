-- +goose Up
-- Add external_id column for Clerk user identity
ALTER TABLE users ADD COLUMN external_id TEXT UNIQUE;
CREATE INDEX idx_users_external_id ON users(external_id);

-- +goose Down
DROP INDEX idx_users_external_id;
ALTER TABLE users DROP COLUMN external_id;
