package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type balanceSnapshotRepository struct {
	db *sql.DB
}

func NewBalanceSnapshotRepository(db *sql.DB) service.BalanceSnapshotRepository {
	return &balanceSnapshotRepository{db: db}
}

func (r *balanceSnapshotRepository) ListSites(ctx context.Context) ([]service.BalanceSite, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, platform, name, base_url, username, email, password_encrypted,
		       enabled, refresh_interval_minutes, last_refresh_at, last_refresh_status,
		       COALESCE(last_refresh_error, ''), created_at, updated_at
		FROM balance_sites
		ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBalanceSites(rows)
}

func (r *balanceSnapshotRepository) GetSite(ctx context.Context, id int64) (*service.BalanceSite, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, platform, name, base_url, username, email, password_encrypted,
		       enabled, refresh_interval_minutes, last_refresh_at, last_refresh_status,
		       COALESCE(last_refresh_error, ''), created_at, updated_at
		FROM balance_sites
		WHERE id = $1`, id)
	site, err := scanBalanceSite(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrBalanceSiteNotFound
		}
		return nil, err
	}
	return site, nil
}

func (r *balanceSnapshotRepository) CreateSite(ctx context.Context, site *service.BalanceSite) error {
	if site == nil {
		return service.ErrBalanceSiteInvalid
	}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO balance_sites (
			platform, name, base_url, username, email, password_encrypted,
			enabled, refresh_interval_minutes
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at, updated_at`,
		site.Platform, site.Name, site.BaseURL, site.Username, site.Email, site.PasswordEncrypted,
		site.Enabled, site.RefreshIntervalMinutes,
	).Scan(&site.ID, &site.CreatedAt, &site.UpdatedAt)
	if err != nil {
		return err
	}
	site.PasswordConfigured = site.PasswordEncrypted != ""
	return nil
}

func (r *balanceSnapshotRepository) UpdateSite(ctx context.Context, site *service.BalanceSite) error {
	if site == nil {
		return service.ErrBalanceSiteInvalid
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE balance_sites
		SET platform = $2,
		    name = $3,
		    base_url = $4,
		    username = $5,
		    email = $6,
		    password_encrypted = $7,
		    enabled = $8,
		    refresh_interval_minutes = $9,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING created_at, updated_at`,
		site.ID, site.Platform, site.Name, site.BaseURL, site.Username, site.Email,
		site.PasswordEncrypted, site.Enabled, site.RefreshIntervalMinutes,
	)
	if err := row.Scan(&site.CreatedAt, &site.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrBalanceSiteNotFound
		}
		return err
	}
	return nil
}

func (r *balanceSnapshotRepository) DeleteSite(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM balance_sites WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrBalanceSiteNotFound
	}
	return nil
}

func (r *balanceSnapshotRepository) ListDueSites(ctx context.Context, now time.Time) ([]service.BalanceSite, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, platform, name, base_url, username, email, password_encrypted,
		       enabled, refresh_interval_minutes, last_refresh_at, last_refresh_status,
		       COALESCE(last_refresh_error, ''), created_at, updated_at
		FROM balance_sites
		WHERE enabled = TRUE
		  AND (
		    last_refresh_at IS NULL
		    OR last_refresh_at + (refresh_interval_minutes || ' minutes')::interval <= $1
		  )
		ORDER BY COALESCE(last_refresh_at, '1970-01-01'::timestamptz) ASC`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBalanceSites(rows)
}

func (r *balanceSnapshotRepository) UpdateSiteRefreshStatus(ctx context.Context, siteID int64, status, message string, refreshedAt *time.Time) error {
	if refreshedAt != nil {
		_, err := r.db.ExecContext(ctx, `
			UPDATE balance_sites
			SET last_refresh_status = $2,
			    last_refresh_error = NULLIF($3, ''),
			    last_refresh_at = $4,
			    updated_at = NOW()
			WHERE id = $1`, siteID, status, message, *refreshedAt)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE balance_sites
		SET last_refresh_status = $2,
		    last_refresh_error = NULLIF($3, ''),
		    updated_at = NOW()
		WHERE id = $1`, siteID, status, message)
	return err
}

func (r *balanceSnapshotRepository) ListSnapshots(ctx context.Context, siteID int64) ([]service.BalanceSnapshot, error) {
	query := `
		SELECT s.id, s.site_id, bs.name, bs.platform, s.external_key_id, s.masked_key, s.key_last4,
		       s.account_id, s.match_status, s.key_name, s.status, s.group_id, s.group_name,
		       s.balance, s.account_balance, s.subscription_balance, s.rate_multiplier, s.fetched_at, s.created_at, s.updated_at
		FROM balance_key_snapshots s
		JOIN balance_sites bs ON bs.id = s.site_id`
	args := []any{}
	if siteID > 0 {
		query += ` WHERE s.site_id = $1`
		args = append(args, siteID)
	}
	query += ` ORDER BY s.site_id ASC, s.key_name ASC, s.id ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBalanceSnapshots(rows)
}

func (r *balanceSnapshotRepository) ListSnapshotsByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]*service.BalanceSnapshot, error) {
	out := make(map[int64]*service.BalanceSnapshot)
	if len(accountIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT ON (s.account_id)
		       s.id, s.site_id, bs.name, bs.platform, s.external_key_id, s.masked_key, s.key_last4,
		       s.account_id, s.match_status, s.key_name, s.status, s.group_id, s.group_name,
		       s.balance, s.account_balance, s.subscription_balance, s.rate_multiplier, s.fetched_at, s.created_at, s.updated_at
		FROM balance_key_snapshots s
		JOIN balance_sites bs ON bs.id = s.site_id
		WHERE s.account_id = ANY($1)
		  AND s.match_status IN ('matched', 'manual')
		ORDER BY s.account_id, s.fetched_at DESC, s.id DESC`, pq.Array(accountIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanBalanceSnapshots(rows)
	if err != nil {
		return nil, err
	}
	for i := range items {
		item := items[i]
		if item.AccountID == nil {
			continue
		}
		out[*item.AccountID] = &item
	}
	return out, nil
}

func (r *balanceSnapshotRepository) ReplaceSiteSnapshots(ctx context.Context, siteID int64, snapshots []service.BalanceSnapshot) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM balance_key_snapshots WHERE site_id = $1`, siteID); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO balance_key_snapshots (
			site_id, external_key_id, masked_key, key_last4, account_id, match_status,
			key_name, status, group_id, group_name, balance, account_balance, subscription_balance,
			rate_multiplier, fetched_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i := range snapshots {
		s := snapshots[i]
		balanceJSON, err := json.Marshal(nullMap(s.Balance))
		if err != nil {
			return err
		}
		accountBalanceJSON, err := json.Marshal(nullMap(s.AccountBalance))
		if err != nil {
			return err
		}
		subJSON, err := json.Marshal(nullMap(s.SubscriptionBalance))
		if err != nil {
			return err
		}
		var accountID any
		if s.AccountID != nil {
			accountID = *s.AccountID
		}
		if _, err := stmt.ExecContext(ctx,
			siteID, s.ExternalKeyID, s.MaskedKey, s.KeyLast4, accountID, s.MatchStatus,
			s.KeyName, s.Status, s.GroupID, s.GroupName, balanceJSON, accountBalanceJSON,
			subJSON, s.RateMultiplier, s.FetchedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *balanceSnapshotRepository) UpsertBinding(ctx context.Context, binding service.AccountBalanceBinding) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO account_balance_bindings (account_id, site_id, external_key_id, key_last4, match_mode)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (account_id) DO UPDATE SET
			site_id = EXCLUDED.site_id,
			external_key_id = EXCLUDED.external_key_id,
			key_last4 = EXCLUDED.key_last4,
			match_mode = EXCLUDED.match_mode,
			updated_at = NOW()`,
		binding.AccountID, binding.SiteID, binding.ExternalKeyID, binding.KeyLast4, binding.MatchMode)
	return err
}

func (r *balanceSnapshotRepository) DeleteBinding(ctx context.Context, accountID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM account_balance_bindings WHERE account_id = $1`, accountID)
	return err
}

func (r *balanceSnapshotRepository) ListBindings(ctx context.Context) ([]service.AccountBalanceBinding, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT account_id, site_id, external_key_id, key_last4, match_mode, created_at, updated_at
		FROM account_balance_bindings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.AccountBalanceBinding{}
	for rows.Next() {
		var item service.AccountBalanceBinding
		if err := rows.Scan(&item.AccountID, &item.SiteID, &item.ExternalKeyID, &item.KeyLast4, &item.MatchMode, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

type balanceSiteScanner interface {
	Scan(dest ...any) error
}

func scanBalanceSite(scanner balanceSiteScanner) (*service.BalanceSite, error) {
	var site service.BalanceSite
	if err := scanner.Scan(
		&site.ID, &site.Platform, &site.Name, &site.BaseURL, &site.Username, &site.Email,
		&site.PasswordEncrypted, &site.Enabled, &site.RefreshIntervalMinutes,
		&site.LastRefreshAt, &site.LastRefreshStatus, &site.LastRefreshError,
		&site.CreatedAt, &site.UpdatedAt,
	); err != nil {
		return nil, err
	}
	site.PasswordConfigured = site.PasswordEncrypted != ""
	return &site, nil
}

func scanBalanceSites(rows *sql.Rows) ([]service.BalanceSite, error) {
	out := []service.BalanceSite{}
	for rows.Next() {
		site, err := scanBalanceSite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *site)
	}
	return out, rows.Err()
}

func scanBalanceSnapshots(rows *sql.Rows) ([]service.BalanceSnapshot, error) {
	out := []service.BalanceSnapshot{}
	for rows.Next() {
		var item service.BalanceSnapshot
		var balanceRaw, accountBalanceRaw, subRaw []byte
		if err := rows.Scan(
			&item.ID, &item.SiteID, &item.SiteName, &item.SitePlatform, &item.ExternalKeyID,
			&item.MaskedKey, &item.KeyLast4, &item.AccountID, &item.MatchStatus,
			&item.KeyName, &item.Status, &item.GroupID, &item.GroupName,
			&balanceRaw, &accountBalanceRaw, &subRaw, &item.RateMultiplier, &item.FetchedAt,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Balance = decodeJSONMap(balanceRaw)
		item.AccountBalance = decodeJSONMap(accountBalanceRaw)
		item.SubscriptionBalance = decodeJSONMap(subRaw)
		out = append(out, item)
	}
	return out, rows.Err()
}

func nullMap(value map[string]any) any {
	if len(value) == 0 {
		return map[string]any{}
	}
	return value
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
