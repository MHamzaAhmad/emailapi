-- name: InsertUnsubscribe :exec
INSERT INTO unsubscribe_list (id, user_id, email_hash, source_email_id, source)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, email_hash) DO NOTHING;

-- name: GetUnsubscribeByUserAndHash :one
SELECT * FROM unsubscribe_list 
WHERE user_id = $1 AND email_hash = $2;

-- name: CheckUnsubscribeBatch :many
-- Check multiple email hashes for a user in a single query
SELECT email_hash FROM unsubscribe_list 
WHERE user_id = $1 AND email_hash = ANY($2::text[]);

-- name: DeleteUnsubscribe :exec
DELETE FROM unsubscribe_list 
WHERE user_id = $1 AND email_hash = $2;

-- name: ListUnsubscribesByUser :many
SELECT id, email_hash, source_email_id, source, created_at
FROM unsubscribe_list 
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUnsubscribesByUser :one
SELECT COUNT(*) FROM unsubscribe_list WHERE user_id = $1;

-- name: ListAllUnsubscribes :many
-- For cache sync on startup
SELECT user_id, email_hash FROM unsubscribe_list;
