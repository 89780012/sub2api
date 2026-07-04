package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type fakeAccountUpstreamMonitorRepo struct {
	snapshot     *AccountUpstreamMonitorSnapshot
	rates        []AccountUpstreamMonitorRate
	saveSuccess  int
	saveFailure  int
	dueAccountID []int64
}

type recordingMonitorHTTPUpstream struct {
	requests  []*http.Request
	responses map[string]string
}

func (u *recordingMonitorHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.requests = append(u.requests, req.Clone(req.Context()))
	body, ok := u.responses[req.URL.Path]
	status := http.StatusOK
	if !ok {
		body = `{}`
		status = http.StatusNotFound
	}
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func (u *recordingMonitorHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
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
	require.Equal(t, AccountUpstreamMonitorProviderUnknown, snapshot.Provider)
	require.Equal(t, AccountUpstreamMonitorStatusUnsupported, snapshot.Status)
	require.NotNil(t, snapshot.LastError)
	require.Equal(t, accountUpstreamMonitorNotConfigured, *snapshot.LastError)
	require.NotContains(t, *snapshot.LastError, "sk-")
	require.Equal(t, 0, repo.saveSuccess)
	require.Equal(t, 1, repo.saveFailure)
}

func TestRefreshWithoutMonitorConfigDoesNotUseInferenceCredentials(t *testing.T) {
	repo := &fakeAccountUpstreamMonitorRepo{}
	upstream := &recordingMonitorHTTPUpstream{}
	svc := NewAccountUpstreamMonitorService(repo, nil, upstream, nil, nil)

	snapshot, err := svc.Refresh(context.Background(), &Account{
		ID:   43,
		Name: "openai-compatible",
		Type: AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://api.openai.com/v1",
			"api_key":  "sk-inference",
		},
	})

	require.NoError(t, err)
	require.Equal(t, AccountUpstreamMonitorProviderUnknown, snapshot.Provider)
	require.Equal(t, AccountUpstreamMonitorStatusUnsupported, snapshot.Status)
	require.NotNil(t, snapshot.LastError)
	require.Equal(t, accountUpstreamMonitorNotConfigured, *snapshot.LastError)
	require.Empty(t, upstream.requests, "monitor refresh must not call the inference base_url/api_key")
	require.Equal(t, 0, repo.saveSuccess)
	require.Equal(t, 1, repo.saveFailure)
}

func TestRefreshDisabledMonitorConfigReturnsDisabledState(t *testing.T) {
	repo := &fakeAccountUpstreamMonitorRepo{}
	svc := NewAccountUpstreamMonitorService(repo, nil, &recordingMonitorHTTPUpstream{}, nil, nil)

	snapshot, err := svc.Refresh(context.Background(), &Account{
		ID:   44,
		Type: AccountTypeAPIKey,
		Credentials: map[string]any{
			"upstream_monitor": map[string]any{
				"enabled":  false,
				"provider": "newapi",
				"site_url": "https://monitor.example.com",
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, AccountUpstreamMonitorProviderUnknown, snapshot.Provider)
	require.Equal(t, AccountUpstreamMonitorStatusUnsupported, snapshot.Status)
	require.NotNil(t, snapshot.LastError)
	require.Equal(t, accountUpstreamMonitorDisabled, *snapshot.LastError)
}

func TestRefreshNewAPIUsesMonitorCookieAndUserID(t *testing.T) {
	repo := &fakeAccountUpstreamMonitorRepo{}
	upstream := &recordingMonitorHTTPUpstream{responses: map[string]string{
		"/base/api/status":           `{"quota_per_unit":100}`,
		"/base/api/user/self":        `{"quota":500,"used_quota":125}`,
		"/base/api/user/self/groups": `{"default":{"ratio":1,"desc":"Default group"},"vip":{"ratio":2.5,"desc":"VIP group"}}`,
	}}
	svc := NewAccountUpstreamMonitorService(repo, nil, upstream, monitorTestConfig(), nil)

	snapshot, err := svc.Refresh(context.Background(), &Account{
		ID:   45,
		Type: AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://api.openai.com/v1",
			"api_key":  "sk-inference",
			"upstream_monitor": map[string]any{
				"enabled":         true,
				"provider":        AccountUpstreamMonitorProviderNewAPI,
				"site_url":        "http://monitor.example.com/base/v1",
				"credential_mode": "token",
				"newapi_cookie":   "session=monitor-secret",
				"newapi_user_id":  "123",
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, AccountUpstreamMonitorProviderNewAPI, snapshot.Provider)
	require.Equal(t, AccountUpstreamMonitorStatusSuccess, snapshot.Status)
	require.NotNil(t, snapshot.SiteURL)
	require.Equal(t, "http://monitor.example.com/base", *snapshot.SiteURL)
	require.Len(t, upstream.requests, 3)
	for _, req := range upstream.requests {
		require.Equal(t, "monitor.example.com", req.URL.Host)
		require.Empty(t, req.Header.Get("Authorization"))
		if req.URL.Path == "/base/api/user/self" || req.URL.Path == "/base/api/user/self/groups" {
			require.Equal(t, "session=monitor-secret", req.Header.Get("Cookie"))
			require.Equal(t, "123", req.Header.Get("New-Api-User"))
		}
	}
}

func TestRefreshSub2APIUsesMonitorBearerToken(t *testing.T) {
	repo := &fakeAccountUpstreamMonitorRepo{}
	upstream := &recordingMonitorHTTPUpstream{responses: map[string]string{
		"/api/v1/auth/me":          `{"balance":12.5}`,
		"/api/v1/groups/available": `[{"id":7,"name":"team","description":"Team group","rate_multiplier":1.8}]`,
	}}
	svc := NewAccountUpstreamMonitorService(repo, nil, upstream, monitorTestConfig(), nil)

	snapshot, err := svc.Refresh(context.Background(), &Account{
		ID:   46,
		Type: AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://api.openai.com/v1",
			"api_key":  "sk-inference",
			"upstream_monitor": map[string]any{
				"enabled":              true,
				"provider":             AccountUpstreamMonitorProviderSub2API,
				"site_url":             "http://sub2.example.com",
				"sub2api_access_token": "monitor-token",
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, AccountUpstreamMonitorProviderSub2API, snapshot.Provider)
	require.Equal(t, AccountUpstreamMonitorStatusSuccess, snapshot.Status)
	require.Len(t, snapshot.Rates, 1)
	require.NotEmpty(t, upstream.requests)
	for _, req := range upstream.requests {
		require.Equal(t, "sub2.example.com", req.URL.Host)
		require.Equal(t, "Bearer monitor-token", req.Header.Get("Authorization"))
		require.Empty(t, req.Header.Get("Cookie"))
		require.Empty(t, req.Header.Get("New-Api-User"))
	}
}

func monitorTestConfig() *config.Config {
	return &config.Config{
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				AllowInsecureHTTP: true,
			},
		},
	}
}
