-- name: CreateWebhook :one
INSERT INTO webhooks (id, user_id, name, url, secret_hash, events, is_active, retry_count, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetWebhookByID :one
SELECT id, user_id, name, url, events, is_active, retry_count, last_success, last_failure, created_at, updated_at
FROM webhooks WHERE id = $1;

-- name: GetWebhooksByUserID :many
SELECT id, user_id, name, url, events, is_active, retry_count, last_success, last_failure, created_at, updated_at
FROM webhooks WHERE user_id = $1 ORDER BY created_at DESC;

-- name: GetActiveWebhooksByEvent :many
SELECT id, user_id, name, url, events, is_active, retry_count, last_success, last_failure, created_at, updated_at
FROM webhooks WHERE is_active = true AND $1 = ANY(events);

-- name: UpdateWebhook :one
UPDATE webhooks SET
    name = $2,
    url = $3,
    events = $4,
    is_active = $5,
    retry_count = $6,
    last_success = $7,
    last_failure = $8,
    updated_at = $9
WHERE id = $1
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM webhooks WHERE id = $1;

-- name: CreateWebhookDelivery :one
INSERT INTO webhook_deliveries (id, webhook_id, event_type, payload, response_code, response_body, success, attempt_count, next_retry, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetWebhookDeliveriesByWebhookID :many
SELECT * FROM webhook_deliveries WHERE webhook_id = $1 ORDER BY created_at DESC LIMIT $2;
