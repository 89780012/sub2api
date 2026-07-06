package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/pkg/balancefetch"
)

const (
	BalancePlatformNewAPI  = "newapi"
	BalancePlatformSub2API = "sub2api"

	BalanceAuthModePassword    = "password"
	BalanceAuthModeAccessToken = "access_token"

	BalanceRefreshStatusSuccess = "success"
	BalanceRefreshStatusError   = "error"

	BalanceMatchUnmatched = "unmatched"
	BalanceMatchMatched   = "matched"
	BalanceMatchAmbiguous = "ambiguous"
	BalanceMatchManual    = "manual"

	BalanceMatchModeAuto   = "auto"
	BalanceMatchModeManual = "manual"
)

var (
	ErrBalanceSiteNotFound = infraerrors.NotFound("BALANCE_SITE_NOT_FOUND", "balance site not found")
	ErrBalanceSiteInvalid  = infraerrors.BadRequest("BALANCE_SITE_INVALID", "invalid balance site configuration")
)

type BalanceSite struct {
	ID                     int64      `json:"id"`
	Platform               string     `json:"platform"`
	Name                   string     `json:"name"`
	BaseURL                string     `json:"base_url"`
	AuthMode               string     `json:"auth_mode"`
	Username               string     `json:"username,omitempty"`
	Email                  string     `json:"email,omitempty"`
	PasswordEncrypted      string     `json:"-"`
	PasswordConfigured     bool       `json:"password_configured"`
	AccessTokenEncrypted   string     `json:"-"`
	AccessTokenConfigured  bool       `json:"access_token_configured"`
	Enabled                bool       `json:"enabled"`
	RefreshIntervalMinutes int        `json:"refresh_interval_minutes"`
	LastRefreshAt          *time.Time `json:"last_refresh_at,omitempty"`
	LastRefreshStatus      string     `json:"last_refresh_status"`
	LastRefreshError       string     `json:"last_refresh_error,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type BalanceSiteInput struct {
	Platform               string
	Name                   string
	BaseURL                string
	AuthMode               string
	Username               string
	Email                  string
	Password               *string
	AccessToken            *string
	Enabled                *bool
	RefreshIntervalMinutes *int
}

type BalanceSnapshot struct {
	ID                  int64          `json:"id"`
	SiteID              int64          `json:"site_id"`
	SiteName            string         `json:"site_name,omitempty"`
	SitePlatform        string         `json:"site_platform,omitempty"`
	ExternalKeyID       string         `json:"external_key_id"`
	MaskedKey           string         `json:"masked_key"`
	KeyLast4            string         `json:"key_last4"`
	AccountID           *int64         `json:"account_id,omitempty"`
	MatchStatus         string         `json:"match_status"`
	KeyName             string         `json:"key_name,omitempty"`
	Status              string         `json:"status,omitempty"`
	GroupID             string         `json:"group_id,omitempty"`
	GroupName           string         `json:"group_name,omitempty"`
	Balance             map[string]any `json:"balance,omitempty"`
	AccountBalance      map[string]any `json:"account_balance,omitempty"`
	SubscriptionBalance map[string]any `json:"subscription_balance,omitempty"`
	RateMultiplier      *float64       `json:"rate_multiplier,omitempty"`
	FetchedAt           time.Time      `json:"fetched_at"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

type AccountBalanceBinding struct {
	AccountID     int64     `json:"account_id"`
	SiteID        int64     `json:"site_id"`
	ExternalKeyID *string   `json:"external_key_id,omitempty"`
	KeyLast4      string    `json:"key_last4"`
	MatchMode     string    `json:"match_mode"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type BalanceBindingInput struct {
	SiteID        int64
	ExternalKeyID *string
	KeyLast4      string
}

type BalanceRefreshResult struct {
	SiteID        int64      `json:"site_id"`
	Status        string     `json:"status"`
	Error         string     `json:"error,omitempty"`
	SnapshotCount int        `json:"snapshot_count"`
	RefreshedAt   *time.Time `json:"refreshed_at,omitempty"`
}

type BalanceSnapshotRepository interface {
	ListSites(ctx context.Context) ([]BalanceSite, error)
	GetSite(ctx context.Context, id int64) (*BalanceSite, error)
	CreateSite(ctx context.Context, site *BalanceSite) error
	UpdateSite(ctx context.Context, site *BalanceSite) error
	DeleteSite(ctx context.Context, id int64) error
	ListDueSites(ctx context.Context, now time.Time) ([]BalanceSite, error)
	UpdateSiteRefreshStatus(ctx context.Context, siteID int64, status, message string, refreshedAt *time.Time) error
	ListSnapshots(ctx context.Context, siteID int64) ([]BalanceSnapshot, error)
	ListSnapshotsByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]*BalanceSnapshot, error)
	ReplaceSiteSnapshots(ctx context.Context, siteID int64, snapshots []BalanceSnapshot) error
	UpsertBinding(ctx context.Context, binding AccountBalanceBinding) error
	DeleteBinding(ctx context.Context, accountID int64) error
	ListBindings(ctx context.Context) ([]AccountBalanceBinding, error)
}

type BalanceSnapshotService struct {
	repo        BalanceSnapshotRepository
	accountRepo AccountRepository
	encryptor   SecretEncryptor
	stopCh      chan struct{}
	stopOnce    sync.Once
}

func NewBalanceSnapshotService(repo BalanceSnapshotRepository, accountRepo AccountRepository, encryptor SecretEncryptor) *BalanceSnapshotService {
	return &BalanceSnapshotService{
		repo:        repo,
		accountRepo: accountRepo,
		encryptor:   encryptor,
		stopCh:      make(chan struct{}),
	}
}

func (s *BalanceSnapshotService) Start() {
	go s.loop()
}

func (s *BalanceSnapshotService) Stop() {
	s.stopOnce.Do(func() { close(s.stopCh) })
}

func (s *BalanceSnapshotService) loop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	s.refreshDue(context.Background())
	for {
		select {
		case <-ticker.C:
			s.refreshDue(context.Background())
		case <-s.stopCh:
			return
		}
	}
}

func (s *BalanceSnapshotService) refreshDue(ctx context.Context) {
	if s == nil || s.repo == nil {
		return
	}
	sites, err := s.repo.ListDueSites(ctx, time.Now())
	if err != nil {
		logger.LegacyPrintf("service.balance_snapshot", "list due balance sites failed: %v", err)
		return
	}
	for i := range sites {
		siteID := sites[i].ID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			if _, err := s.RefreshSite(ctx, siteID); err != nil {
				logger.LegacyPrintf("service.balance_snapshot", "refresh balance site %d failed: %v", siteID, err)
			}
		}()
	}
}

func (s *BalanceSnapshotService) ListSites(ctx context.Context) ([]BalanceSite, error) {
	return s.repo.ListSites(ctx)
}

func (s *BalanceSnapshotService) CreateSite(ctx context.Context, input BalanceSiteInput) (*BalanceSite, error) {
	site, err := s.normalizeSiteInput(input, nil)
	if err != nil {
		return nil, err
	}
	if site.AuthMode == BalanceAuthModeAccessToken {
		if input.AccessToken == nil || strings.TrimSpace(*input.AccessToken) == "" {
			return nil, infraerrors.BadRequest("BALANCE_SITE_ACCESS_TOKEN_REQUIRED", "access token is required")
		}
		encrypted, err := s.encryptor.Encrypt(strings.TrimSpace(*input.AccessToken))
		if err != nil {
			return nil, err
		}
		site.AccessTokenEncrypted = encrypted
	} else {
		if input.Password == nil || strings.TrimSpace(*input.Password) == "" {
			return nil, infraerrors.BadRequest("BALANCE_SITE_PASSWORD_REQUIRED", "password is required")
		}
		encrypted, err := s.encryptor.Encrypt(strings.TrimSpace(*input.Password))
		if err != nil {
			return nil, err
		}
		site.PasswordEncrypted = encrypted
	}
	if err := s.repo.CreateSite(ctx, site); err != nil {
		return nil, err
	}
	site.PasswordConfigured = site.PasswordEncrypted != ""
	site.AccessTokenConfigured = site.AccessTokenEncrypted != ""
	return site, nil
}

func (s *BalanceSnapshotService) UpdateSite(ctx context.Context, id int64, input BalanceSiteInput) (*BalanceSite, error) {
	existing, err := s.repo.GetSite(ctx, id)
	if err != nil {
		return nil, err
	}
	site, err := s.normalizeSiteInput(input, existing)
	if err != nil {
		return nil, err
	}
	site.ID = id
	site.PasswordEncrypted = existing.PasswordEncrypted
	site.AccessTokenEncrypted = existing.AccessTokenEncrypted
	if input.Password != nil && strings.TrimSpace(*input.Password) != "" {
		encrypted, err := s.encryptor.Encrypt(strings.TrimSpace(*input.Password))
		if err != nil {
			return nil, err
		}
		site.PasswordEncrypted = encrypted
	}
	if input.AccessToken != nil && strings.TrimSpace(*input.AccessToken) != "" {
		encrypted, err := s.encryptor.Encrypt(strings.TrimSpace(*input.AccessToken))
		if err != nil {
			return nil, err
		}
		site.AccessTokenEncrypted = encrypted
	}
	if site.AuthMode == BalanceAuthModePassword && site.PasswordEncrypted == "" {
		return nil, infraerrors.BadRequest("BALANCE_SITE_PASSWORD_REQUIRED", "password is required")
	}
	if site.AuthMode == BalanceAuthModeAccessToken && site.AccessTokenEncrypted == "" {
		return nil, infraerrors.BadRequest("BALANCE_SITE_ACCESS_TOKEN_REQUIRED", "access token is required")
	}
	if err := s.repo.UpdateSite(ctx, site); err != nil {
		return nil, err
	}
	site.PasswordConfigured = site.PasswordEncrypted != ""
	site.AccessTokenConfigured = site.AccessTokenEncrypted != ""
	return site, nil
}

func (s *BalanceSnapshotService) DeleteSite(ctx context.Context, id int64) error {
	return s.repo.DeleteSite(ctx, id)
}

func (s *BalanceSnapshotService) ListSnapshots(ctx context.Context, siteID int64) ([]BalanceSnapshot, error) {
	return s.repo.ListSnapshots(ctx, siteID)
}

func (s *BalanceSnapshotService) UpsertBinding(ctx context.Context, accountID int64, input BalanceBindingInput) error {
	if accountID <= 0 || input.SiteID <= 0 {
		return infraerrors.BadRequest("BALANCE_BINDING_INVALID", "invalid balance binding")
	}
	if _, err := s.accountRepo.GetByID(ctx, accountID); err != nil {
		return err
	}
	keyLast4 := strings.TrimSpace(input.KeyLast4)
	if keyLast4 == "" && input.ExternalKeyID != nil {
		snapshots, err := s.repo.ListSnapshots(ctx, input.SiteID)
		if err != nil {
			return err
		}
		for i := range snapshots {
			if snapshots[i].ExternalKeyID == *input.ExternalKeyID {
				keyLast4 = snapshots[i].KeyLast4
				break
			}
		}
	}
	if keyLast4 == "" {
		return infraerrors.BadRequest("BALANCE_BINDING_KEY_REQUIRED", "key_last4 is required")
	}
	return s.repo.UpsertBinding(ctx, AccountBalanceBinding{
		AccountID:     accountID,
		SiteID:        input.SiteID,
		ExternalKeyID: input.ExternalKeyID,
		KeyLast4:      keyLast4,
		MatchMode:     BalanceMatchModeManual,
	})
}

func (s *BalanceSnapshotService) DeleteBinding(ctx context.Context, accountID int64) error {
	return s.repo.DeleteBinding(ctx, accountID)
}

func (s *BalanceSnapshotService) SnapshotsByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]*BalanceSnapshot, error) {
	return s.repo.ListSnapshotsByAccountIDs(ctx, accountIDs)
}

func (s *BalanceSnapshotService) RefreshAll(ctx context.Context) ([]BalanceRefreshResult, error) {
	sites, err := s.repo.ListSites(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]BalanceRefreshResult, 0, len(sites))
	for i := range sites {
		if !sites[i].Enabled {
			continue
		}
		result, err := s.RefreshSite(ctx, sites[i].ID)
		if err != nil {
			results = append(results, BalanceRefreshResult{SiteID: sites[i].ID, Status: BalanceRefreshStatusError, Error: err.Error()})
			continue
		}
		results = append(results, *result)
	}
	return results, nil
}

func (s *BalanceSnapshotService) RefreshSite(ctx context.Context, siteID int64) (*BalanceRefreshResult, error) {
	site, err := s.repo.GetSite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	baseURL, err := balancefetch.NormalizeBaseURL(site.BaseURL)
	if err != nil {
		_ = s.repo.UpdateSiteRefreshStatus(ctx, siteID, BalanceRefreshStatusError, err.Error(), nil)
		return nil, err
	}
	fetchConfig := balancefetch.SiteConfig{
		Platform: balancefetch.Platform(site.Platform),
		Name:     site.Name,
		BaseURL:  baseURL,
		AuthMode: site.AuthMode,
		Username: site.Username,
		Email:    site.Email,
	}
	if site.AuthMode == BalanceAuthModeAccessToken {
		accessToken, err := s.encryptor.Decrypt(site.AccessTokenEncrypted)
		if err != nil {
			_ = s.repo.UpdateSiteRefreshStatus(ctx, siteID, BalanceRefreshStatusError, "decrypt access token failed", nil)
			return nil, err
		}
		fetchConfig.AccessToken = accessToken
	} else {
		password, err := s.encryptor.Decrypt(site.PasswordEncrypted)
		if err != nil {
			_ = s.repo.UpdateSiteRefreshStatus(ctx, siteID, BalanceRefreshStatusError, "decrypt password failed", nil)
			return nil, err
		}
		fetchConfig.Password = password
	}
	fetched, err := balancefetch.FetchConfiguredSiteContext(ctx, fetchConfig)
	if err != nil {
		_ = s.repo.UpdateSiteRefreshStatus(ctx, siteID, BalanceRefreshStatusError, err.Error(), nil)
		return nil, err
	}
	now := time.Now()
	snapshots, err := s.snapshotsFromFetched(ctx, site, fetched, now)
	if err != nil {
		_ = s.repo.UpdateSiteRefreshStatus(ctx, siteID, BalanceRefreshStatusError, err.Error(), nil)
		return nil, err
	}
	if err := s.repo.ReplaceSiteSnapshots(ctx, siteID, snapshots); err != nil {
		_ = s.repo.UpdateSiteRefreshStatus(ctx, siteID, BalanceRefreshStatusError, err.Error(), nil)
		return nil, err
	}
	if err := s.repo.UpdateSiteRefreshStatus(ctx, siteID, BalanceRefreshStatusSuccess, "", &now); err != nil {
		return nil, err
	}
	return &BalanceRefreshResult{SiteID: siteID, Status: BalanceRefreshStatusSuccess, SnapshotCount: len(snapshots), RefreshedAt: &now}, nil
}

func (s *BalanceSnapshotService) normalizeSiteInput(input BalanceSiteInput, existing *BalanceSite) (*BalanceSite, error) {
	site := &BalanceSite{}
	if existing != nil {
		*site = *existing
	}
	platform := strings.ToLower(strings.TrimSpace(input.Platform))
	if platform != "" {
		site.Platform = platform
	}
	if site.Platform != BalancePlatformNewAPI && site.Platform != BalancePlatformSub2API {
		return nil, ErrBalanceSiteInvalid
	}
	authMode := strings.ToLower(strings.TrimSpace(input.AuthMode))
	if authMode != "" {
		site.AuthMode = authMode
	}
	if site.AuthMode == "" {
		site.AuthMode = BalanceAuthModePassword
	}
	if site.AuthMode != BalanceAuthModePassword && site.AuthMode != BalanceAuthModeAccessToken {
		return nil, infraerrors.BadRequest("BALANCE_SITE_AUTH_MODE_INVALID", "invalid balance site auth mode")
	}
	if strings.TrimSpace(input.Name) != "" {
		site.Name = strings.TrimSpace(input.Name)
	}
	baseURL := strings.TrimSpace(input.BaseURL)
	if baseURL != "" {
		normalized, err := balancefetch.NormalizeBaseURL(baseURL)
		if err != nil {
			return nil, infraerrors.BadRequest("BALANCE_SITE_URL_INVALID", err.Error())
		}
		site.BaseURL = normalized
	}
	site.Username = strings.TrimSpace(input.Username)
	site.Email = strings.TrimSpace(input.Email)
	if existing != nil {
		if input.Username == "" {
			site.Username = existing.Username
		}
		if input.Email == "" {
			site.Email = existing.Email
		}
	}
	if site.Name == "" {
		site.Name = site.Platform
	}
	if site.BaseURL == "" {
		return nil, infraerrors.BadRequest("BALANCE_SITE_URL_REQUIRED", "base_url is required")
	}
	if site.AuthMode == BalanceAuthModePassword && site.Platform == BalancePlatformNewAPI && site.Username == "" {
		return nil, infraerrors.BadRequest("BALANCE_SITE_USERNAME_REQUIRED", "username is required")
	}
	if site.AuthMode == BalanceAuthModePassword && site.Platform == BalancePlatformSub2API && site.Email == "" {
		return nil, infraerrors.BadRequest("BALANCE_SITE_EMAIL_REQUIRED", "email is required")
	}
	if input.Enabled != nil {
		site.Enabled = *input.Enabled
	} else if existing == nil {
		site.Enabled = true
	}
	if input.RefreshIntervalMinutes != nil {
		site.RefreshIntervalMinutes = *input.RefreshIntervalMinutes
	}
	if site.RefreshIntervalMinutes <= 0 {
		site.RefreshIntervalMinutes = 180
	}
	return site, nil
}

func (s *BalanceSnapshotService) snapshotsFromFetched(ctx context.Context, site *BalanceSite, fetched *balancefetch.Site, now time.Time) ([]BalanceSnapshot, error) {
	if fetched == nil {
		return nil, nil
	}
	accounts, _, err := s.accountRepo.List(ctx, pagination.PaginationParams{Page: 1, PageSize: 100000})
	if err != nil {
		return nil, err
	}
	bindings, err := s.repo.ListBindings(ctx)
	if err != nil {
		return nil, err
	}
	manualByExternal := make(map[string]AccountBalanceBinding)
	manualByLast4 := make(map[string]AccountBalanceBinding)
	for _, binding := range bindings {
		if binding.SiteID != site.ID {
			continue
		}
		if binding.ExternalKeyID != nil && *binding.ExternalKeyID != "" {
			manualByExternal[*binding.ExternalKeyID] = binding
		}
		if binding.KeyLast4 != "" {
			manualByLast4[binding.KeyLast4] = binding
		}
	}
	accountsByLast4 := make(map[string][]int64)
	for i := range accounts {
		last4 := accountKeyLast4(&accounts[i])
		if last4 == "" {
			continue
		}
		accountsByLast4[last4] = append(accountsByLast4[last4], accounts[i].ID)
	}
	out := make([]BalanceSnapshot, 0, len(fetched.Keys))
	for _, key := range fetched.Keys {
		last4 := last4FromMaskedKey(key.MaskedKey)
		status := BalanceMatchUnmatched
		var accountID *int64
		if binding, ok := manualByExternal[key.ID]; ok {
			id := binding.AccountID
			accountID = &id
			status = BalanceMatchManual
		} else if binding, ok := manualByLast4[last4]; ok {
			id := binding.AccountID
			accountID = &id
			status = BalanceMatchManual
		} else if matches := accountsByLast4[last4]; last4 != "" {
			switch len(matches) {
			case 0:
			case 1:
				id := matches[0]
				accountID = &id
				status = BalanceMatchMatched
			default:
				status = BalanceMatchAmbiguous
			}
		}
		out = append(out, BalanceSnapshot{
			SiteID:              site.ID,
			SiteName:            site.Name,
			SitePlatform:        site.Platform,
			ExternalKeyID:       key.ID,
			MaskedKey:           key.MaskedKey,
			KeyLast4:            last4,
			AccountID:           accountID,
			MatchStatus:         status,
			KeyName:             key.Name,
			Status:              key.Status,
			GroupID:             key.GroupID,
			GroupName:           key.GroupName,
			Balance:             structToMap(key.KeyBalance),
			AccountBalance:      fetchedAccountBalance(fetched),
			SubscriptionBalance: structToMap(key.SubscriptionBalance),
			RateMultiplier:      key.RateMultiplier,
			FetchedAt:           now,
		})
	}
	return out, nil
}

func fetchedAccountBalance(site *balancefetch.Site) map[string]any {
	if site == nil || site.Account == nil {
		return nil
	}
	return structToMap(site.Account.Balance)
}

func accountKeyLast4(account *Account) string {
	if account == nil {
		return ""
	}
	for _, key := range []string{"api_key", "key", "access_token"} {
		value := account.GetCredential(key)
		if len(value) >= 4 {
			return value[len(value)-4:]
		}
	}
	return ""
}

func last4FromMaskedKey(masked string) string {
	masked = strings.TrimSpace(masked)
	if len(masked) < 4 {
		return ""
	}
	return masked[len(masked)-4:]
}

func structToMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
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

func (s *BalanceSnapshotService) String() string {
	return fmt.Sprintf("BalanceSnapshotService")
}
