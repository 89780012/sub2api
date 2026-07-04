INSERT INTO settings (key, value, created_at, updated_at)
VALUES
    ('account_upstream_monitor_enabled', 'true', NOW(), NOW()),
    ('account_upstream_monitor_interval_minutes', '60', NOW(), NOW())
ON CONFLICT (key) DO NOTHING;
