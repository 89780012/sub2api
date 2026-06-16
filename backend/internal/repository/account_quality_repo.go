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
real_base AS (
  SELECT
    COALESCE(us.account_id, opf.account_id) AS account_id,
    COALESCE(us.success_requests, 0)::bigint AS success_requests,
    COALESCE(opf.failure_requests, 0)::bigint AS failure_requests,
    COALESCE(us.ttft_sample_count, 0)::bigint AS ttft_sample_count,
    COALESCE(us.ttft_le_5s_count, 0)::bigint AS ttft_le_5s_count,
    COALESCE(us.ttft_le_10s_count, 0)::bigint AS ttft_le_10s_count,
    COALESCE(us.ttft_gt_10s_count, 0)::bigint AS ttft_gt_10s_count,
    COALESCE(us.ttft_gt_20s_count, 0)::bigint AS ttft_gt_20s_count,
    COALESCE(us.ttft_gt_40s_count, 0)::bigint AS ttft_gt_40s_count
  FROM usage_success us
  FULL OUTER JOIN ops_failures opf ON opf.account_id = us.account_id
),
scheduled_test AS (
  SELECT
    str.account_id,
    COUNT(*)::bigint AS total_requests,
    COUNT(*) FILTER (WHERE str.status = 'success')::bigint AS success_requests,
    COUNT(*) FILTER (WHERE str.status <> 'success')::bigint AS failure_requests,
    COUNT(str.first_token_ms)::bigint AS ttft_sample_count,
    COUNT(*) FILTER (WHERE str.first_token_ms IS NOT NULL AND str.first_token_ms <= 5000)::bigint AS ttft_le_5s_count,
    COUNT(*) FILTER (WHERE str.first_token_ms IS NOT NULL AND str.first_token_ms <= 10000)::bigint AS ttft_le_10s_count,
    COUNT(*) FILTER (WHERE str.first_token_ms IS NOT NULL AND str.first_token_ms > 10000)::bigint AS ttft_gt_10s_count,
    COUNT(*) FILTER (WHERE str.first_token_ms IS NOT NULL AND str.first_token_ms > 20000)::bigint AS ttft_gt_20s_count,
    COUNT(*) FILTER (WHERE str.first_token_ms IS NOT NULL AND str.first_token_ms > 40000)::bigint AS ttft_gt_40s_count
  FROM scheduled_test_results str
  WHERE str.account_id IS NOT NULL
    AND str.created_at >= $1
    AND str.created_at < $2
    AND str.request_type = 'stream'
  GROUP BY str.account_id
),
base AS (
  SELECT
    COALESCE(rb.account_id, st.account_id) AS account_id,
    COALESCE(rb.success_requests, 0)::bigint AS real_success_requests,
    COALESCE(rb.failure_requests, 0)::bigint AS real_failure_requests,
    COALESCE(rb.ttft_sample_count, 0)::bigint AS real_ttft_sample_count,
    COALESCE(rb.ttft_le_5s_count, 0)::bigint AS real_ttft_le_5s_count,
    COALESCE(rb.ttft_le_10s_count, 0)::bigint AS real_ttft_le_10s_count,
    COALESCE(rb.ttft_gt_10s_count, 0)::bigint AS real_ttft_gt_10s_count,
    COALESCE(rb.ttft_gt_20s_count, 0)::bigint AS real_ttft_gt_20s_count,
    COALESCE(rb.ttft_gt_40s_count, 0)::bigint AS real_ttft_gt_40s_count,
    COALESCE(st.total_requests, 0)::bigint AS auxiliary_total_requests,
    COALESCE(st.success_requests, 0)::bigint AS auxiliary_success_requests,
    COALESCE(st.failure_requests, 0)::bigint AS auxiliary_failure_requests,
    COALESCE(st.ttft_sample_count, 0)::bigint AS auxiliary_ttft_sample_count,
    COALESCE(st.ttft_le_5s_count, 0)::bigint AS auxiliary_ttft_le_5s_count,
    COALESCE(st.ttft_le_10s_count, 0)::bigint AS auxiliary_ttft_le_10s_count,
    COALESCE(st.ttft_gt_10s_count, 0)::bigint AS auxiliary_ttft_gt_10s_count,
    COALESCE(st.ttft_gt_20s_count, 0)::bigint AS auxiliary_ttft_gt_20s_count,
    COALESCE(st.ttft_gt_40s_count, 0)::bigint AS auxiliary_ttft_gt_40s_count
  FROM real_base rb
  FULL OUTER JOIN scheduled_test st ON st.account_id = rb.account_id
),
weighted AS (
  SELECT
    account_id,
    real_success_requests,
    real_failure_requests,
    real_ttft_sample_count,
    real_ttft_le_5s_count,
    real_ttft_le_10s_count,
    real_ttft_gt_10s_count,
    real_ttft_gt_20s_count,
    real_ttft_gt_40s_count,
    auxiliary_total_requests,
    auxiliary_success_requests,
    auxiliary_failure_requests,
    auxiliary_ttft_sample_count,
    auxiliary_ttft_le_5s_count,
    auxiliary_ttft_le_10s_count,
    auxiliary_ttft_gt_10s_count,
    auxiliary_ttft_gt_20s_count,
    auxiliary_ttft_gt_40s_count,
    (real_success_requests + real_failure_requests)::bigint AS real_total_requests,
    LEAST(auxiliary_total_requests::double precision / 6.0, 1.0) *
      CASE
        WHEN (real_success_requests + real_failure_requests) >= 3 THEN 0.15
        ELSE 0.35
      END AS auxiliary_weight
  FROM base
),
scored AS (
  SELECT
    account_id,
    (real_success_requests + real_failure_requests + auxiliary_total_requests)::bigint AS total_requests,
    (real_success_requests + auxiliary_success_requests)::bigint AS success_requests,
    (real_failure_requests + auxiliary_failure_requests)::bigint AS failure_requests,
    CASE
      WHEN weighted_success_requests + weighted_failure_requests > 0
        THEN weighted_success_requests / (weighted_success_requests + weighted_failure_requests)
      ELSE 0
    END AS recent_success_rate,
    CASE
      WHEN weighted_success_requests + weighted_failure_requests > 0
        THEN weighted_failure_requests / (weighted_success_requests + weighted_failure_requests)
      ELSE 0
    END AS error_rate,
    (real_ttft_sample_count + auxiliary_ttft_sample_count)::bigint AS ttft_sample_count,
    (real_ttft_le_5s_count + auxiliary_ttft_le_5s_count)::bigint AS ttft_le_5s_count,
    (real_ttft_le_10s_count + auxiliary_ttft_le_10s_count)::bigint AS ttft_le_10s_count,
    (real_ttft_gt_10s_count + auxiliary_ttft_gt_10s_count)::bigint AS ttft_gt_10s_count,
    (real_ttft_gt_20s_count + auxiliary_ttft_gt_20s_count)::bigint AS ttft_gt_20s_count,
    (real_ttft_gt_40s_count + auxiliary_ttft_gt_40s_count)::bigint AS ttft_gt_40s_count,
    CASE
      WHEN weighted_ttft_sample_count > 0
        THEN weighted_ttft_le_5s_count / weighted_ttft_sample_count
      ELSE 0
    END AS ttft_le_5s_rate,
    CASE
      WHEN weighted_ttft_sample_count > 0
        THEN weighted_ttft_le_10s_count / weighted_ttft_sample_count
      ELSE 0
    END AS ttft_le_10s_rate,
    CASE
      WHEN weighted_ttft_sample_count > 0
        THEN weighted_ttft_gt_10s_count / weighted_ttft_sample_count
      ELSE 0
    END AS ttft_gt_10s_rate,
    CASE
      WHEN weighted_ttft_sample_count > 0
        THEN weighted_ttft_gt_20s_count / weighted_ttft_sample_count
      ELSE 0
    END AS ttft_gt_20s_rate,
    CASE
      WHEN weighted_ttft_sample_count > 0
        THEN weighted_ttft_gt_40s_count / weighted_ttft_sample_count
      ELSE 0
    END AS ttft_gt_40s_rate,
    LEAST((real_total_requests::double precision + (auxiliary_total_requests::double precision * auxiliary_weight)) / 12.0, 1.0) AS sample_confidence,
    auxiliary_total_requests,
    auxiliary_success_requests,
    auxiliary_failure_requests,
    auxiliary_ttft_sample_count,
    auxiliary_ttft_le_5s_count,
    auxiliary_ttft_le_10s_count,
    auxiliary_ttft_gt_10s_count,
    auxiliary_ttft_gt_20s_count,
    auxiliary_ttft_gt_40s_count,
    auxiliary_weight
  FROM (
    SELECT
      *,
      real_success_requests::double precision + (auxiliary_success_requests::double precision * auxiliary_weight) AS weighted_success_requests,
      real_failure_requests::double precision + (auxiliary_failure_requests::double precision * auxiliary_weight) AS weighted_failure_requests,
      real_ttft_sample_count::double precision + (auxiliary_ttft_sample_count::double precision * auxiliary_weight) AS weighted_ttft_sample_count,
      real_ttft_le_5s_count::double precision + (auxiliary_ttft_le_5s_count::double precision * auxiliary_weight) AS weighted_ttft_le_5s_count,
      real_ttft_le_10s_count::double precision + (auxiliary_ttft_le_10s_count::double precision * auxiliary_weight) AS weighted_ttft_le_10s_count,
      real_ttft_gt_10s_count::double precision + (auxiliary_ttft_gt_10s_count::double precision * auxiliary_weight) AS weighted_ttft_gt_10s_count,
      real_ttft_gt_20s_count::double precision + (auxiliary_ttft_gt_20s_count::double precision * auxiliary_weight) AS weighted_ttft_gt_20s_count,
      real_ttft_gt_40s_count::double precision + (auxiliary_ttft_gt_40s_count::double precision * auxiliary_weight) AS weighted_ttft_gt_40s_count
    FROM weighted
  ) weighted_counts
),
event_stream AS (
  SELECT
    ul.account_id,
    ul.created_at,
    ul.first_token_ms::bigint AS ttft_ms,
    0::int AS is_error,
    CASE WHEN ul.first_token_ms IS NOT NULL AND ul.first_token_ms > 10000 THEN 1 ELSE 0 END::int AS is_slow
  FROM usage_logs ul
  WHERE ul.account_id IS NOT NULL
    AND ul.created_at >= $1
    AND ul.created_at < $2
  UNION ALL
  SELECT
    oe.account_id,
    oe.created_at,
    NULL::bigint AS ttft_ms,
    1::int AS is_error,
    0::int AS is_slow
  FROM ops_error_logs oe
  WHERE oe.account_id IS NOT NULL
    AND oe.created_at >= $1
    AND oe.created_at < $2
    AND COALESCE(oe.is_business_limited, false) = false
  UNION ALL
  SELECT
    str.account_id,
    str.created_at,
    str.first_token_ms::bigint AS ttft_ms,
    CASE WHEN str.status = 'success' THEN 0 ELSE 1 END::int AS is_error,
    CASE WHEN str.status = 'success' AND str.first_token_ms IS NOT NULL AND str.first_token_ms > 10000 THEN 1 ELSE 0 END::int AS is_slow
  FROM scheduled_test_results str
  WHERE str.account_id IS NOT NULL
    AND str.created_at >= $1
    AND str.created_at < $2
    AND str.request_type = 'stream'
),
ordered_events AS (
  SELECT
    es.account_id,
    es.created_at,
    es.ttft_ms,
    es.is_error,
    es.is_slow,
    SUM(CASE WHEN es.is_error = 0 THEN 1 ELSE 0 END) OVER (PARTITION BY es.account_id ORDER BY es.created_at, COALESCE(es.ttft_ms, 0)) AS error_group_id,
    SUM(CASE WHEN es.is_slow = 0 THEN 1 ELSE 0 END) OVER (PARTITION BY es.account_id ORDER BY es.created_at, COALESCE(es.ttft_ms, 0)) AS slow_group_id
  FROM event_stream es
),
error_runs AS (
  SELECT
    account_id,
    MAX(created_at) AS end_at,
    COUNT(*)::bigint AS streak
  FROM ordered_events
  WHERE is_error = 1
  GROUP BY account_id, error_group_id
  HAVING COUNT(*) >= 2
),
latest_error_run AS (
  SELECT DISTINCT ON (account_id)
    account_id,
    end_at,
    streak
  FROM error_runs
  ORDER BY account_id, end_at DESC, streak DESC
),
slow_runs AS (
  SELECT
    account_id,
    MAX(created_at) AS end_at,
    COUNT(*)::bigint AS streak,
    COUNT(*) FILTER (WHERE ttft_ms > 20000)::bigint AS severe20_count,
    COUNT(*) FILTER (WHERE ttft_ms > 40000)::bigint AS severe40_count
  FROM ordered_events
  WHERE is_slow = 1
  GROUP BY account_id, slow_group_id
  HAVING COUNT(*) >= 3
),
latest_slow_run AS (
  SELECT DISTINCT ON (account_id)
    account_id,
    end_at,
    streak,
    severe20_count,
    severe40_count
  FROM slow_runs
  ORDER BY account_id, end_at DESC, streak DESC
),
aux_ordered_desc AS (
  SELECT
    str.account_id,
    str.created_at,
    str.id,
    CASE WHEN str.status = 'success' THEN 1 ELSE 0 END::int AS is_success,
    CASE WHEN str.status = 'success' AND str.first_token_ms IS NOT NULL AND str.first_token_ms <= 10000 THEN 1 ELSE 0 END::int AS is_fast,
    SUM(CASE WHEN str.status = 'success' THEN 0 ELSE 1 END) OVER (PARTITION BY str.account_id ORDER BY str.created_at DESC, str.id DESC) AS success_breaks,
    SUM(CASE WHEN str.status = 'success' AND str.first_token_ms IS NOT NULL AND str.first_token_ms <= 10000 THEN 0 ELSE 1 END) OVER (PARTITION BY str.account_id ORDER BY str.created_at DESC, str.id DESC) AS fast_breaks
  FROM scheduled_test_results str
  WHERE str.account_id IS NOT NULL
    AND str.created_at >= $1
    AND str.created_at < $2
    AND str.request_type = 'stream'
),
aux_success_head AS (
  SELECT account_id, created_at
  FROM aux_ordered_desc
  WHERE is_success = 1
    AND success_breaks = 0
),
aux_fast_head AS (
  SELECT account_id, created_at
  FROM aux_ordered_desc
  WHERE is_fast = 1
    AND fast_breaks = 0
),
penalties AS (
  SELECT
    s.account_id,
    COALESCE(ler.streak, 0)::bigint AS error_streak,
    COALESCE(lsr.streak, 0)::bigint AS slow_streak,
    COALESCE(lsr.severe20_count, 0)::bigint AS slow_streak_gt20_count,
    COALESCE(lsr.severe40_count, 0)::bigint AS slow_streak_gt40_count,
    COALESCE((SELECT COUNT(*) FROM aux_success_head ash WHERE ash.account_id = s.account_id), 0)::bigint AS recovery_success_streak,
    COALESCE((SELECT COUNT(*) FROM aux_fast_head afh WHERE afh.account_id = s.account_id), 0)::bigint AS recovery_fast_streak,
    COALESCE((SELECT COUNT(*) FROM aux_success_head ash WHERE ler.end_at IS NOT NULL AND ash.account_id = s.account_id AND ash.created_at > ler.end_at), 0)::bigint AS recovery_success_after_error,
    COALESCE((SELECT COUNT(*) FROM aux_fast_head afh WHERE lsr.end_at IS NOT NULL AND afh.account_id = s.account_id AND afh.created_at > lsr.end_at), 0)::bigint AS recovery_fast_after_slow
  FROM scored s
  LEFT JOIN latest_error_run ler ON ler.account_id = s.account_id
  LEFT JOIN latest_slow_run lsr ON lsr.account_id = s.account_id
),
aggregate_scores AS (
  SELECT
    s.*,
    p.error_streak,
    p.slow_streak,
    p.recovery_success_streak,
    p.recovery_fast_streak,
    p.recovery_success_after_error,
    p.recovery_fast_after_slow,
    ROUND((0.20 * s.sample_confidence * ((0.70 * s.ttft_le_5s_rate) + (0.30 * s.ttft_le_10s_rate)))::numeric, 4)::double precision AS fast_bonus,
    ROUND((s.sample_confidence * ((0.10 * s.ttft_gt_10s_rate) + (0.30 * s.ttft_gt_20s_rate) + (0.60 * s.ttft_gt_40s_rate)))::numeric, 4)::double precision AS slow_penalty,
    ROUND((0.70 * s.error_rate)::numeric, 4)::double precision AS error_penalty,
    0.60::double precision AS neutral_base,
    ROUND(
      LEAST(
        GREATEST(
          0.60
          + (0.25 * s.recent_success_rate)
          + (0.35 * s.ttft_le_5s_rate)
          + (0.20 * s.ttft_le_10s_rate)
          + (0.20 * s.sample_confidence * ((0.70 * s.ttft_le_5s_rate) + (0.30 * s.ttft_le_10s_rate)))
          - (s.sample_confidence * ((0.10 * s.ttft_gt_10s_rate) + (0.30 * s.ttft_gt_20s_rate) + (0.60 * s.ttft_gt_40s_rate)))
          - (0.70 * s.error_rate),
          0
        ),
        1
      )::numeric,
      4
    )::double precision AS base_quality_score,
    LEAST(
      CASE
        WHEN p.error_streak >= 2 THEN 0.38 + (LEAST((p.error_streak - 2)::double precision, 3.0) * 0.09)
        ELSE 0
      END,
      0.55
    ) AS error_burst_penalty,
    LEAST(
      CASE
        WHEN p.slow_streak >= 3 THEN
          0.22
          + (LEAST((p.slow_streak - 3)::double precision, 4.0) * 0.06)
          + CASE WHEN p.slow_streak_gt20_count > 0 THEN 0.05 ELSE 0 END
          + CASE WHEN p.slow_streak_gt40_count > 0 THEN 0.07 ELSE 0 END
        ELSE 0
      END,
      0.45
    ) AS slow_burst_penalty
  FROM scored s
  LEFT JOIN penalties p ON p.account_id = s.account_id
),
final_scores AS (
  SELECT
    a.*,
    ROUND(LEAST(a.error_burst_penalty + a.slow_burst_penalty, 0.60)::numeric, 4)::double precision AS transient_penalty,
    ROUND(
      LEAST(
        LEAST(a.error_burst_penalty, a.recovery_success_after_error::double precision * 0.14)
        + LEAST(a.slow_burst_penalty, a.recovery_fast_after_slow::double precision * 0.10),
        LEAST(a.error_burst_penalty + a.slow_burst_penalty, 0.45)
      )::numeric,
      4
    )::double precision AS recovery_credit
  FROM aggregate_scores a
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
  base_quality_score,
  effective_quality_score,
  transient_penalty,
  recovery_credit,
  slow_streak,
  error_streak,
  recovery_success_streak,
  recovery_fast_streak,
  quality_score,
  auxiliary_total_requests,
  auxiliary_success_requests,
  auxiliary_failure_requests,
  auxiliary_ttft_sample_count,
  auxiliary_ttft_le_5s_count,
  auxiliary_ttft_le_10s_count,
  auxiliary_ttft_gt_10s_count,
  auxiliary_ttft_gt_20s_count,
  auxiliary_ttft_gt_40s_count,
  auxiliary_weight,
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
  fast_bonus,
  slow_penalty,
  error_penalty,
  neutral_base,
  base_quality_score,
  ROUND(LEAST(GREATEST(base_quality_score - transient_penalty + recovery_credit, 0), 1)::numeric, 4)::double precision AS effective_quality_score,
  transient_penalty,
  recovery_credit,
  slow_streak,
  error_streak,
  recovery_success_streak,
  recovery_fast_streak,
  ROUND(LEAST(GREATEST(base_quality_score - transient_penalty + recovery_credit, 0), 1)::numeric, 4)::double precision AS quality_score,
  auxiliary_total_requests,
  auxiliary_success_requests,
  auxiliary_failure_requests,
  auxiliary_ttft_sample_count,
  auxiliary_ttft_le_5s_count,
  auxiliary_ttft_le_10s_count,
  auxiliary_ttft_gt_10s_count,
  auxiliary_ttft_gt_20s_count,
  auxiliary_ttft_gt_40s_count,
  auxiliary_weight,
  NOW()
FROM final_scores
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
  base_quality_score = EXCLUDED.base_quality_score,
  effective_quality_score = EXCLUDED.effective_quality_score,
  transient_penalty = EXCLUDED.transient_penalty,
  recovery_credit = EXCLUDED.recovery_credit,
  slow_streak = EXCLUDED.slow_streak,
  error_streak = EXCLUDED.error_streak,
  recovery_success_streak = EXCLUDED.recovery_success_streak,
  recovery_fast_streak = EXCLUDED.recovery_fast_streak,
  quality_score = EXCLUDED.quality_score,
  auxiliary_total_requests = EXCLUDED.auxiliary_total_requests,
  auxiliary_success_requests = EXCLUDED.auxiliary_success_requests,
  auxiliary_failure_requests = EXCLUDED.auxiliary_failure_requests,
  auxiliary_ttft_sample_count = EXCLUDED.auxiliary_ttft_sample_count,
  auxiliary_ttft_le_5s_count = EXCLUDED.auxiliary_ttft_le_5s_count,
  auxiliary_ttft_le_10s_count = EXCLUDED.auxiliary_ttft_le_10s_count,
  auxiliary_ttft_gt_10s_count = EXCLUDED.auxiliary_ttft_gt_10s_count,
  auxiliary_ttft_gt_20s_count = EXCLUDED.auxiliary_ttft_gt_20s_count,
  auxiliary_ttft_gt_40s_count = EXCLUDED.auxiliary_ttft_gt_40s_count,
  auxiliary_weight = EXCLUDED.auxiliary_weight,
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
  base_quality_score,
  effective_quality_score,
  transient_penalty,
  recovery_credit,
  slow_streak,
  error_streak,
  recovery_success_streak,
  recovery_fast_streak,
  quality_score,
  auxiliary_total_requests,
  auxiliary_success_requests,
  auxiliary_failure_requests,
  auxiliary_ttft_sample_count,
  auxiliary_ttft_le_5s_count,
  auxiliary_ttft_le_10s_count,
  auxiliary_ttft_gt_10s_count,
  auxiliary_ttft_gt_20s_count,
  auxiliary_ttft_gt_40s_count,
  auxiliary_weight,
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
			&snapshot.BaseQualityScore,
			&snapshot.EffectiveQualityScore,
			&snapshot.TransientPenalty,
			&snapshot.RecoveryCredit,
			&snapshot.SlowStreak,
			&snapshot.ErrorStreak,
			&snapshot.RecoverySuccessStreak,
			&snapshot.RecoveryFastStreak,
			&snapshot.QualityScore,
			&snapshot.AuxiliaryTotalRequests,
			&snapshot.AuxiliarySuccessRequests,
			&snapshot.AuxiliaryFailureRequests,
			&snapshot.AuxiliaryTTFTSampleCount,
			&snapshot.AuxiliaryTTFTLE5sCount,
			&snapshot.AuxiliaryTTFTLE10sCount,
			&snapshot.AuxiliaryTTFTGT10sCount,
			&snapshot.AuxiliaryTTFTGT20sCount,
			&snapshot.AuxiliaryTTFTGT40sCount,
			&snapshot.AuxiliaryWeight,
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
