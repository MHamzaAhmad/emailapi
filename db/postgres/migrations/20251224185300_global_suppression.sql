-- +goose Up
-- Make user_id nullable to support global suppressions (hard bounces, complaints)
-- Global suppressions (user_id IS NULL) block ALL users from sending to that email

-- Drop the foreign key constraint
ALTER TABLE suppression_list DROP CONSTRAINT suppression_list_user_id_fkey;

-- Make user_id nullable
ALTER TABLE suppression_list ALTER COLUMN user_id DROP NOT NULL;

-- Re-add foreign key with ON DELETE SET NULL for user deletions
-- NULL user_id = global suppression, still valid
ALTER TABLE suppression_list 
ADD CONSTRAINT suppression_list_user_id_fkey 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

-- +goose Down
-- Revert: make user_id NOT NULL again

-- First, delete any global suppressions (user_id IS NULL)
DELETE FROM suppression_list WHERE user_id IS NULL;

-- Drop the new constraint
ALTER TABLE suppression_list DROP CONSTRAINT suppression_list_user_id_fkey;

-- Make user_id NOT NULL again
ALTER TABLE suppression_list ALTER COLUMN user_id SET NOT NULL;

-- Re-add original foreign key
ALTER TABLE suppression_list 
ADD CONSTRAINT suppression_list_user_id_fkey 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
