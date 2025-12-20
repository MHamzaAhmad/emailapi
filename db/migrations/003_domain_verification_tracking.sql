-- Add last_verified_at to track when SES was last queried for verification status
ALTER TABLE domains ADD COLUMN last_verified_at TIMESTAMPTZ;

-- Index for efficiently finding domains that need refresh
CREATE INDEX idx_domains_last_verified_at ON domains(last_verified_at);
