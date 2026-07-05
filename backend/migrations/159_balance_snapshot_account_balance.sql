ALTER TABLE balance_key_snapshots
    ADD COLUMN IF NOT EXISTS account_balance JSONB NOT NULL DEFAULT '{}'::jsonb;
