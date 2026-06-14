ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS selection_reason JSONB;

COMMENT ON COLUMN usage_logs.selection_reason IS 'Structured explanation of account/channel selection for this request';
