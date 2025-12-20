-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    api_key_hash TEXT,
    api_key_prefix TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_api_key_hash ON users(api_key_hash) WHERE api_key_hash IS NOT NULL;

-- Create emails table
CREATE TABLE IF NOT EXISTS emails (
    id TEXT PRIMARY KEY,
    from_address TEXT NOT NULL,
    to_addresses TEXT[] NOT NULL,
    cc_addresses TEXT[],
    bcc_addresses TEXT[],
    subject TEXT NOT NULL,
    body TEXT,
    html_body TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    provider_id TEXT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    webhook_id TEXT,
    metadata JSONB,
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_emails_user_id ON emails(user_id);
CREATE INDEX idx_emails_status ON emails(status);
CREATE INDEX idx_emails_created_at ON emails(created_at DESC);

-- Create webhooks table
CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    secret_hash TEXT NOT NULL,
    events TEXT[] NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    retry_count INTEGER NOT NULL DEFAULT 3,
    last_success TIMESTAMPTZ,
    last_failure TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_user_id ON webhooks(user_id);
CREATE INDEX idx_webhooks_is_active ON webhooks(is_active) WHERE is_active = true;

-- Create webhook_deliveries table
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id TEXT PRIMARY KEY,
    webhook_id TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload TEXT NOT NULL,
    response_code INTEGER,
    response_body TEXT,
    success BOOLEAN NOT NULL DEFAULT false,
    attempt_count INTEGER NOT NULL DEFAULT 1,
    next_retry TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX idx_webhook_deliveries_created_at ON webhook_deliveries(created_at DESC);
