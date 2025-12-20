-- name: CreateUser :one
INSERT INTO users (id, email, name, role, api_key_hash, api_key_prefix, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetUserByID :one
SELECT id, email, name, role, api_key_prefix, is_active, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, name, role, api_key_prefix, is_active, created_at, updated_at
FROM users WHERE email = $1;

-- name: GetUserByAPIKey :one
SELECT id, email, name, role, api_key_prefix, is_active, created_at, updated_at
FROM users WHERE api_key_hash = $1 AND is_active = true;

-- name: UpdateUser :one
UPDATE users SET
    email = $2,
    name = $3,
    role = $4,
    is_active = $5,
    updated_at = $6
WHERE id = $1
RETURNING *;

-- name: UpdateUserAPIKey :exec
UPDATE users SET api_key_hash = $2, api_key_prefix = $3, updated_at = NOW() WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
