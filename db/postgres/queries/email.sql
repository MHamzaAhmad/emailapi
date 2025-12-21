-- name: CreateEmail :one
INSERT INTO emails (
    id, user_id, from_address, to_addresses, cc_addresses, bcc_addresses,
    subject, body, html, status, metadata, scheduled_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetEmailByID :one
SELECT * FROM emails WHERE id = $1;

-- name: GetEmailsByUserID :many
SELECT * FROM emails 
WHERE user_id = $1 
ORDER BY created_at DESC 
LIMIT $2 OFFSET $3;

-- name: UpdateEmailStatus :one
UPDATE emails 
SET status = $2, error_message = $3, updated_at = NOW() 
WHERE id = $1 
RETURNING *;

-- name: UpdateEmailSent :one
UPDATE emails 
SET status = 'sent', provider_id = $2, sent_at = NOW(), updated_at = NOW() 
WHERE id = $1 
RETURNING *;

-- name: DeleteEmail :exec
DELETE FROM emails WHERE id = $1;

-- name: GetEmailsWithPendingAttachments :many
SELECT DISTINCT e.* FROM emails e
JOIN email_attachments ea ON ea.email_id = e.id
WHERE ea.scan_status IN ('pending', 'scanning')
ORDER BY e.created_at;

-- name: CreateEmailAttachment :one
INSERT INTO email_attachments (
    id, email_id, filename, content_type, s3_key, size_bytes, scan_status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetAttachmentsByEmailID :many
SELECT * FROM email_attachments WHERE email_id = $1;

-- name: GetAttachmentByS3Key :one
SELECT * FROM email_attachments WHERE s3_key = $1;

-- name: UpdateAttachmentScanStatus :one
UPDATE email_attachments 
SET scan_status = $2, scan_checked_at = NOW() 
WHERE id = $1 
RETURNING *;

-- name: CheckAllAttachmentsScanned :one
SELECT 
    COUNT(*) FILTER (WHERE scan_status IN ('pending', 'scanning')) = 0 AS all_scanned,
    COUNT(*) FILTER (WHERE scan_status = 'clean') = COUNT(*) AS all_clean,
    COUNT(*) FILTER (WHERE scan_status = 'threats_found') > 0 AS has_threats
FROM email_attachments 
WHERE email_id = $1;

-- name: DeleteAttachmentsByEmailID :exec
DELETE FROM email_attachments WHERE email_id = $1;

-- name: CountEmailsByUserID :one
SELECT COUNT(*) FROM emails WHERE user_id = $1;

