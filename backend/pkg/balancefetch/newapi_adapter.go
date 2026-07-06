package balancefetch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"time"
)

type newAPILoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type newAPILoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		ID int `json:"id"`
	} `json:"data"`
}

type newAPIUserSelfResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		ID        int    `json:"id"`
		Username  string `json:"username"`
		Display   string `json:"display_name"`
		Group     string `json:"group"`
		Quota     int64  `json:"quota"`
		UsedQuota int64  `json:"used_quota"`
	} `json:"data"`
}

type newAPITokenResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Items []struct {
			ID             int    `json:"id"`
			Name           string `json:"name"`
			Key            string `json:"key"`
			Group          string `json:"group"`
			Status         int    `json:"status"`
			UnlimitedQuota bool   `json:"unlimited_quota"`
			RemainQuota    int64  `json:"remain_quota"`
			UsedQuota      int64  `json:"used_quota"`
		} `json:"items"`
	} `json:"data"`
}

type newAPIGroupInfo struct {
	Desc  string  `json:"desc"`
	Ratio float64 `json:"ratio"`
}

type newAPIGroupsResponse struct {
	Success bool                       `json:"success"`
	Message string                     `json:"message"`
	Data    map[string]newAPIGroupInfo `json:"data"`
}

func fetchNewAPISite(ctx context.Context, site siteConfig) (*unifiedSite, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("init cookie jar: %w", err)
	}
	client := &http.Client{Jar: jar, Timeout: 20 * time.Second}

	userID := 0
	accessToken := ""
	if useAccessTokenMode(site.AuthMode) {
		accessToken = normalizeBearerToken(site.AccessToken)
		if accessToken == "" {
			return nil, fmt.Errorf("empty access token")
		}
		userID, accessToken = parseNewAPIAccessToken(accessToken)
	} else {
		var err error
		userID, err = newAPILogin(ctx, client, site)
		if err != nil {
			return nil, fmt.Errorf("login: %w", err)
		}
	}
	userSelf, err := newAPIGetUserSelf(ctx, client, site.BaseURL, userID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("get user self: %w", err)
	}
	if userID <= 0 {
		userID = userSelf.Data.ID
	}
	groups, err := newAPIGetGroups(ctx, client, site.BaseURL, userID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("get groups: %w", err)
	}
	tokens, err := newAPIGetTokens(ctx, client, site.BaseURL, userID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("get tokens: %w", err)
	}

	return normalizeNewAPI(site, userSelf, groups, tokens), nil
}

func newAPILogin(ctx context.Context, client *http.Client, site siteConfig) (int, error) {
	var resp newAPILoginResponse
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, site.BaseURL+"/api/user/login?turnstile=", nil)
	if err != nil {
		return 0, err
	}
	setNewAPIHeaders(req, 0, "")
	if err := doJSON(client, req, newAPILoginRequest{Username: site.Username, Password: site.Password}, &resp); err != nil {
		return 0, err
	}
	if !resp.Success {
		return 0, fmt.Errorf("%s", resp.Message)
	}
	return resp.Data.ID, nil
}

func newAPIGetUserSelf(ctx context.Context, client *http.Client, baseURL string, userID int, accessToken string) (*newAPIUserSelfResponse, error) {
	var resp newAPIUserSelfResponse
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/user/self", nil)
	if err != nil {
		return nil, err
	}
	setNewAPIHeaders(req, userID, accessToken)
	if err := doJSON(client, req, nil, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("%s", resp.Message)
	}
	return &resp, nil
}

func newAPIGetGroups(ctx context.Context, client *http.Client, baseURL string, userID int, accessToken string) (*newAPIGroupsResponse, error) {
	var resp newAPIGroupsResponse
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/user/self/groups", nil)
	if err != nil {
		return nil, err
	}
	setNewAPIHeaders(req, userID, accessToken)
	if err := doJSON(client, req, nil, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("%s", resp.Message)
	}
	return &resp, nil
}

func newAPIGetTokens(ctx context.Context, client *http.Client, baseURL string, userID int, accessToken string) (*newAPITokenResponse, error) {
	var resp newAPITokenResponse
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/token/?p=1&size=20", nil)
	if err != nil {
		return nil, err
	}
	setNewAPIHeaders(req, userID, accessToken)
	if err := doJSON(client, req, nil, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("%s", resp.Message)
	}
	return &resp, nil
}

func setNewAPIHeaders(req *http.Request, userID int, accessToken string) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Cache-Control", "no-store")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	if userID > 0 {
		req.Header.Set("New-Api-User", strconv.Itoa(userID))
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
}

func parseNewAPIAccessToken(raw string) (int, string) {
	raw = strings.TrimSpace(raw)
	for _, sep := range []string{":", "|", ","} {
		left, right, ok := strings.Cut(raw, sep)
		if !ok {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(left))
		if err == nil && id > 0 && strings.TrimSpace(right) != "" {
			return id, strings.TrimSpace(right)
		}
	}
	return 0, raw
}

func normalizeNewAPI(site siteConfig, userSelf *newAPIUserSelfResponse, groups *newAPIGroupsResponse, tokens *newAPITokenResponse) *unifiedSite {
	out := &unifiedSite{
		Platform:  platformNewAPI,
		Name:      site.Name,
		BaseURL:   site.BaseURL,
		FetchedAt: time.Now().Format(time.RFC3339),
		Account: &unifiedAccount{
			ID:          strconv.Itoa(userSelf.Data.ID),
			Login:       userSelf.Data.Username,
			DisplayName: userSelf.Data.Display,
			GroupName:   userSelf.Data.Group,
			Balance:     newAPIAccountBalance(userSelf.Data.Quota),
		},
		Keys:   make([]unifiedKey, 0, len(tokens.Data.Items)),
		Groups: make([]unifiedGroup, 0, len(groups.Data)),
	}

	for name, group := range groups.Data {
		ratio := group.Ratio
		out.Groups = append(out.Groups, unifiedGroup{
			Name:           name,
			RateMultiplier: &ratio,
			Description:    group.Desc,
		})
	}

	for _, item := range tokens.Data.Items {
		group := groups.Data[item.Group]
		ratio := group.Ratio
		out.Keys = append(out.Keys, unifiedKey{
			ID:               strconv.Itoa(item.ID),
			Name:             item.Name,
			MaskedKey:        maskKey(item.Key),
			Status:           newAPIStatus(item.Status),
			GroupName:        item.Group,
			GroupDescription: group.Desc,
			RateMultiplier:   &ratio,
			KeyBalance: quotaBalance(
				0,
				float64(item.UsedQuota),
				float64(item.RemainQuota),
				item.UnlimitedQuota,
			),
		})
	}
	return out
}

func newAPIStatus(status int) string {
	if status == 1 {
		return "active"
	}
	return strconv.Itoa(status)
}
