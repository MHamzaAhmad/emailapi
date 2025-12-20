-- atlas:import users.sql


-- Domains table for sending domain management
CREATE TABLE domains (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    verified_for_sending BOOLEAN NOT NULL DEFAULT false,
    dkim_tokens TEXT[],
    dkim_status TEXT NOT NULL DEFAULT 'pending',
    mail_from_domain TEXT,
    mail_from_status TEXT,
    region TEXT NOT NULL DEFAULT 'us-east-1',
    last_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, domain_name)
);

CREATE INDEX idx_domains_user_id ON domains(user_id);
CREATE INDEX idx_domains_domain_name ON domains(domain_name);
CREATE INDEX idx_domains_status ON domains(status);
CREATE INDEX idx_domains_last_verified_at ON domains(last_verified_at);

