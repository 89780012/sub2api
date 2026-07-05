CREATE TABLE IF NOT EXISTS balance_sites (
    id BIGSERIAL PRIMARY KEY,
    platform VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    username VARCHAR(255) NOT NULL DEFAULT '',
    email VARCHAR(255) NOT NULL DEFAULT '',
    password_encrypted TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    refresh_interval_minutes INTEGER NOT NULL DEFAULT 180,
    last_refresh_at TIMESTAMPTZ,
    last_refresh_status VARCHAR(20) NOT NULL DEFAULT '',
    last_refresh_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT balance_sites_platform_check CHECK (platform IN ('newapi', 'sub2api')),
    CONSTRAINT balance_sites_refresh_interval_check CHECK (refresh_interval_minutes > 0)
);

CREATE TABLE IF NOT EXISTS balance_key_snapshots (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT NOT NULL REFERENCES balance_sites(id) ON DELETE CASCADE,
    external_key_id VARCHAR(255) NOT NULL DEFAULT '',
    masked_key VARCHAR(255) NOT NULL DEFAULT '',
    key_last4 VARCHAR(16) NOT NULL DEFAULT '',
    account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    match_status VARCHAR(20) NOT NULL DEFAULT 'unmatched',
    key_name VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT '',
    group_id VARCHAR(255) NOT NULL DEFAULT '',
    group_name VARCHAR(255) NOT NULL DEFAULT '',
    balance JSONB NOT NULL DEFAULT '{}'::jsonb,
    subscription_balance JSONB NOT NULL DEFAULT '{}'::jsonb,
    rate_multiplier NUMERIC(10,4),
    fetched_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT balance_key_snapshots_match_status_check CHECK (match_status IN ('unmatched', 'matched', 'ambiguous', 'manual'))
);

CREATE TABLE IF NOT EXISTS account_balance_bindings (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    site_id BIGINT NOT NULL REFERENCES balance_sites(id) ON DELETE CASCADE,
    external_key_id VARCHAR(255),
    key_last4 VARCHAR(16) NOT NULL DEFAULT '',
    match_mode VARCHAR(20) NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT account_balance_bindings_match_mode_check CHECK (match_mode IN ('auto', 'manual'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_account_balance_bindings_account
    ON account_balance_bindings(account_id);

CREATE INDEX IF NOT EXISTS idx_balance_sites_enabled_refresh
    ON balance_sites(enabled, last_refresh_at);

CREATE INDEX IF NOT EXISTS idx_balance_key_snapshots_site_external_key
    ON balance_key_snapshots(site_id, external_key_id);

CREATE INDEX IF NOT EXISTS idx_balance_key_snapshots_key_last4
    ON balance_key_snapshots(key_last4);

CREATE INDEX IF NOT EXISTS idx_balance_key_snapshots_account
    ON balance_key_snapshots(account_id, fetched_at DESC);

CREATE INDEX IF NOT EXISTS idx_account_balance_bindings_site_key
    ON account_balance_bindings(site_id, key_last4);
