ALTER TABLE balance_sites
    ADD COLUMN IF NOT EXISTS auth_mode VARCHAR(20) NOT NULL DEFAULT 'password',
    ADD COLUMN IF NOT EXISTS access_token_encrypted TEXT NOT NULL DEFAULT '';

ALTER TABLE balance_sites
    ALTER COLUMN password_encrypted SET DEFAULT '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'balance_sites_auth_mode_check'
    ) THEN
        ALTER TABLE balance_sites
            ADD CONSTRAINT balance_sites_auth_mode_check CHECK (auth_mode IN ('password', 'access_token'));
    END IF;
END $$;
