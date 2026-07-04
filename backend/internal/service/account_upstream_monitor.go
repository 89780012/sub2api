package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/robfig/cron/v3"
)

const (
	AccountUpstreamMonitorProviderSub2API           = "sub2api"
	AccountUpstreamMonitorProviderNewAPI            = "newapi"
	AccountUpstreamMonitorProviderUnsupported       = "unsupported"
	AccountUpstreamMonitorProviderUnknown           = "unknown"
	AccountUpstreamMonitorStatusUnknown             = "unknown"
	AccountUpstreamMonitorStatusUnsupported         = "unsupported"
	AccountUpstreamMonitorStatusSuccess             = "success"
	AccountUpstreamMonitorStatusFailed              = "failed"
	AccountUpstreamMonitorDefaultIntervalMins       = 60
	AccountUpstreamMonitorMinIntervalMins           = 15
	AccountUpstreamMonitorMaxIntervalMins           = 1440
	accountUpstreamMonitorDefaultMaxWorkers         = 5
	accountUpstreamMonitorDefaultHTTPTimeout        = 15 * time.Second
	accountUpstreamMonitorRunnerTimeout             = 5 * time.Minute
	accountUpstreamMonitorBodyLimit           int64 = 4 << 20
	accountUpstreamMonitorCredentialsKey            = "upstream_monitor"
	accountUpstreamMonitorCredentialModeToken       = "token"
	accountUpstreamMonitorNotConfigured             = "upstream monitor not configured"
	accountUpstreamMonitorDisabled                  = "upstream monitor disabled"
)

var (
	ErrAccountUpstreamMonitorNotFound = infraerrors.NotFound("ACCOUNT_UPSTREAM_MONITOR_NOT_FOUND", "account upstream monitor snapshot not found")
)

type AccountUpstreamMonitorRate struct {
	RateKey         string     `json:"rate_key"`
	DisplayName     string     `json:"display_name"`
	Description     *string    `json:"description,omitempty"`
	Ratio           float64    `json:"ratio"`
	CompletionRatio *float64   `json:"completion_ratio,omitempty"`
	FirstSeenAt     *time.Time `json:"first_seen_at,omitempty"`
	LastSeenAt      *time.Time `json:"last_seen_at,omitempty"`
}

type AccountUpstreamMonitorSnapshot struct {
	AccountID     int64                        `json:"account_id"`
	Provider      string                       `json:"provider"`
	Status        string                       `json:"status"`
	SiteURL       *string                      `json:"site_url,omitempty"`
	Balance       *float64                     `json:"balance,omitempty"`
	BalanceUnit   *string                      `json:"balance_unit,omitempty"`
	Quota         *float64                     `json:"quota,omitempty"`
	QuotaUsed     *float64                     `json:"quota_used,omitempty"`
	TodayCost     *float64                     `json:"today_cost,omitempty"`
	TotalCost     *float64                     `json:"total_cost,omitempty"`
	LastCheckedAt *time.Time                   `json:"last_checked_at,omitempty"`
	LastSuccessAt *time.Time                   `json:"last_success_at,omitempty"`
	LastError     *string                      `json:"last_error,omitempty"`
	RawMeta       map[string]any               `json:"raw_meta,omitempty"`
	Rates         []AccountUpstreamMonitorRate `json:"rates"`
	CreatedAt     time.Time                    `json:"created_at,omitempty"`
	UpdatedAt     time.Time                    `json:"updated_at,omitempty"`
}

type AccountUpstreamMonitorRefreshResult struct {
	Snapshot *AccountUpstreamMonitorSnapshot
	Rates    []AccountUpstreamMonitorRate
}

type accountUpstreamMonitorCredentials struct {
	Enabled            bool
	Provider           string
	SiteURL            string
	CredentialMode     string
	NewAPICookie       string
	NewAPIUserID       string
	Sub2APIAccessToken string
}

type upstreamMonitorRequestAuth struct {
	BearerToken  string
	Cookie       string
	NewAPIUserID string
}

type AccountUpstreamMonitorRepository interface {
	Get(ctx context.Context, accountID int64) (*AccountUpstreamMonitorSnapshot, error)
	GetBatch(ctx context.Context, accountIDs []int64) (map[int64]*AccountUpstreamMonitorSnapshot, error)
	SaveSuccess(ctx context.Context, snapshot *AccountUpstreamMonitorSnapshot, rates []AccountUpstreamMonitorRate) error
	SaveFailure(ctx context.Context, snapshot *AccountUpstreamMonitorSnapshot) error
	ListDueAccountIDs(ctx context.Context, before time.Time, limit int) ([]int64, error)
}

type AccountUpstreamMonitorRuntime struct {
	Enabled         bool
	IntervalMinutes int
}

type AccountUpstreamMonitorService struct {
	repo                AccountUpstreamMonitorRepository
	accountRepo         AccountRepository
	httpUpstream        HTTPUpstream
	cfg                 *config.Config
	tlsFPProfileService *TLSFingerprintProfileService
}

func NewAccountUpstreamMonitorService(
	repo AccountUpstreamMonitorRepository,
	accountRepo AccountRepository,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
	tlsFPProfileService *TLSFingerprintProfileService,
) *AccountUpstreamMonitorService {
	return &AccountUpstreamMonitorService{
		repo:                repo,
		accountRepo:         accountRepo,
		httpUpstream:        httpUpstream,
		cfg:                 cfg,
		tlsFPProfileService: tlsFPProfileService,
	}
}

func (s *AccountUpstreamMonitorService) Get(ctx context.Context, accountID int64) (*AccountUpstreamMonitorSnapshot, error) {
	return s.repo.Get(ctx, accountID)
}

func (s *AccountUpstreamMonitorService) GetBatch(ctx context.Context, accountIDs []int64) (map[int64]*AccountUpstreamMonitorSnapshot, error) {
	return s.repo.GetBatch(ctx, accountIDs)
}

func (s *AccountUpstreamMonitorService) RefreshAccount(ctx context.Context, accountID int64) (*AccountUpstreamMonitorSnapshot, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.Refresh(ctx, account)
}

func (s *AccountUpstreamMonitorService) Refresh(ctx context.Context, account *Account) (*AccountUpstreamMonitorSnapshot, error) {
	now := time.Now()
	snapshot, rates := s.collect(ctx, account, now)
	if snapshot.Status == AccountUpstreamMonitorStatusSuccess {
		if err := s.repo.SaveSuccess(ctx, snapshot, rates); err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.SaveFailure(ctx, snapshot); err != nil {
			return nil, err
		}
	}
	return s.repo.Get(ctx, snapshot.AccountID)
}

func (s *AccountUpstreamMonitorService) RefreshDue(ctx context.Context, interval time.Duration, maxWorkers int) {
	if maxWorkers <= 0 {
		maxWorkers = accountUpstreamMonitorDefaultMaxWorkers
	}
	before := time.Now().Add(-interval)
	accountIDs, err := s.repo.ListDueAccountIDs(ctx, before, maxWorkers*20)
	if err != nil {
		logger.LegacyPrintf("service.account_upstream_monitor", "[AccountUpstreamMonitor] list due accounts failed: %v", err)
		return
	}
	if len(accountIDs) == 0 {
		return
	}
	accounts, err := s.accountRepo.GetByIDs(ctx, accountIDs)
	if err != nil {
		logger.LegacyPrintf("service.account_upstream_monitor", "[AccountUpstreamMonitor] load due accounts failed: %v", err)
		return
	}

	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	for _, account := range accounts {
		if account == nil {
			continue
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(acc *Account) {
			defer wg.Done()
			defer func() { <-sem }()
			childCtx, cancel := context.WithTimeout(ctx, accountUpstreamMonitorDefaultHTTPTimeout)
			defer cancel()
			if _, refreshErr := s.Refresh(childCtx, acc); refreshErr != nil {
				logger.LegacyPrintf("service.account_upstream_monitor", "[AccountUpstreamMonitor] refresh account=%d failed: %v", acc.ID, refreshErr)
			}
		}(account)
	}
	wg.Wait()
}

func (s *AccountUpstreamMonitorService) collect(ctx context.Context, account *Account, now time.Time) (*AccountUpstreamMonitorSnapshot, []AccountUpstreamMonitorRate) {
	snapshot := &AccountUpstreamMonitorSnapshot{
		AccountID:     0,
		Provider:      AccountUpstreamMonitorProviderUnknown,
		Status:        AccountUpstreamMonitorStatusFailed,
		LastCheckedAt: &now,
		Rates:         []AccountUpstreamMonitorRate{},
		RawMeta:       map[string]any{},
	}
	if account == nil {
		msg := "account is required"
		snapshot.LastError = &msg
		return snapshot, nil
	}
	snapshot.AccountID = account.ID

	monitorCreds, unsupportedReason, err := s.resolveAccountMonitorCredentials(account)
	if err != nil {
		snapshot.Provider = AccountUpstreamMonitorProviderUnsupported
		snapshot.Status = AccountUpstreamMonitorStatusUnsupported
		snapshot.LastError = stringPtr(safeUpstreamMonitorError(err))
		return snapshot, nil
	}
	if unsupportedReason != "" {
		snapshot.Provider = AccountUpstreamMonitorProviderUnknown
		if unsupportedReason != accountUpstreamMonitorNotConfigured && unsupportedReason != accountUpstreamMonitorDisabled {
			snapshot.Provider = AccountUpstreamMonitorProviderUnsupported
		}
		snapshot.Status = AccountUpstreamMonitorStatusUnsupported
		if monitorCreds != nil && monitorCreds.SiteURL != "" {
			snapshot.SiteURL = stringPtr(monitorCreds.SiteURL)
		}
		snapshot.LastError = stringPtr(unsupportedReason)
		return snapshot, nil
	}
	siteURL := monitorCreds.SiteURL
	snapshot.SiteURL = &siteURL

	switch monitorCreds.Provider {
	case AccountUpstreamMonitorProviderSub2API:
		sub2Snapshot, sub2Rates, err := s.collectSub2API(ctx, account, monitorCreds, now)
		if err == nil {
			sub2Snapshot.AccountID = account.ID
			sub2Snapshot.SiteURL = &siteURL
			return sub2Snapshot, sub2Rates
		}
		snapshot.Provider = AccountUpstreamMonitorProviderSub2API
		snapshot.Status = AccountUpstreamMonitorStatusFailed
		if !isUpstreamMonitorFailure(err) {
			snapshot.Status = AccountUpstreamMonitorStatusUnsupported
		}
		snapshot.LastError = stringPtr(safeUpstreamMonitorError(err))
		return snapshot, nil
	case AccountUpstreamMonitorProviderNewAPI:
		newSnapshot, newRates, err := s.collectNewAPI(ctx, account, monitorCreds, now)
		if err == nil {
			newSnapshot.AccountID = account.ID
			newSnapshot.SiteURL = &siteURL
			return newSnapshot, newRates
		}
		snapshot.Provider = AccountUpstreamMonitorProviderNewAPI
		snapshot.Status = AccountUpstreamMonitorStatusFailed
		if !isUpstreamMonitorFailure(err) {
			snapshot.Status = AccountUpstreamMonitorStatusUnsupported
		}
		snapshot.LastError = stringPtr(safeUpstreamMonitorError(err))
		return snapshot, nil
	default:
		snapshot.Provider = AccountUpstreamMonitorProviderUnsupported
		snapshot.Status = AccountUpstreamMonitorStatusUnsupported
		snapshot.LastError = stringPtr("unsupported upstream monitor provider")
		return snapshot, nil
	}
}

func (s *AccountUpstreamMonitorService) resolveAccountMonitorCredentials(account *Account) (*accountUpstreamMonitorCredentials, string, error) {
	if account == nil {
		return nil, "", errors.New("account is required")
	}
	if account.Type == AccountTypeBedrock || account.Type == AccountTypeServiceAccount {
		return nil, "unsupported account type", nil
	}

	rawMonitor, ok := account.Credentials[accountUpstreamMonitorCredentialsKey]
	if account.Credentials == nil || !ok || rawMonitor == nil {
		return nil, accountUpstreamMonitorNotConfigured, nil
	}
	monitorMap, ok := accountMonitorMapStringAnyFromAny(rawMonitor)
	if !ok || len(monitorMap) == 0 {
		return nil, accountUpstreamMonitorNotConfigured, nil
	}

	creds := &accountUpstreamMonitorCredentials{
		Enabled:            accountMonitorBoolFromAny(monitorMap["enabled"]),
		Provider:           strings.ToLower(strings.TrimSpace(accountMonitorStringFromAny(monitorMap["provider"]))),
		SiteURL:            strings.TrimSpace(accountMonitorStringFromAny(monitorMap["site_url"])),
		CredentialMode:     strings.ToLower(strings.TrimSpace(accountMonitorStringFromAny(monitorMap["credential_mode"]))),
		NewAPICookie:       strings.TrimSpace(accountMonitorStringFromAny(monitorMap["newapi_cookie"])),
		NewAPIUserID:       strings.TrimSpace(accountMonitorStringFromAny(monitorMap["newapi_user_id"])),
		Sub2APIAccessToken: strings.TrimSpace(accountMonitorStringFromAny(monitorMap["sub2api_access_token"])),
	}
	if !creds.Enabled {
		return creds, accountUpstreamMonitorDisabled, nil
	}
	if creds.CredentialMode == "" {
		creds.CredentialMode = accountUpstreamMonitorCredentialModeToken
	}
	if creds.CredentialMode != accountUpstreamMonitorCredentialModeToken {
		return creds, "unsupported upstream monitor credential mode", nil
	}
	switch creds.Provider {
	case AccountUpstreamMonitorProviderNewAPI:
		if creds.NewAPICookie == "" || creds.NewAPIUserID == "" {
			return creds, "missing NewAPI monitor cookie or user ID", nil
		}
	case AccountUpstreamMonitorProviderSub2API:
		if creds.Sub2APIAccessToken == "" {
			return creds, "missing Sub2API monitor access token", nil
		}
	default:
		return creds, "unsupported upstream monitor provider", nil
	}
	if creds.SiteURL == "" {
		return creds, "missing upstream monitor site URL", nil
	}

	siteURL := normalizeUpstreamMonitorSiteURL(creds.SiteURL)
	if siteURL == "" {
		return creds, "", errors.New("invalid upstream monitor site URL")
	}
	validated, err := s.validateMonitorSiteURL(siteURL)
	if err != nil {
		return creds, "", fmt.Errorf("invalid upstream monitor site URL: %w", err)
	}
	creds.SiteURL = strings.TrimRight(validated, "/")
	return creds, "", nil
}

func normalizeUpstreamMonitorSiteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	for len(segments) > 0 {
		last := strings.ToLower(segments[len(segments)-1])
		if len(segments) >= 2 && strings.ToLower(segments[len(segments)-2]) == "api" && last == "v1" {
			segments = segments[:len(segments)-2]
			continue
		}
		if last == "v1" || last == "v1beta" || last == "antigravity" {
			segments = segments[:len(segments)-1]
			continue
		}
		break
	}
	if len(segments) == 0 {
		u.Path = ""
	} else {
		u.Path = "/" + strings.Join(segments, "/")
	}
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/")
}

func (s *AccountUpstreamMonitorService) validateMonitorSiteURL(raw string) (string, error) {
	if s.cfg == nil {
		return "", errors.New("config is not available")
	}
	if !s.cfg.Security.URLAllowlist.Enabled {
		return urlvalidator.ValidateURLFormat(raw, s.cfg.Security.URLAllowlist.AllowInsecureHTTP)
	}
	return urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     s.cfg.Security.URLAllowlist.UpstreamHosts,
		RequireAllowlist: true,
		AllowPrivate:     s.cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
}

func (s *AccountUpstreamMonitorService) collectSub2API(ctx context.Context, account *Account, creds *accountUpstreamMonitorCredentials, now time.Time) (*AccountUpstreamMonitorSnapshot, []AccountUpstreamMonitorRate, error) {
	auth := upstreamMonitorRequestAuth{BearerToken: creds.Sub2APIAccessToken}
	meBody, err := s.getUpstreamMonitorJSON(ctx, account, creds.SiteURL+"/api/v1/auth/me", auth)
	if err != nil {
		return nil, nil, err
	}
	var me struct {
		Balance float64 `json:"balance"`
	}
	if err := json.Unmarshal(meBody, &me); err != nil {
		return nil, nil, newUpstreamMonitorUnsupportedError("upstream monitor endpoint not supported", err)
	}

	unit := "USD"
	snapshot := &AccountUpstreamMonitorSnapshot{
		Provider:      AccountUpstreamMonitorProviderSub2API,
		Status:        AccountUpstreamMonitorStatusSuccess,
		Balance:       &me.Balance,
		BalanceUnit:   &unit,
		LastCheckedAt: &now,
		LastSuccessAt: &now,
		RawMeta:       map[string]any{},
	}

	if statsBody, statsErr := s.getUpstreamMonitorJSON(ctx, account, creds.SiteURL+"/api/v1/usage/dashboard/stats", auth); statsErr == nil {
		var stats struct {
			TodayActualCost float64 `json:"today_actual_cost"`
			TotalActualCost float64 `json:"total_actual_cost"`
		}
		if json.Unmarshal(statsBody, &stats) == nil {
			snapshot.TodayCost = &stats.TodayActualCost
			snapshot.TotalCost = &stats.TotalActualCost
		}
	}

	groupsBody, err := s.getUpstreamMonitorJSON(ctx, account, creds.SiteURL+"/api/v1/groups/available", auth)
	if err != nil {
		return nil, nil, err
	}
	var groups []struct {
		ID             int64   `json:"id"`
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		RateMultiplier float64 `json:"rate_multiplier"`
	}
	if err := json.Unmarshal(groupsBody, &groups); err != nil {
		return nil, nil, newUpstreamMonitorUnsupportedError("upstream monitor endpoint not supported", err)
	}
	overrides := map[string]float64{}
	if ratesBody, ratesErr := s.getUpstreamMonitorJSON(ctx, account, creds.SiteURL+"/api/v1/groups/rates", auth); ratesErr == nil {
		_ = json.Unmarshal(ratesBody, &overrides)
	}
	rates := make([]AccountUpstreamMonitorRate, 0, len(groups))
	for _, group := range groups {
		name := strings.TrimSpace(group.Name)
		if name == "" {
			name = strconv.FormatInt(group.ID, 10)
		}
		ratio := group.RateMultiplier
		if value, ok := overrides[strconv.FormatInt(group.ID, 10)]; ok {
			ratio = value
		}
		rates = append(rates, AccountUpstreamMonitorRate{
			RateKey:     strconv.FormatInt(group.ID, 10),
			DisplayName: name,
			Description: optionalString(group.Description),
			Ratio:       ratio,
			FirstSeenAt: &now,
			LastSeenAt:  &now,
		})
	}
	return snapshot, rates, nil
}

func (s *AccountUpstreamMonitorService) collectNewAPI(ctx context.Context, account *Account, creds *accountUpstreamMonitorCredentials, now time.Time) (*AccountUpstreamMonitorSnapshot, []AccountUpstreamMonitorRate, error) {
	quotaPerUnit := 500000.0
	if statusBody, err := s.getUpstreamMonitorJSON(ctx, account, creds.SiteURL+"/api/status", upstreamMonitorRequestAuth{}); err == nil {
		var status struct {
			QuotaPerUnit float64 `json:"quota_per_unit"`
		}
		if json.Unmarshal(statusBody, &status) == nil && status.QuotaPerUnit > 0 {
			quotaPerUnit = status.QuotaPerUnit
		}
	}

	auth := upstreamMonitorRequestAuth{Cookie: creds.NewAPICookie, NewAPIUserID: creds.NewAPIUserID}
	selfBody, err := s.getUpstreamMonitorJSON(ctx, account, creds.SiteURL+"/api/user/self", auth)
	if err != nil {
		return nil, nil, err
	}
	var self struct {
		Quota     float64 `json:"quota"`
		UsedQuota float64 `json:"used_quota"`
	}
	if err := json.Unmarshal(selfBody, &self); err != nil {
		return nil, nil, newUpstreamMonitorUnsupportedError("upstream monitor endpoint not supported", err)
	}
	balance := self.Quota / quotaPerUnit
	totalCost := self.UsedQuota / quotaPerUnit
	unit := "USD"
	snapshot := &AccountUpstreamMonitorSnapshot{
		Provider:      AccountUpstreamMonitorProviderNewAPI,
		Status:        AccountUpstreamMonitorStatusSuccess,
		Balance:       &balance,
		BalanceUnit:   &unit,
		Quota:         &self.Quota,
		QuotaUsed:     &self.UsedQuota,
		TotalCost:     &totalCost,
		LastCheckedAt: &now,
		LastSuccessAt: &now,
		RawMeta: map[string]any{
			"quota_per_unit": quotaPerUnit,
		},
	}

	groupsBody, err := s.getUpstreamMonitorJSON(ctx, account, creds.SiteURL+"/api/user/self/groups", auth)
	if err != nil {
		return nil, nil, err
	}
	raw := map[string]struct {
		Ratio json.RawMessage `json:"ratio"`
		Desc  string          `json:"desc"`
	}{}
	if err := json.Unmarshal(groupsBody, &raw); err != nil {
		return nil, nil, newUpstreamMonitorUnsupportedError("upstream monitor endpoint not supported", err)
	}
	rates := make([]AccountUpstreamMonitorRate, 0, len(raw))
	for name, item := range raw {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		var ratio float64
		if err := json.Unmarshal(item.Ratio, &ratio); err != nil {
			continue
		}
		rates = append(rates, AccountUpstreamMonitorRate{
			RateKey:     name,
			DisplayName: name,
			Description: optionalString(item.Desc),
			Ratio:       ratio,
			FirstSeenAt: &now,
			LastSeenAt:  &now,
		})
	}
	return snapshot, rates, nil
}

func (s *AccountUpstreamMonitorService) getUpstreamMonitorJSON(ctx context.Context, account *Account, endpoint string, auth upstreamMonitorRequestAuth) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, newUpstreamMonitorUnsupportedError("invalid upstream monitor URL", err)
	}
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(auth.BearerToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(auth.BearerToken))
	}
	if strings.TrimSpace(auth.Cookie) != "" {
		req.Header.Set("Cookie", strings.TrimSpace(auth.Cookie))
	}
	if strings.TrimSpace(auth.NewAPIUserID) != "" {
		req.Header.Set("New-Api-User", strings.TrimSpace(auth.NewAPIUserID))
	}

	resp, err := s.doMonitorRequest(req, account)
	if err != nil {
		return nil, newUpstreamMonitorFailureError("upstream monitor request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, accountUpstreamMonitorBodyLimit+1))
	if err != nil {
		return nil, newUpstreamMonitorFailureError("upstream monitor response read failed", err)
	}
	if int64(len(body)) > accountUpstreamMonitorBodyLimit {
		return nil, newUpstreamMonitorFailureError("upstream monitor response too large", nil)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, newUpstreamMonitorFailureError("upstream authentication failed", nil)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, newUpstreamMonitorUnsupportedError("upstream monitor endpoint not supported", nil)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, newUpstreamMonitorFailureError(fmt.Sprintf("upstream monitor request failed with HTTP %d", resp.StatusCode), nil)
	}
	return body, nil
}

func (s *AccountUpstreamMonitorService) doMonitorRequest(req *http.Request, account *Account) (*http.Response, error) {
	if s.httpUpstream == nil {
		return nil, errors.New("upstream HTTP client is not configured")
	}
	proxyURL := ""
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	if s.tlsFPProfileService == nil {
		return s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, nil)
	}
	return s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
}

type upstreamMonitorErrorKind string

const (
	upstreamMonitorErrorUnsupported upstreamMonitorErrorKind = "unsupported"
	upstreamMonitorErrorFailure     upstreamMonitorErrorKind = "failure"
)

type upstreamMonitorError struct {
	kind    upstreamMonitorErrorKind
	message string
	err     error
}

func (e *upstreamMonitorError) Error() string {
	if e == nil {
		return ""
	}
	if e.err == nil {
		return e.message
	}
	return e.message + ": " + e.err.Error()
}

func (e *upstreamMonitorError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func newUpstreamMonitorUnsupportedError(message string, err error) error {
	return &upstreamMonitorError{kind: upstreamMonitorErrorUnsupported, message: message, err: err}
}

func newUpstreamMonitorFailureError(message string, err error) error {
	return &upstreamMonitorError{kind: upstreamMonitorErrorFailure, message: message, err: err}
}

func shouldTryNextUpstreamMonitorProvider(err error) bool {
	var monitorErr *upstreamMonitorError
	if errors.As(err, &monitorErr) {
		return monitorErr.kind == upstreamMonitorErrorUnsupported || strings.Contains(monitorErr.message, "authentication")
	}
	return true
}

func isUpstreamMonitorFailure(err error) bool {
	var monitorErr *upstreamMonitorError
	return errors.As(err, &monitorErr) && monitorErr.kind == upstreamMonitorErrorFailure
}

func safeUpstreamMonitorError(err error) string {
	if err == nil {
		return ""
	}
	var monitorErr *upstreamMonitorError
	if errors.As(err, &monitorErr) && strings.TrimSpace(monitorErr.message) != "" {
		return monitorErr.message
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "context deadline") || strings.Contains(msg, "timeout"):
		return "upstream monitor request timed out"
	case strings.Contains(msg, "authentication") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden"):
		return "upstream authentication failed"
	default:
		return "upstream monitor request failed"
	}
}

func accountMonitorMapStringAnyFromAny(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	default:
		return nil, false
	}
}

func accountMonitorStringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func accountMonitorBoolFromAny(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on":
			return true
		default:
			return false
		}
	case json.Number:
		return typed.String() == "1"
	case int:
		return typed == 1
	case int64:
		return typed == 1
	case float64:
		return typed == 1
	default:
		return false
	}
}

func optionalString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func stringPtr(v string) *string {
	return &v
}

type AccountUpstreamMonitorRunnerService struct {
	monitorSvc *AccountUpstreamMonitorService
	settingSvc *SettingService
	cfg        *config.Config

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once
}

func NewAccountUpstreamMonitorRunnerService(
	monitorSvc *AccountUpstreamMonitorService,
	settingSvc *SettingService,
	cfg *config.Config,
) *AccountUpstreamMonitorRunnerService {
	return &AccountUpstreamMonitorRunnerService{monitorSvc: monitorSvc, settingSvc: settingSvc, cfg: cfg}
}

func (s *AccountUpstreamMonitorRunnerService) Start() {
	if s == nil || s.monitorSvc == nil {
		return
	}
	s.startOnce.Do(func() {
		loc := time.Local
		if s.cfg != nil {
			if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}
		c := cron.New(cron.WithParser(scheduledTestCronParser), cron.WithLocation(loc))
		_, err := c.AddFunc("* * * * *", func() { s.runOnce() })
		if err != nil {
			logger.LegacyPrintf("service.account_upstream_monitor", "[AccountUpstreamMonitorRunner] not started: %v", err)
			return
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.account_upstream_monitor", "[AccountUpstreamMonitorRunner] started")
	})
}

func (s *AccountUpstreamMonitorRunnerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.account_upstream_monitor", "[AccountUpstreamMonitorRunner] cron stop timed out")
			}
		}
	})
}

func (s *AccountUpstreamMonitorRunnerService) runOnce() {
	time.Sleep(15 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), accountUpstreamMonitorRunnerTimeout)
	defer cancel()

	runtime := AccountUpstreamMonitorRuntime{Enabled: true, IntervalMinutes: AccountUpstreamMonitorDefaultIntervalMins}
	if s.settingSvc != nil {
		runtime = s.settingSvc.GetAccountUpstreamMonitorRuntime(ctx)
	}
	if !runtime.Enabled {
		return
	}
	s.monitorSvc.RefreshDue(ctx, time.Duration(runtime.IntervalMinutes)*time.Minute, accountUpstreamMonitorDefaultMaxWorkers)
}
