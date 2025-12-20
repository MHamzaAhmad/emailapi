-- name: CreateEmail :one
INSERT INTO emails (
    id, from_address, to_addresses, cc_addresses, bcc_addresses,
    subject, body, html_body, status, provider_id, user_id, webhook_id,
    metadata, scheduled_at, sent_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
) RETURNING *;

-- name: GetEmailByID :one
SELECT * FROM emails WHERE id = $1;

-- name: GetEmailsByUserID :many
SELECT * FROM emails 
WHERE user_id = $1 
ORDER BY created_at DESC 
LIMIT $2 OFFSET $3;

-- name: UpdateEmail :one
UPDATE emails SET
    from_address = $2,
    to_addresses = $3,
    cc_addresses = $4,
    bcc_addresses = $5,
    subject = $6,
    body = $7,
    html_body = $8,
    status = $9,
    provider_id = $10,
    webhook_id = $11,
    metadata = $12,
    scheduled_at = $13,
    sent_at = $14,
    updated_at = $15
WHERE id = $1
RETURNING *;

-- name: UpdateEmailStatus :exec
UPDATE emails SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: CountEmailsByUserID :one
SELECT COUNT(*) FROM emails WHERE user_id = $1;
