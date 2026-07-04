package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeAccountUpstreamMonitorRepo struct {
	snapshot     *AccountUpstreamMonitorSnapshot
	rates        []AccountUpstreamMonitorRate
	saveSuccess  int
	saveFailure  int
	dueAccountID []int64
}

func (r *fakeAccountUpstreamMonitorRepo) Get(context.Context, int64) (*AccountUpstreamMonitorSnapshot, error) {
	if r.snapshot == nil {
		return nil, ErrAccountUpstreamMonitorNotFound
	}
	r.snapshot.Rates = r.rates
	return r.snapshot, nil
}

func (r *fakeAccountUpstreamMonitorRepo) GetBatch(context.Context, []int64) (map[int64]*AccountUpstreamMonitorSnapshot, error) {
	if r.snapshot == nil {
		return map[int64]*AccountUpstreamMonitorSnapshot{}, nil
	}
	return map[int64]*AccountUpstreamMonitorSnapshot{r.snapshot.AccountID: r.snapshot}, nil
}

func (r *fakeAccountUpstreamMonitorRepo) SaveSuccess(_ context.Context, snapshot *AccountUpstreamMonitorSnapshot, rates []AccountUpstreamMonitorRate) error {
	r.saveSuccess++
	r.snapshot = snapshot
	r.rates = rates
	return nil
}

func (r *fakeAccountUpstreamMonitorRepo) SaveFailure(_ context.Context, snapshot *AccountUpstreamMonitorSnapshot) error {
	r.saveFailure++
	r.snapshot = snapshot
	r.rates = nil
	return nil
}

func (r *fakeAccountUpstreamMonitorRepo) ListDueAccountIDs(context.Context, time.Time, int) ([]int64, error) {
	return r.dueAccountID, nil
}

func TestNormalizeUpstreamMonitorSiteURL(t *testing.T) {
	require.Equal(t, "https://example.com", normalizeUpstreamMonitorSiteURL(" https://example.com/api/v1/ "))
	require.Equal(t, "https://example.com", normalizeUpstreamMonitorSiteURL("https://example.com/v1"))
	require.Equal(t, "https://example.com/base", normalizeUpstreamMonitorSiteURL("https://example.com/base/v1"))
	require.Equal(t, "https://example.com/other", normalizeUpstreamMonitorSiteURL("https://example.com/other"))
}

func TestSafeUpstreamMonitorErrorDoesNotExposeWrappedError(t *testing.T) {
	err := newUpstreamMonitorFailureError(
		"upstream monitor request failed",
		errors.New("Authorization: Bearer sk-test-secret"),
	)

	require.Equal(t, "upstream monitor request failed", safeUpstreamMonitorError(err))
}

func TestRefreshUnsupportedAccountSavesFailureSnapshotOnly(t *testing.T) {
	repo := &fakeAccountUpstreamMonitorRepo{}
	svc := NewAccountUpstreamMonitorService(repo, nil, nil, nil, nil)

	snapshot, err := svc.Refresh(context.Background(), &Account{
		ID:          42,
		Name:        "missing-monitor-credentials",
		Credentials: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, int64(42), snapshot.AccountID)
	require.Equal(t, AccountUpstreamMonitorProviderUnsupported, snapshot.Provider)
	require.Equal(t, AccountUpstreamMonitorStatusUnsupported, snapshot.Status)
	require.NotNil(t, snapshot.LastError)
	require.NotContains(t, *snapshot.LastError, "sk-")
	require.Equal(t, 0, repo.saveSuccess)
	require.Equal(t, 1, repo.saveFailure)
}
