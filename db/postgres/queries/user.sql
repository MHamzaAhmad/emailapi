-- name: CreateUser :one
INSERT INTO users (id, email, name, role, is_active, external_id, plan, polar_customer_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetUserByID :one
SELECT id, email, name, role, is_active, external_id, plan, polar_customer_id, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, name, role, is_active, external_id, plan, polar_customer_id, created_at, updated_at
FROM users WHERE email = $1;

-- name: GetUserByExternalID :one
SELECT id, email, name, role, is_active, external_id, plan, polar_customer_id, created_at, updated_at
FROM users WHERE external_id = $1;

-- name: GetUserByPolarCustomerID :one
SELECT id, email, name, role, is_active, external_id, plan, polar_customer_id, created_at, updated_at
FROM users WHERE polar_customer_id = $1;

-- name: UpdateUser :one
UPDATE users SET
    email = $2,
    name = $3,
    role = $4,
    is_active = $5,
    external_id = $6,
    plan = $7,
    polar_customer_id = $8,
    updated_at = $9
WHERE id = $1
RETURNING *;

-- name: UpdateUserPlan :exec
UPDATE users SET plan = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateUserPolarCustomerID :exec
UPDATE users SET polar_customer_id = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, name, role, is_active, external_id, plan, polar_customer_id, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*)::INTEGER as total FROM users;

-- name: ListUsersWithReputation :many
SELECT 
    u.id, u.email, u.name, u.role, u.is_active, u.external_id, u.plan, u.polar_customer_id, u.created_at, u.updated_at,
    COALESCE(r.is_suspended, FALSE) as is_suspended,
    COALESCE(r.is_flagged, FALSE) as is_flagged
FROM users u
LEFT JOIN user_reputation r ON u.id = r.user_id
ORDER BY u.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetUserWithReputation :one
SELECT 
    u.id, u.email, u.name, u.role, u.is_active, u.external_id, u.plan, u.polar_customer_id, u.created_at, u.updated_at,
    COALESCE(r.total_bounces, 0) as total_bounces,
    COALESCE(r.hard_bounces, 0) as hard_bounces,
    COALESCE(r.soft_bounces, 0) as soft_bounces,
    COALESCE(r.complaints, 0) as complaints,
    COALESCE(r.bounces_30d, 0) as bounces_30d,
    COALESCE(r.complaints_30d, 0) as complaints_30d,
    COALESCE(r.suspension_score, 0) as suspension_score,
    COALESCE(r.is_flagged, FALSE) as is_flagged,
    r.flagged_at,
    r.flagged_reason,
    COALESCE(r.is_suspended, FALSE) as is_suspended,
    r.suspended_at,
    r.suspended_by,
    r.suspension_reason
FROM users u
LEFT JOIN user_reputation r ON u.id = r.user_id
WHERE u.id = $1;

