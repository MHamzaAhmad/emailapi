-- name: CreateDomain :exec
INSERT INTO domains (
    id, user_id, domain_name, status, verified_for_sending,
    dkim_tokens, dkim_status, mail_from_domain, mail_from_status,
    region, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
);

-- name: GetDomainByID :one
SELECT * FROM domains WHERE id = $1;

-- name: GetDomainByName :one
SELECT * FROM domains WHERE user_id = $1 AND domain_name = $2;

-- name: GetDomainsByUserID :many
SELECT * FROM domains WHERE user_id = $1 ORDER BY created_at DESC;

-- name: UpdateDomain :exec
UPDATE domains SET
    status = $2,
    verified_for_sending = $3,
    dkim_tokens = $4,
    dkim_status = $5,
    mail_from_domain = $6,
    mail_from_status = $7,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteDomain :exec
DELETE FROM domains WHERE id = $1;
