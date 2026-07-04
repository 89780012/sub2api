CREATE TABLE IF NOT EXISTS account_upstream_monitor_snapshots (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL DEFAULT 'unknown',
    status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    site_url VARCHAR(512),
    balance DECIMAL(20,8),
    balance_unit VARCHAR(32),
    quota DECIMAL(20,8),
    quota_used DECIMAL(20,8),
    today_cost DECIMAL(20,8),
    total_cost DECIMAL(20,8),
    last_checked_at TIMESTAMPTZ,
    last_success_at TIMESTAMPTZ,
    last_error TEXT,
    raw_meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_upstream_monitor_snapshots_status_checked
    ON account_upstream_monitor_snapshots (status, last_checked_at);

CREATE INDEX IF NOT EXISTS idx_account_upstream_monitor_snapshots_provider
    ON account_upstream_monitor_snapshots (provider);

CREATE TABLE IF NOT EXISTS account_upstream_monitor_rates (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL DEFAULT 'sub2api',
    rate_key VARCHAR(256) NOT NULL,
    display_name VARCHAR(256) NOT NULL,
    description VARCHAR(512),
    ratio DECIMAL(20,8) NOT NULL,
    completion_ratio DECIMAL(20,8),
    first_seen_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT account_upstream_monitor_rates_account_rate_key_key UNIQUE (account_id, rate_key)
);

CREATE INDEX IF NOT EXISTS idx_account_upstream_monitor_rates_account_seen
    ON account_upstream_monitor_rates (account_id, last_seen_at);
