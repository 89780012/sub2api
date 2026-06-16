ALTER TABLE scheduled_test_results
ADD COLUMN IF NOT EXISTS account_id BIGINT REFERENCES accounts(id) ON DELETE CASCADE,
ADD COLUMN IF NOT EXISTS model_id VARCHAR(100) NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS first_token_ms BIGINT,
ADD COLUMN IF NOT EXISTS has_first_token BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS failure_kind VARCHAR(50) NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS request_type VARCHAR(20) NOT NULL DEFAULT 'stream';

UPDATE scheduled_test_results r
SET
    account_id = p.account_id,
    model_id = p.model_id
FROM scheduled_test_plans p
WHERE r.plan_id = p.id
  AND (r.account_id IS NULL OR r.model_id = '');

CREATE INDEX IF NOT EXISTS idx_str_account_created
    ON scheduled_test_results(account_id, created_at DESC);
