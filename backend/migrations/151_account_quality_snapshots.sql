CREATE TABLE IF NOT EXISTS account_quality_snapshots (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    total_requests BIGINT NOT NULL DEFAULT 0,
    success_requests BIGINT NOT NULL DEFAULT 0,
    recent_success_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    ttft_sample_count BIGINT NOT NULL DEFAULT 0,
    ttft_le_5s_count BIGINT NOT NULL DEFAULT 0,
    ttft_le_10s_count BIGINT NOT NULL DEFAULT 0,
    ttft_le_5s_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    ttft_le_10s_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    quality_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_quality_snapshots_score
    ON account_quality_snapshots (quality_score DESC, recent_success_rate DESC);

CREATE INDEX IF NOT EXISTS idx_account_quality_snapshots_updated_at
    ON account_quality_snapshots (updated_at DESC);
