ALTER TABLE groups
ADD COLUMN IF NOT EXISTS primary_account_mode VARCHAR(20) NOT NULL DEFAULT 'off',
ADD COLUMN IF NOT EXISTS manual_primary_account_id BIGINT,
ADD COLUMN IF NOT EXISTS active_primary_account_id BIGINT,
ADD COLUMN IF NOT EXISTS active_primary_source VARCHAR(50) NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS active_primary_reason VARCHAR(255) NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS active_primary_switched_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS primary_failover_cooldown_seconds INTEGER NOT NULL DEFAULT 30,
ADD COLUMN IF NOT EXISTS primary_allow_manual_auto_replace BOOLEAN NOT NULL DEFAULT FALSE;
