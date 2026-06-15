ALTER TABLE usage_logs
ADD COLUMN IF NOT EXISTS schedule_trace JSONB;
