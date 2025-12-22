-- name: InsertSuppression :exec
INSERT INTO suppression_list (email_hash, user_id, reason, bounce_type, source_message_id, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (email_hash) DO UPDATE SET
    reason = EXCLUDED.reason,
    bounce_type = EXCLUDED.bounce_type,
    source_message_id = EXCLUDED.source_message_id,
    expires_at = EXCLUDED.expires_at,
    created_at = NOW();

-- name: GetSuppressionByHash :one
SELECT * FROM suppression_list WHERE email_hash = $1;

-- name: DeleteExpiredSuppressions :execrows
DELETE FROM suppression_list 
WHERE expires_at IS NOT NULL AND expires_at < NOW();

-- name: DeleteSuppressionByHash :exec
DELETE FROM suppression_list WHERE email_hash = $1;

-- name: ListActiveSuppressions :many
SELECT email_hash, user_id, reason, bounce_type, expires_at, created_at
FROM suppression_list 
WHERE expires_at IS NULL OR expires_at > NOW();

-- name: ListSuppressionsByUser :many
SELECT email_hash, reason, bounce_type, expires_at, created_at
FROM suppression_list 
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSuppressionsByUser :one
SELECT COUNT(*) FROM suppression_list WHERE user_id = $1;
