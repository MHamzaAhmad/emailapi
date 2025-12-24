-- +goose Up
DROP INDEX IF EXISTS idx_api_keys_environment;
ALTER TABLE api_keys DROP COLUMN IF EXISTS environment;

-- +goose Down
ALTER TABLE api_keys ADD COLUMN environment TEXT NOT NULL DEFAULT 'live';
CREATE INDEX idx_api_keys_environment ON api_keys(environment);
