-- name: CreateUser :one
INSERT INTO users (id, email, name, role, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetUserByID :one
SELECT id, email, name, role, is_active, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, name, role, is_active, created_at, updated_at
FROM users WHERE email = $1;

-- name: UpdateUser :one
UPDATE users SET
    email = $2,
    name = $3,
    role = $4,
    is_active = $5,
    updated_at = $6
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, name, role, is_active, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
