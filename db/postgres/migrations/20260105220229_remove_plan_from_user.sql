-- +goose Up
-- Remove plan column from users table (now managed entirely by Polar)

DROP INDEX IF EXISTS idx_users_plan;
ALTER TABLE users DROP COLUMN IF EXISTS plan;
DROP TYPE IF EXISTS user_plan;

-- +goose Down
-- Re-create plan enum and column
CREATE TYPE user_plan AS ENUM ('free', 'starter', 'growth');
ALTER TABLE users ADD COLUMN plan user_plan NOT NULL DEFAULT 'free';
CREATE INDEX idx_users_plan ON users(plan);
