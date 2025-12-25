-- Up Migration

-- Optimize index for key_hash lookups
DROP INDEX IF EXISTS idx_api_keys_key_hash;
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);

-- Down Migration
DROP INDEX IF EXISTS idx_api_keys_key_hash;
