package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type accountQualityRepository struct {
	db *sql.DB
}

func NewAccountQualityRepository(db *sql.DB) service.AccountQualityRepository {
	return &accountQualityRepository{db: db}
}

func (r *accountQualityRepository) RefreshSnapshots(ctx context.Context, windowStart, windowEnd time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil account quality repository")
	}

	const query = `
WITH usage_success AS (
  SELECT
    ul.account_id,
    COUNT(*)::bigint AS success_requests,
    COUNT(ul.first_token_ms)::bigint AS ttft_sample_count,
    COUNT(*) FILTER (WHERE ul.first_token_ms IS NOT NULL AND ul.first_token_ms <= 5000)::bigint AS ttft_le_5s_count,
    COUNT(*) FILTER (WHERE ul.first_token_ms IS NOT NULL AND ul.first_token_ms <= 10000)::bigint AS ttft_le_10s_count,
    COUNT(*) FILTER (WHERE ul.first_token_ms IS NOT NULL AND ul.first_token_ms > 10000)::bigint AS ttft_gt_10s_count,
    COUNT(*) FILTER (WHERE ul.first_token_ms IS NOT NULL AND ul.first_token_ms > 20000)::bigint AS ttft_gt_20s_count,
    COUNT(*) FILTER (WHERE ul.first_token_ms IS NOT NULL AND ul.first_token_ms > 40000)::bigint AS ttft_gt_40s_count
  FROM usage_logs ul
  WHERE ul.account_id IS NOT NULL
    AND ul.created_at >= $1
    AND ul.created_at < $2
  GROUP BY ul.account_id
),
ops_failures AS (
  SELECT
    oe.account_id,
    COUNT(*)::bigint AS failure_requests
  FROM ops_error_logs oe
  WHERE oe.account_id IS NOT NULL
    AND oe.created_at >= $1
    AND oe.created_at < $2
    AND COALESCE(oe.is_business_limited, false) = false
  GROUP BY oe.account_id
),
combined AS (
  SELECT
    COALESCE(us.account_id, of.account_id) AS account_id,
    COALESCE(us.success_requests, 0)::bigint AS success_requests,
    COALESCE(of.failure_requests, 0)::bigint AS failure_requests,
    COALESCE(us.ttft_sample_count, 0)::bigint AS ttft_sample_count,
    COALESCE(us.ttft_le_5s_count, 0)::bigint AS ttft_le_5s_count,
    COALESCE(us.ttft_le_10s_count, 0)::bigint AS ttft_le_10s_count,
    COALESCE(us.ttft_gt_10s_count, 0)::bigint AS ttft_gt_10s_count,
    COALESCE(us.ttft_gt_20s_count, 0)::bigint AS ttft_gt_20s_count,
    COALESCE(us.ttft_gt_40s_count, 0)::bigint AS ttft_gt_40s_count
  FROM usage_success us
  FULL OUTER JOIN ops_failures of ON of.account_id = us.account_id
),
scored AS (
  SELECT
    account_id,
    success_requests + failure_requests AS total_requests,
    success_requests,
    failure_requests,
    CASE WHEN success_requests + failure_requests > 0
      THEN success_requests::double precision / (success_requests + failure_requests)::double precision
      ELSE 0 END AS recent_success_rate,
    CASE WHEN success_requests + failure_requests > 0
      THEN failure_requests::double precision / (success_requests + failure_requests)::double precision
      ELSE 0 END AS error_rate,
    ttft_sample_count,
    ttft_le_5s_count,
    ttft_le_10s_count,
    ttft_gt_10s_count,
    ttft_gt_20s_count,
    ttft_gt_40s_count,
    CASE WHEN ttft_sample_count > 0
      THEN ttft_le_5s_count::double precision / ttft_sample_count::double precision
      ELSE 0 END AS ttft_le_5s_rate,
    CASE WHEN ttft_sample_count > 0
      THEN ttft_le_10s_count::double precision / ttft_sample_count::double precision
      ELSE 0 END AS ttft_le_10s_rate,
    CASE WHEN ttft_sample_count > 0
      THEN ttft_gt_10s_count::double precision / ttft_sample_count::double precision
      ELSE 0 END AS ttft_gt_10s_rate,
    CASE WHEN ttft_sample_count > 0
      THEN ttft_gt_20s_count::double precision / ttft_sample_count::double precision
      ELSE 0 END AS ttft_gt_20s_rate,
    CASE WHEN ttft_sample_count > 0
      THEN ttft_gt_40s_count::double precision / ttft_sample_count::double precision
      ELSE 0 END AS ttft_gt_40s_rate,
    LEAST((success_requests + failure_requests)::double precision / 12.0, 1.0) AS sample_confidence
  FROM combined
)
INSERT INTO account_quality_snapshots (
  account_id,
  window_start,
  window_end,
  total_requests,
  success_requests,
  failure_requests,
  recent_success_rate,
  error_rate,
  ttft_sample_count,
  ttft_le_5s_count,
  ttft_le_10s_count,
  ttft_gt_10s_count,
  ttft_gt_20s_count,
  ttft_gt_40s_count,
  ttft_le_5s_rate,
  ttft_le_10s_rate,
  ttft_gt_10s_rate,
  ttft_gt_20s_rate,
  ttft_gt_40s_rate,
  sample_confidence,
  fast_bonus,
  slow_penalty,
  error_penalty,
  neutral_base,
  quality_score,
  updated_at
)
SELECT
  account_id,
  $1,
  $2,
  total_requests,
  success_requests,
  failure_requests,
  recent_success_rate,
  error_rate,
  ttft_sample_count,
  ttft_le_5s_count,
  ttft_le_10s_count,
  ttft_gt_10s_count,
  ttft_gt_20s_count,
  ttft_gt_40s_count,
  ttft_le_5s_rate,
  ttft_le_10s_rate,
  ttft_gt_10s_rate,
  ttft_gt_20s_rate,
  ttft_gt_40s_rate,
  sample_confidence,
  ROUND((0.20 * sample_confidence * ((0.70 * ttft_le_5s_rate) + (0.30 * ttft_le_10s_rate)))::numeric, 4)::double precision,
  ROUND((sample_confidence * ((0.10 * ttft_gt_10s_rate) + (0.30 * ttft_gt_20s_rate) + (0.60 * ttft_gt_40s_rate)))::numeric, 4)::double precision,
  ROUND((0.70 * error_rate)::numeric, 4)::double precision,
  0.60,
  ROUND(LEAST(GREATEST(0.60 + (0.25 * recent_success_rate) + (0.35 * ttft_le_5s_rate) + (0.20 * ttft_le_10s_rate) + (0.20 * sample_confidence * ((0.70 * ttft_le_5s_rate) + (0.30 * ttft_le_10s_rate))) - (sample_confidence * ((0.10 * ttft_gt_10s_rate) + (0.30 * ttft_gt_20s_rate) + (0.60 * ttft_gt_40s_rate))) - (0.70 * error_rate), 0), 1)::numeric, 4)::double precision,
  NOW()
FROM scored
ON CONFLICT (account_id) DO UPDATE SET
  window_start = EXCLUDED.window_start,
  window_end = EXCLUDED.window_end,
  total_requests = EXCLUDED.total_requests,
  success_requests = EXCLUDED.success_requests,
  failure_requests = EXCLUDED.failure_requests,
  recent_success_rate = EXCLUDED.recent_success_rate,
  error_rate = EXCLUDED.error_rate,
  ttft_sample_count = EXCLUDED.ttft_sample_count,
  ttft_le_5s_count = EXCLUDED.ttft_le_5s_count,
  ttft_le_10s_count = EXCLUDED.ttft_le_10s_count,
  ttft_gt_10s_count = EXCLUDED.ttft_gt_10s_count,
  ttft_gt_20s_count = EXCLUDED.ttft_gt_20s_count,
  ttft_gt_40s_count = EXCLUDED.ttft_gt_40s_count,
  ttft_le_5s_rate = EXCLUDED.ttft_le_5s_rate,
  ttft_le_10s_rate = EXCLUDED.ttft_le_10s_rate,
  ttft_gt_10s_rate = EXCLUDED.ttft_gt_10s_rate,
  ttft_gt_20s_rate = EXCLUDED.ttft_gt_20s_rate,
  ttft_gt_40s_rate = EXCLUDED.ttft_gt_40s_rate,
  sample_confidence = EXCLUDED.sample_confidence,
  fast_bonus = EXCLUDED.fast_bonus,
  slow_penalty = EXCLUDED.slow_penalty,
  error_penalty = EXCLUDED.error_penalty,
  neutral_base = EXCLUDED.neutral_base,
  quality_score = EXCLUDED.quality_score,
  updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query, windowStart.UTC(), windowEnd.UTC())
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM account_quality_snapshots WHERE window_end < $1`, windowEnd.UTC())
	return err
}

func (r *accountQualityRepository) GetSnapshotsByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]*service.AccountQualitySnapshot, error) {
	out := make(map[int64]*service.AccountQualitySnapshot, len(accountIDs))
	if r == nil || r.db == nil || len(accountIDs) == 0 {
		return out, nil
	}

	const query = `
SELECT
  account_id,
  window_start,
  window_end,
  total_requests,
  success_requests,
  failure_requests,
  recent_success_rate,
  error_rate,
  ttft_sample_count,
  ttft_le_5s_count,
  ttft_le_10s_count,
  ttft_gt_10s_count,
  ttft_gt_20s_count,
  ttft_gt_40s_count,
  ttft_le_5s_rate,
  ttft_le_10s_rate,
  ttft_gt_10s_rate,
  ttft_gt_20s_rate,
  ttft_gt_40s_rate,
  sample_confidence,
  fast_bonus,
  slow_penalty,
  error_penalty,
  neutral_base,
  quality_score,
  updated_at
FROM account_quality_snapshots
WHERE account_id = ANY($1)`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(accountIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		snapshot := &service.AccountQualitySnapshot{}
		if err := rows.Scan(
			&snapshot.AccountID,
			&snapshot.WindowStart,
			&snapshot.WindowEnd,
			&snapshot.TotalRequests,
			&snapshot.SuccessRequests,
			&snapshot.FailureRequests,
			&snapshot.RecentSuccessRate,
			&snapshot.ErrorRate,
			&snapshot.TTFTSampleCount,
			&snapshot.TTFTLE5sCount,
			&snapshot.TTFTLE10sCount,
			&snapshot.TTFTGT10sCount,
			&snapshot.TTFTGT20sCount,
			&snapshot.TTFTGT40sCount,
			&snapshot.TTFTLE5sRate,
			&snapshot.TTFTLE10sRate,
			&snapshot.TTFTGT10sRate,
			&snapshot.TTFTGT20sRate,
			&snapshot.TTFTGT40sRate,
			&snapshot.SampleConfidence,
			&snapshot.FastBonus,
			&snapshot.SlowPenalty,
			&snapshot.ErrorPenalty,
			&snapshot.NeutralBase,
			&snapshot.QualityScore,
			&snapshot.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out[snapshot.AccountID] = snapshot
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
