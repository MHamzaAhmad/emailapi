-- +goose Up
-- +goose StatementBegin

-- User reputation tracking table
CREATE TABLE user_reputation (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Aggregate counters (updated by async job)
    total_bounces INTEGER NOT NULL DEFAULT 0,
    hard_bounces INTEGER NOT NULL DEFAULT 0,
    soft_bounces INTEGER NOT NULL DEFAULT 0,
    complaints INTEGER NOT NULL DEFAULT 0,
    
    -- Rolling window counters (last 30 days)
    bounces_30d INTEGER NOT NULL DEFAULT 0,
    complaints_30d INTEGER NOT NULL DEFAULT 0,
    
    -- Threshold tracking
    suspension_score DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    is_flagged BOOLEAN NOT NULL DEFAULT FALSE,
    flagged_at TIMESTAMPTZ,
    flagged_reason TEXT,
    
    -- Manual review
    is_suspended BOOLEAN NOT NULL DEFAULT FALSE,
    suspended_at TIMESTAMPTZ,
    suspended_by TEXT,
    suspension_reason TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_user_reputation UNIQUE (user_id)
);

-- Incidents table for detailed tracking
CREATE TABLE reputation_incidents (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    incident_type TEXT NOT NULL,  -- 'bounce_hard', 'bounce_soft', 'complaint'
    message_id TEXT NOT NULL,
    recipient_email_hash TEXT NOT NULL,
    
    -- SES-provided details
    bounce_type TEXT,
    bounce_subtype TEXT,
    complaint_feedback_type TEXT,
    diagnostic_code TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_reputation_incidents_user ON reputation_incidents(user_id);
CREATE INDEX idx_reputation_incidents_created ON reputation_incidents(created_at);
CREATE INDEX idx_user_reputation_flagged ON user_reputation(is_flagged) WHERE is_flagged = TRUE;
CREATE INDEX idx_user_reputation_suspended ON user_reputation(is_suspended) WHERE is_suspended = TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS reputation_incidents;
DROP TABLE IF EXISTS user_reputation;
-- +goose StatementEnd
