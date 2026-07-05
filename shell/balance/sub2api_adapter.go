package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const timezone = "Asia/Shanghai"

type sub2APIResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type sub2APILoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type sub2APILoginData struct {
	AccessToken string `json:"access_token"`
}

type sub2APIMeData struct {
	ID             int64   `json:"id"`
	Email          string  `json:"email"`
	Role           string  `json:"role"`
	Status         string  `json:"status"`
	Balance        float64 `json:"balance"`
	TotalRecharged float64 `json:"total_recharged"`
	Concurrency    int     `json:"concurrency"`
	RPMLimit       int     `json:"rpm_limit"`
}

type sub2APIGroup struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Platform         string  `json:"platform"`
	RateMultiplier   float64 `json:"rate_multiplier"`
	SubscriptionType string  `json:"subscription_type"`
	DailyLimitUSD    float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD   float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD  float64 `json:"monthly_limit_usd"`
	Status           string  `json:"status"`
}

type sub2APIKeyItem struct {
	ID        int64        `json:"id"`
	Key       string       `json:"key"`
	MaskedKey string       `json:"masked_key"`
	Name      string       `json:"name"`
	GroupID   int64        `json:"group_id"`
	Status    string       `json:"status"`
	Quota     float64      `json:"quota"`
	QuotaUsed float64      `json:"quota_used"`
	Group     sub2APIGroup `json:"group"`
}

type sub2APIKeysData struct {
	Items []sub2APIKeyItem `json:"items"`
}

type sub2APISubscriptionItem struct {
	ID              int64        `json:"id"`
	GroupID         int64        `json:"group_id"`
	Status          string       `json:"status"`
	StartsAt        string       `json:"starts_at"`
	ExpiresAt       string       `json:"expires_at"`
	DailyUsageUSD   float64      `json:"daily_usage_usd"`
	WeeklyUsageUSD  float64      `json:"weekly_usage_usd"`
	MonthlyUsageUSD float64      `json:"monthly_usage_usd"`
	Group           sub2APIGroup `json:"group"`
}

func fetchSub2APISite(site siteConfig) (*unifiedSite, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	token, err := sub2APILogin(client, site)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	me, err := sub2APIGetMe(client, site.BaseURL, token)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	keys, err := sub2APIGetKeys(client, site.BaseURL, token)
	if err != nil {
		return nil, fmt.Errorf("get keys: %w", err)
	}
	groups, err := sub2APIGetGroups(client, site.BaseURL, token)
	if err != nil {
		return nil, fmt.Errorf("get groups: %w", err)
	}
	subs, err := sub2APIGetSubscriptions(client, site.BaseURL, token)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions: %w", err)
	}

	return normalizeSub2API(site, me, keys, groups, subs), nil
}

func sub2APILogin(client *http.Client, site siteConfig) (string, error) {
	data, err := doSub2JSON[sub2APILoginData](client, site.BaseURL, http.MethodPost, "/api/v1/auth/login", "", sub2APILoginRequest{
		Email:    site.Email,
		Password: site.Password,
	})
	if err != nil {
		return "", err
	}
	if data.AccessToken == "" {
		return "", fmt.Errorf("empty access token")
	}
	return data.AccessToken, nil
}

func sub2APIGetMe(client *http.Client, baseURL, token string) (*sub2APIMeData, error) {
	path := "/api/v1/auth/me?timezone=" + url.QueryEscape(timezone)
	data, err := doSub2JSON[sub2APIMeData](client, baseURL, http.MethodGet, path, token, nil)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func sub2APIGetKeys(client *http.Client, baseURL, token string) (*sub2APIKeysData, error) {
	path := "/api/v1/keys?page=1&page_size=100&sort_by=created_at&sort_order=desc&timezone=" + url.QueryEscape(timezone)
	data, err := doSub2JSON[sub2APIKeysData](client, baseURL, http.MethodGet, path, token, nil)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func sub2APIGetGroups(client *http.Client, baseURL, token string) ([]sub2APIGroup, error) {
	path := "/api/v1/groups/available?timezone=" + url.QueryEscape(timezone)
	return doSub2JSON[[]sub2APIGroup](client, baseURL, http.MethodGet, path, token, nil)
}

func sub2APIGetSubscriptions(client *http.Client, baseURL, token string) ([]sub2APISubscriptionItem, error) {
	path := "/api/v1/subscriptions/active?timezone=" + url.QueryEscape(timezone)
	return doSub2JSON[[]sub2APISubscriptionItem](client, baseURL, http.MethodGet, path, token, nil)
}

func doSub2JSON[T any](client *http.Client, baseURL, method, path, token string, reqBody any) (T, error) {
	var zero T
	req, err := http.NewRequest(method, baseURL+path, nil)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh")
	if reqBody != nil {
		req.Header.Set("Origin", baseURL)
		req.Header.Set("Referer", baseURL+"/login")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	var resp sub2APIResponse[T]
	if err := doJSON(client, req, reqBody, &resp); err != nil {
		return zero, err
	}
	if resp.Code != 0 {
		return zero, fmt.Errorf("business error: code=%d message=%s", resp.Code, resp.Message)
	}
	return resp.Data, nil
}

func normalizeSub2API(site siteConfig, me *sub2APIMeData, keys *sub2APIKeysData, groups []sub2APIGroup, subs []sub2APISubscriptionItem) *unifiedSite {
	groupMap := make(map[int64]sub2APIGroup, len(groups))
	outGroups := make([]unifiedGroup, 0, len(groups))
	for _, group := range groups {
		groupMap[group.ID] = group
		rate := group.RateMultiplier
		outGroups = append(outGroups, unifiedGroup{
			ID:               strconv.FormatInt(group.ID, 10),
			Name:             group.Name,
			Platform:         group.Platform,
			RateMultiplier:   &rate,
			SubscriptionType: group.SubscriptionType,
			Status:           group.Status,
			DailyLimitUSD:    floatPtr(group.DailyLimitUSD),
			WeeklyLimitUSD:   floatPtr(group.WeeklyLimitUSD),
			MonthlyLimitUSD:  floatPtr(group.MonthlyLimitUSD),
		})
	}

	subByGroup := make(map[int64]subscriptionBalance, len(subs))
	outSubs := make([]unifiedSubscription, 0, len(subs))
	for _, sub := range subs {
		normalized := normalizeSub2APISubscription(sub, groupMap)
		outSubs = append(outSubs, normalized)
		if _, exists := subByGroup[sub.GroupID]; !exists {
			subByGroup[sub.GroupID] = subscriptionBalance(normalized)
		}
	}

	outKeys := make([]unifiedKey, 0, len(keys.Items))
	for _, item := range keys.Items {
		groupName := item.Group.Name
		rate := item.Group.RateMultiplier
		if group, ok := groupMap[item.GroupID]; ok {
			groupName = group.Name
			rate = group.RateMultiplier
		}

		masked := item.MaskedKey
		if masked == "" {
			masked = maskKey(item.Key)
		}

		var keyBalance *balance
		if item.Quota != 0 || item.QuotaUsed != 0 {
			keyBalance = usdBalance(item.Quota, item.QuotaUsed)
		}

		var subBalance *subscriptionBalance
		if sub, ok := subByGroup[item.GroupID]; ok {
			copy := sub
			subBalance = &copy
		}

		outKeys = append(outKeys, unifiedKey{
			ID:                  strconv.FormatInt(item.ID, 10),
			Name:                item.Name,
			MaskedKey:           masked,
			Status:              item.Status,
			GroupID:             strconv.FormatInt(item.GroupID, 10),
			GroupName:           groupName,
			RateMultiplier:      &rate,
			KeyBalance:          keyBalance,
			SubscriptionBalance: subBalance,
		})
	}

	return &unifiedSite{
		Platform: platformSub2API,
		Name:     site.Name,
		BaseURL:  site.BaseURL,
		Account: &unifiedAccount{
			ID:             strconv.FormatInt(me.ID, 10),
			Login:          me.Email,
			Role:           me.Role,
			Status:         me.Status,
			Balance:        accountUSDBalance(me.Balance),
			TotalRecharged: floatPtr(me.TotalRecharged),
			Concurrency:    intPtr(me.Concurrency),
			RPMLimit:       intPtr(me.RPMLimit),
		},
		Keys:          outKeys,
		Groups:        outGroups,
		Subscriptions: outSubs,
		FetchedAt:     time.Now().Format(time.RFC3339),
	}
}

func normalizeSub2APISubscription(sub sub2APISubscriptionItem, groupMap map[int64]sub2APIGroup) unifiedSubscription {
	group := sub.Group
	if mapped, ok := groupMap[sub.GroupID]; ok {
		group = mapped
	}
	rate := group.RateMultiplier
	return unifiedSubscription{
		ID:             strconv.FormatInt(sub.ID, 10),
		Status:         sub.Status,
		GroupID:        strconv.FormatInt(sub.GroupID, 10),
		GroupName:      group.Name,
		RateMultiplier: &rate,
		StartsAt:       sub.StartsAt,
		ExpiresAt:      sub.ExpiresAt,
		Daily:          makeSubscriptionWindow(group.DailyLimitUSD, sub.DailyUsageUSD),
		Weekly:         makeSubscriptionWindow(group.WeeklyLimitUSD, sub.WeeklyUsageUSD),
		Monthly:        makeSubscriptionWindow(group.MonthlyLimitUSD, sub.MonthlyUsageUSD),
	}
}
