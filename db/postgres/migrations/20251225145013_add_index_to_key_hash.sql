-- +goose Up
-- Optimize index for key_hash lookups
DROP INDEX IF EXISTS idx_api_keys_key_hash;
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);

-- +goose Down
DROP INDEX IF EXISTS idx_api_keys_key_hash;
