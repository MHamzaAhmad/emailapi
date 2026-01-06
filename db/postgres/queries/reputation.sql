-- name: EnsureUserReputation :exec
INSERT INTO user_reputation (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO NOTHING;

-- name: GetUserReputation :one
SELECT * FROM user_reputation WHERE user_id = $1;

-- name: InsertReputationIncident :exec
INSERT INTO reputation_incidents (
    id, user_id, incident_type, message_id, recipient_email_hash,
    bounce_type, bounce_subtype, complaint_feedback_type, diagnostic_code
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: CountIncidentsByUser :one
SELECT 
    COUNT(*) FILTER (WHERE incident_type = 'bounce_hard')::INTEGER as hard_bounces,
    COUNT(*) FILTER (WHERE incident_type = 'bounce_soft')::INTEGER as soft_bounces,
    COUNT(*) FILTER (WHERE incident_type = 'complaint')::INTEGER as complaints,
    COUNT(*) FILTER (WHERE incident_type IN ('bounce_hard', 'bounce_soft') AND created_at > NOW() - INTERVAL '30 days')::INTEGER as bounces_30d,
    COUNT(*) FILTER (WHERE incident_type = 'complaint' AND created_at > NOW() - INTERVAL '30 days')::INTEGER as complaints_30d
FROM reputation_incidents
WHERE user_id = $1;

-- name: UpdateUserReputationStats :exec
UPDATE user_reputation SET
    total_bounces = $2,
    hard_bounces = $3,
    soft_bounces = $4,
    complaints = $5,
    bounces_30d = $6,
    complaints_30d = $7,
    suspension_score = $8,
    is_flagged = $9,
    flagged_at = CASE WHEN $9 AND NOT is_flagged THEN NOW() ELSE flagged_at END,
    flagged_reason = $10,
    updated_at = NOW()
WHERE user_id = $1;

-- name: SuspendUserReputation :exec
UPDATE user_reputation SET
    is_suspended = TRUE,
    suspended_at = NOW(),
    suspended_by = $2,
    suspension_reason = $3,
    updated_at = NOW()
WHERE user_id = $1;

-- name: UnsuspendUserReputation :exec
UPDATE user_reputation SET
    is_suspended = FALSE,
    is_flagged = FALSE,
    suspended_at = NULL,
    suspended_by = NULL,
    suspension_reason = NULL,
    flagged_at = NULL,
    flagged_reason = NULL,
    updated_at = NOW()
WHERE user_id = $1;

-- name: ListFlaggedUserReputations :many
SELECT ur.*, u.email as user_email, u.name as user_name
FROM user_reputation ur
JOIN users u ON ur.user_id = u.id
WHERE ur.is_flagged = TRUE AND ur.is_suspended = FALSE
ORDER BY ur.suspension_score DESC
LIMIT $1 OFFSET $2;

-- name: ListReputationIncidents :many
SELECT * FROM reputation_incidents
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountFlaggedUsers :one
SELECT COUNT(*)::INTEGER as total
FROM user_reputation
WHERE is_flagged = TRUE AND is_suspended = FALSE;
