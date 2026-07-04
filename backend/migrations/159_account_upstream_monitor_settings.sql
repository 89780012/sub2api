INSERT INTO settings (key, value, updated_at)
VALUES
    ('account_upstream_monitor_enabled', 'true', NOW()),
    ('account_upstream_monitor_interval_minutes', '60', NOW())
ON CONFLICT (key) DO NOTHING;
