package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbrate "github.com/Wei-Shaw/sub2api/ent/accountupstreammonitorrate"
	dbsnapshot "github.com/Wei-Shaw/sub2api/ent/accountupstreammonitorsnapshot"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type accountUpstreamMonitorRepository struct {
	client *dbent.Client
	db     *sql.DB
}

func NewAccountUpstreamMonitorRepository(client *dbent.Client, db *sql.DB) service.AccountUpstreamMonitorRepository {
	return &accountUpstreamMonitorRepository{client: client, db: db}
}

func (r *accountUpstreamMonitorRepository) Get(ctx context.Context, accountID int64) (*service.AccountUpstreamMonitorSnapshot, error) {
	if accountID <= 0 {
		return nil, service.ErrAccountUpstreamMonitorNotFound
	}
	client := clientFromContext(ctx, r.client)
	row, err := client.AccountUpstreamMonitorSnapshot.Query().
		Where(dbsnapshot.AccountIDEQ(accountID)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrAccountUpstreamMonitorNotFound, nil)
	}
	out := accountUpstreamMonitorSnapshotToService(row)
	if out == nil {
		return nil, service.ErrAccountUpstreamMonitorNotFound
	}
	rates, err := r.listRates(ctx, []int64{accountID})
	if err != nil {
		return nil, err
	}
	out.Rates = rates[accountID]
	if out.Rates == nil {
		out.Rates = []service.AccountUpstreamMonitorRate{}
	}
	return out, nil
}

func (r *accountUpstreamMonitorRepository) GetBatch(ctx context.Context, accountIDs []int64) (map[int64]*service.AccountUpstreamMonitorSnapshot, error) {
	ids := normalizeAccountMonitorIDs(accountIDs)
	out := make(map[int64]*service.AccountUpstreamMonitorSnapshot, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	client := clientFromContext(ctx, r.client)
	rows, err := client.AccountUpstreamMonitorSnapshot.Query().
		Where(dbsnapshot.AccountIDIn(ids...)).
		All(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrAccountUpstreamMonitorNotFound, nil)
	}
	for _, row := range rows {
		snapshot := accountUpstreamMonitorSnapshotToService(row)
		if snapshot != nil {
			out[snapshot.AccountID] = snapshot
		}
	}
	rates, err := r.listRates(ctx, ids)
	if err != nil {
		return nil, err
	}
	for accountID, snapshot := range out {
		snapshot.Rates = rates[accountID]
		if snapshot.Rates == nil {
			snapshot.Rates = []service.AccountUpstreamMonitorRate{}
		}
	}
	return out, nil
}

func (r *accountUpstreamMonitorRepository) SaveSuccess(ctx context.Context, snapshot *service.AccountUpstreamMonitorSnapshot, rates []service.AccountUpstreamMonitorRate) error {
	if snapshot == nil || snapshot.AccountID <= 0 {
		return service.ErrAccountUpstreamMonitorNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := upsertAccountUpstreamMonitorSnapshot(ctx, tx, snapshot); err != nil {
		return err
	}
	seenKeys := make([]string, 0, len(rates))
	for _, rate := range rates {
		rate.RateKey = normalizeRateKey(rate.RateKey)
		if rate.RateKey == "" {
			continue
		}
		if rate.DisplayName == "" {
			rate.DisplayName = rate.RateKey
		}
		if rate.FirstSeenAt == nil {
			rate.FirstSeenAt = snapshot.LastSuccessAt
		}
		if rate.LastSeenAt == nil {
			rate.LastSeenAt = snapshot.LastSuccessAt
		}
		if rate.FirstSeenAt == nil {
			now := time.Now()
			rate.FirstSeenAt = &now
		}
		if rate.LastSeenAt == nil {
			rate.LastSeenAt = rate.FirstSeenAt
		}
		if err := upsertAccountUpstreamMonitorRate(ctx, tx, snapshot.AccountID, snapshot.Provider, rate); err != nil {
			return err
		}
		seenKeys = append(seenKeys, rate.RateKey)
	}
	if len(seenKeys) == 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM account_upstream_monitor_rates WHERE account_id = $1`, snapshot.AccountID); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `DELETE FROM account_upstream_monitor_rates WHERE account_id = $1 AND NOT (rate_key = ANY($2))`, snapshot.AccountID, pq.Array(seenKeys)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *accountUpstreamMonitorRepository) SaveFailure(ctx context.Context, snapshot *service.AccountUpstreamMonitorSnapshot) error {
	if snapshot == nil || snapshot.AccountID <= 0 {
		return service.ErrAccountUpstreamMonitorNotFound
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO account_upstream_monitor_snapshots (
			account_id, provider, status, site_url, last_checked_at, last_error, raw_meta, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7::jsonb, '{}'::jsonb), NOW(), NOW())
		ON CONFLICT (account_id) DO UPDATE SET
			provider = EXCLUDED.provider,
			status = EXCLUDED.status,
			site_url = EXCLUDED.site_url,
			balance = CASE WHEN EXCLUDED.status = 'unsupported' THEN NULL ELSE account_upstream_monitor_snapshots.balance END,
			balance_unit = CASE WHEN EXCLUDED.status = 'unsupported' THEN NULL ELSE account_upstream_monitor_snapshots.balance_unit END,
			quota = CASE WHEN EXCLUDED.status = 'unsupported' THEN NULL ELSE account_upstream_monitor_snapshots.quota END,
			quota_used = CASE WHEN EXCLUDED.status = 'unsupported' THEN NULL ELSE account_upstream_monitor_snapshots.quota_used END,
			today_cost = CASE WHEN EXCLUDED.status = 'unsupported' THEN NULL ELSE account_upstream_monitor_snapshots.today_cost END,
			total_cost = CASE WHEN EXCLUDED.status = 'unsupported' THEN NULL ELSE account_upstream_monitor_snapshots.total_cost END,
			last_checked_at = EXCLUDED.last_checked_at,
			last_error = EXCLUDED.last_error,
			raw_meta = EXCLUDED.raw_meta,
			updated_at = NOW()
	`, snapshot.AccountID, snapshot.Provider, snapshot.Status, snapshot.SiteURL, snapshot.LastCheckedAt, snapshot.LastError, jsonbOrDefault(snapshot.RawMeta))
	if err != nil {
		return translatePersistenceError(err, service.ErrAccountUpstreamMonitorNotFound, nil)
	}
	if snapshot.Status == service.AccountUpstreamMonitorStatusUnsupported {
		if _, err := r.db.ExecContext(ctx, `DELETE FROM account_upstream_monitor_rates WHERE account_id = $1`, snapshot.AccountID); err != nil {
			return err
		}
	}
	return nil
}

func (r *accountUpstreamMonitorRepository) ListDueAccountIDs(ctx context.Context, before time.Time, limit int) ([]int64, error) {
	if limit <= 0 {
		return []int64{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id
		FROM accounts a
		LEFT JOIN account_upstream_monitor_snapshots s ON s.account_id = a.id
		WHERE a.deleted_at IS NULL
			AND a.type NOT IN ('bedrock', 'service_account')
			AND COALESCE(a.credentials->>'base_url', '') <> ''
			AND (
				COALESCE(a.credentials->>'api_key', '') <> ''
				OR COALESCE(a.credentials->>'token', '') <> ''
				OR COALESCE(a.credentials->>'access_token', '') <> ''
			)
			AND (s.account_id IS NULL OR s.last_checked_at IS NULL OR s.last_checked_at < $1)
		ORDER BY s.last_checked_at NULLS FIRST, a.id ASC
		LIMIT $2
	`, before, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *accountUpstreamMonitorRepository) listRates(ctx context.Context, accountIDs []int64) (map[int64][]service.AccountUpstreamMonitorRate, error) {
	ids := normalizeAccountMonitorIDs(accountIDs)
	out := make(map[int64][]service.AccountUpstreamMonitorRate, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	client := clientFromContext(ctx, r.client)
	rows, err := client.AccountUpstreamMonitorRate.Query().
		Where(dbrate.AccountIDIn(ids...)).
		Order(dbrate.ByAccountID(), dbrate.ByRateKey()).
		All(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrAccountUpstreamMonitorNotFound, nil)
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		out[row.AccountID] = append(out[row.AccountID], accountUpstreamMonitorRateToService(row))
	}
	return out, nil
}

func upsertAccountUpstreamMonitorSnapshot(ctx context.Context, tx *sql.Tx, snapshot *service.AccountUpstreamMonitorSnapshot) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO account_upstream_monitor_snapshots (
			account_id, provider, status, site_url, balance, balance_unit, quota, quota_used,
			today_cost, total_cost, last_checked_at, last_success_at, last_error, raw_meta, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, COALESCE($14::jsonb, '{}'::jsonb), NOW(), NOW())
		ON CONFLICT (account_id) DO UPDATE SET
			provider = EXCLUDED.provider,
			status = EXCLUDED.status,
			site_url = EXCLUDED.site_url,
			balance = EXCLUDED.balance,
			balance_unit = EXCLUDED.balance_unit,
			quota = EXCLUDED.quota,
			quota_used = EXCLUDED.quota_used,
			today_cost = EXCLUDED.today_cost,
			total_cost = EXCLUDED.total_cost,
			last_checked_at = EXCLUDED.last_checked_at,
			last_success_at = EXCLUDED.last_success_at,
			last_error = EXCLUDED.last_error,
			raw_meta = EXCLUDED.raw_meta,
			updated_at = NOW()
	`, snapshot.AccountID, snapshot.Provider, snapshot.Status, snapshot.SiteURL, snapshot.Balance, snapshot.BalanceUnit, snapshot.Quota, snapshot.QuotaUsed, snapshot.TodayCost, snapshot.TotalCost, snapshot.LastCheckedAt, snapshot.LastSuccessAt, snapshot.LastError, jsonbOrDefault(snapshot.RawMeta))
	return err
}

func upsertAccountUpstreamMonitorRate(ctx context.Context, tx *sql.Tx, accountID int64, provider string, rate service.AccountUpstreamMonitorRate) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO account_upstream_monitor_rates (
			account_id, provider, rate_key, display_name, description, ratio, completion_ratio, first_seen_at, last_seen_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (account_id, rate_key) DO UPDATE SET
			provider = EXCLUDED.provider,
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			ratio = EXCLUDED.ratio,
			completion_ratio = EXCLUDED.completion_ratio,
			first_seen_at = LEAST(account_upstream_monitor_rates.first_seen_at, EXCLUDED.first_seen_at),
			last_seen_at = EXCLUDED.last_seen_at,
			updated_at = NOW()
	`, accountID, provider, rate.RateKey, rate.DisplayName, rate.Description, rate.Ratio, rate.CompletionRatio, rate.FirstSeenAt, rate.LastSeenAt)
	return err
}

func accountUpstreamMonitorSnapshotToService(row *dbent.AccountUpstreamMonitorSnapshot) *service.AccountUpstreamMonitorSnapshot {
	if row == nil {
		return nil
	}
	return &service.AccountUpstreamMonitorSnapshot{
		AccountID:     row.AccountID,
		Provider:      row.Provider.String(),
		Status:        row.Status.String(),
		SiteURL:       row.SiteURL,
		Balance:       row.Balance,
		BalanceUnit:   row.BalanceUnit,
		Quota:         row.Quota,
		QuotaUsed:     row.QuotaUsed,
		TodayCost:     row.TodayCost,
		TotalCost:     row.TotalCost,
		LastCheckedAt: row.LastCheckedAt,
		LastSuccessAt: row.LastSuccessAt,
		LastError:     row.LastError,
		RawMeta:       row.RawMeta,
		Rates:         []service.AccountUpstreamMonitorRate{},
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func accountUpstreamMonitorRateToService(row *dbent.AccountUpstreamMonitorRate) service.AccountUpstreamMonitorRate {
	firstSeenAt := row.FirstSeenAt
	lastSeenAt := row.LastSeenAt
	return service.AccountUpstreamMonitorRate{
		RateKey:         row.RateKey,
		DisplayName:     row.DisplayName,
		Description:     row.Description,
		Ratio:           row.Ratio,
		CompletionRatio: row.CompletionRatio,
		FirstSeenAt:     &firstSeenAt,
		LastSeenAt:      &lastSeenAt,
	}
}

func normalizeAccountMonitorIDs(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizeRateKey(v string) string {
	if len(v) > 256 {
		return v[:256]
	}
	return v
}

func jsonbOrDefault(v map[string]any) []byte {
	if len(v) == 0 {
		return []byte(`{}`)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}
