-- +goose Up
-- Add user plan enum and polar customer tracking

-- Create plan enum type
CREATE TYPE user_plan AS ENUM ('free', 'scale', 'payg');

-- Add plan column with default 'free'
ALTER TABLE users ADD COLUMN plan user_plan NOT NULL DEFAULT 'free';

-- Add Polar customer ID for linking (nullable, free users don't have this)
ALTER TABLE users ADD COLUMN polar_customer_id TEXT;

-- Indexes for lookups
CREATE INDEX idx_users_plan ON users(plan);
CREATE INDEX idx_users_polar_customer_id ON users(polar_customer_id) 
    WHERE polar_customer_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_users_polar_customer_id;
DROP INDEX IF EXISTS idx_users_plan;
ALTER TABLE users DROP COLUMN polar_customer_id;
ALTER TABLE users DROP COLUMN plan;
DROP TYPE user_plan;
