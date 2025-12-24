-- name: CreateApiKey :one
INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, scopes, is_active, expires_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetApiKeyByID :one
SELECT id, user_id, name, key_hash, key_prefix, scopes, is_active, last_used_at, expires_at, created_at, updated_at
FROM api_keys WHERE id = $1;

-- name: GetApiKeyByHash :one
SELECT id, user_id, name, key_hash, key_prefix, scopes, is_active, last_used_at, expires_at, created_at, updated_at
FROM api_keys WHERE key_hash = $1 AND is_active = true;

-- name: GetApiKeyByPrefix :one
SELECT id, user_id, name, key_hash, key_prefix, scopes, is_active, last_used_at, expires_at, created_at, updated_at
FROM api_keys WHERE key_prefix = $1;

-- name: ListApiKeysByUserID :many
SELECT id, user_id, name, key_prefix, scopes, is_active, last_used_at, expires_at, created_at, updated_at
FROM api_keys WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateApiKey :one
UPDATE api_keys SET
    name = $2,
    scopes = $3,
    is_active = $4,
    expires_at = $5,
    updated_at = $6
WHERE id = $1
RETURNING *;

-- name: UpdateApiKeyLastUsed :exec
UPDATE api_keys SET last_used_at = NOW() WHERE id = $1;

-- name: RevokeApiKey :one
UPDATE api_keys SET is_active = false, updated_at = NOW() WHERE id = $1
RETURNING *;

-- name: DeleteApiKey :exec
DELETE FROM api_keys WHERE id = $1;

-- name: CountActiveApiKeysByUserID :one
SELECT COUNT(*) FROM api_keys WHERE user_id = $1 AND is_active = true;

-- name: ListApiKeysByUserIDPaginated :many
SELECT id, user_id, name, key_prefix, scopes, is_active, last_used_at, expires_at, created_at, updated_at
FROM api_keys
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountApiKeysByUserID :one
SELECT COUNT(*) FROM api_keys WHERE user_id = $1;
