-- +goose Up
-- +goose StatementBegin
-- Create the user_role enum type
CREATE TYPE user_role AS ENUM ('member', 'admin');

-- Alter the users table to use the new enum type
ALTER TABLE users 
    ALTER COLUMN role DROP DEFAULT,
    ALTER COLUMN role TYPE user_role USING role::user_role,
    ALTER COLUMN role SET DEFAULT 'member'::user_role;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Convert back to TEXT
ALTER TABLE users 
    ALTER COLUMN role DROP DEFAULT,
    ALTER COLUMN role TYPE TEXT USING role::TEXT,
    ALTER COLUMN role SET DEFAULT 'member';

-- Drop the enum type
DROP TYPE user_role;
-- +goose StatementEnd
