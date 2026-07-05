package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://ai.input.im"
	timezone       = "Asia/Shanghai"
)

type APIResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type SiteConfig struct {
	Name     string
	BaseURL  string
	Email    string
	Password string
	Legacy   bool
}

type MeData struct {
	ID             int64   `json:"id"`
	Email          string  `json:"email"`
	Role           string  `json:"role"`
	Status         string  `json:"status"`
	Balance        float64 `json:"balance"`
	TotalRecharged float64 `json:"total_recharged"`
	Concurrency    int     `json:"concurrency"`
	RPMLimit       int     `json:"rpm_limit"`
}

type Group struct {
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

type KeyItem struct {
	ID        int64   `json:"id"`
	Key       string  `json:"key"`
	Name      string  `json:"name"`
	GroupID   int64   `json:"group_id"`
	Status    string  `json:"status"`
	Quota     float64 `json:"quota"`
	QuotaUsed float64 `json:"quota_used"`
	Group     Group   `json:"group"`
}

type KeysData struct {
	Items []KeyItem `json:"items"`
}

type SubscriptionItem struct {
	ID              int64   `json:"id"`
	GroupID         int64   `json:"group_id"`
	Status          string  `json:"status"`
	StartsAt        string  `json:"starts_at"`
	ExpiresAt       string  `json:"expires_at"`
	DailyUsageUSD   float64 `json:"daily_usage_usd"`
	WeeklyUsageUSD  float64 `json:"weekly_usage_usd"`
	MonthlyUsageUSD float64 `json:"monthly_usage_usd"`
	Group           Group   `json:"group"`
}

type Output struct {
	AccountBalance float64              `json:"account_balance"`
	Account        OutputAccount        `json:"account"`
	Keys           []OutputKey          `json:"keys"`
	Groups         []OutputGroup        `json:"groups"`
	Subscriptions  []OutputSubscription `json:"subscriptions"`
	FetchedAt      string               `json:"fetched_at"`
}

type MultiSiteOutput struct {
	Sites     []SiteOutput `json:"sites"`
	FetchedAt string       `json:"fetched_at"`
}

type SiteOutput struct {
	Name    string  `json:"name,omitempty"`
	BaseURL string  `json:"base_url"`
	Error   string  `json:"error,omitempty"`
	Data    *Output `json:"data,omitempty"`
}

type OutputAccount struct {
	ID             int64   `json:"id"`
	Email          string  `json:"email"`
	Role           string  `json:"role"`
	Status         string  `json:"status"`
	Balance        float64 `json:"balance"`
	TotalRecharged float64 `json:"total_recharged"`
	Concurrency    int     `json:"concurrency"`
	RPMLimit       int     `json:"rpm_limit"`
}

type OutputGroup struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Platform         string  `json:"platform"`
	RateMultiplier   float64 `json:"rate_multiplier"`
	SubscriptionType string  `json:"subscription_type"`
	Status           string  `json:"status"`
	DailyLimitUSD    float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD   float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD  float64 `json:"monthly_limit_usd"`
}

type OutputKey struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Key            string  `json:"key"`
	MaskedKey      string  `json:"masked_key"`
	Status         string  `json:"status"`
	GroupID        int64   `json:"group_id"`
	GroupName      string  `json:"group_name"`
	RateMultiplier float64 `json:"rate_multiplier"`
	Quota          float64 `json:"quota"`
	QuotaUsed      float64 `json:"quota_used"`
}

type OutputSubscription struct {
	ID             int64         `json:"id"`
	Status         string        `json:"status"`
	GroupID        int64         `json:"group_id"`
	GroupName      string        `json:"group_name"`
	RateMultiplier float64       `json:"rate_multiplier"`
	StartsAt       string        `json:"starts_at"`
	ExpiresAt      string        `json:"expires_at"`
	Daily          BalanceWindow `json:"daily"`
	Weekly         BalanceWindow `json:"weekly"`
	Monthly        BalanceWindow `json:"monthly"`
}

type BalanceWindow struct {
	LimitUSD  float64 `json:"limit_usd"`
	UsedUSD   float64 `json:"used_usd"`
	RemainUSD float64 `json:"remain_usd"`
}

func main() {
	if err := loadDotEnv(); err != nil {
		exitErr("load .env failed", err)
	}

	sites, err := loadSiteConfigs()
	if err != nil {
		exitErr("load site config failed", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	if len(sites) == 1 && sites[0].Legacy {
		out, err := fetchSite(client, sites[0])
		if err != nil {
			exitErr("fetch site failed", err)
		}
		if err := encodeJSON(out); err != nil {
			exitErr("encode JSON failed", err)
		}
		return
	}

	out := MultiSiteOutput{
		Sites:     make([]SiteOutput, 0, len(sites)),
		FetchedAt: time.Now().Format(time.RFC3339),
	}
	for _, site := range sites {
		result := SiteOutput{
			Name:    site.Name,
			BaseURL: site.BaseURL,
		}
		data, err := fetchSite(client, site)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}
		out.Sites = append(out.Sites, result)
	}

	if err := encodeJSON(out); err != nil {
		exitErr("encode JSON failed", err)
	}
}

func fetchSite(client *http.Client, site SiteConfig) (*Output, error) {
	token, err := login(client, site.BaseURL, site.Email, site.Password)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	me, err := getMe(client, site.BaseURL, token)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	keysData, err := getKeys(client, site.BaseURL, token)
	if err != nil {
		return nil, fmt.Errorf("get keys: %w", err)
	}

	groups, err := getGroups(client, site.BaseURL, token)
	if err != nil {
		return nil, fmt.Errorf("get groups: %w", err)
	}

	subs, err := getSubscriptions(client, site.BaseURL, token)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions: %w", err)
	}

	groupMap := make(map[int64]Group, len(groups))
	outputGroups := make([]OutputGroup, 0, len(groups))
	for _, g := range groups {
		groupMap[g.ID] = g
		outputGroups = append(outputGroups, OutputGroup{
			ID:               g.ID,
			Name:             g.Name,
			Platform:         g.Platform,
			RateMultiplier:   g.RateMultiplier,
			SubscriptionType: g.SubscriptionType,
			Status:           g.Status,
			DailyLimitUSD:    g.DailyLimitUSD,
			WeeklyLimitUSD:   g.WeeklyLimitUSD,
			MonthlyLimitUSD:  g.MonthlyLimitUSD,
		})
	}

	outputKeys := make([]OutputKey, 0, len(keysData.Items))
	for _, item := range keysData.Items {
		groupName := item.Group.Name
		rate := item.Group.RateMultiplier

		if g, ok := groupMap[item.GroupID]; ok {
			groupName = g.Name
			rate = g.RateMultiplier
		}

		outputKeys = append(outputKeys, OutputKey{
			ID:             item.ID,
			Name:           item.Name,
			Key:            item.Key,
			MaskedKey:      maskKey(item.Key),
			Status:         item.Status,
			GroupID:        item.GroupID,
			GroupName:      groupName,
			RateMultiplier: rate,
			Quota:          item.Quota,
			QuotaUsed:      item.QuotaUsed,
		})
	}

	outputSubs := make([]OutputSubscription, 0, len(subs))
	for _, s := range subs {
		rate := s.Group.RateMultiplier
		groupName := s.Group.Name
		if g, ok := groupMap[s.GroupID]; ok {
			rate = g.RateMultiplier
			groupName = g.Name
		}

		outputSubs = append(outputSubs, OutputSubscription{
			ID:             s.ID,
			Status:         s.Status,
			GroupID:        s.GroupID,
			GroupName:      groupName,
			RateMultiplier: rate,
			StartsAt:       s.StartsAt,
			ExpiresAt:      s.ExpiresAt,
			Daily:          makeBalanceWindow(s.Group.DailyLimitUSD, s.DailyUsageUSD),
			Weekly:         makeBalanceWindow(s.Group.WeeklyLimitUSD, s.WeeklyUsageUSD),
			Monthly:        makeBalanceWindow(s.Group.MonthlyLimitUSD, s.MonthlyUsageUSD),
		})
	}

	return &Output{
		AccountBalance: me.Balance,
		Account: OutputAccount{
			ID:             me.ID,
			Email:          me.Email,
			Role:           me.Role,
			Status:         me.Status,
			Balance:        me.Balance,
			TotalRecharged: me.TotalRecharged,
			Concurrency:    me.Concurrency,
			RPMLimit:       me.RPMLimit,
		},
		Keys:          outputKeys,
		Groups:        outputGroups,
		Subscriptions: outputSubs,
		FetchedAt:     time.Now().Format(time.RFC3339),
	}, nil
}

func encodeJSON(out any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func loadSiteConfigs() ([]SiteConfig, error) {
	if sites, err := loadIndexedSiteConfigs(); err != nil || len(sites) > 0 {
		return sites, err
	}

	email := strings.TrimSpace(os.Getenv("AI_INPUT_EMAIL"))
	password := strings.TrimSpace(os.Getenv("AI_INPUT_PASSWORD"))
	if email == "" || password == "" {
		return nil, fmt.Errorf("configure AI_INPUT_EMAIL and AI_INPUT_PASSWORD, or use AI_INPUT_SITE_1_URL/EMAIL/PASSWORD")
	}

	baseURL, err := normalizeBaseURL(envOrDefault("AI_INPUT_BASE_URL", defaultBaseURL))
	if err != nil {
		return nil, fmt.Errorf("AI_INPUT_BASE_URL: %w", err)
	}

	return []SiteConfig{{
		Name:     strings.TrimSpace(os.Getenv("AI_INPUT_NAME")),
		BaseURL:  baseURL,
		Email:    email,
		Password: password,
		Legacy:   true,
	}}, nil
}

func loadIndexedSiteConfigs() ([]SiteConfig, error) {
	indexes := siteConfigIndexes()
	if len(indexes) == 0 {
		return nil, nil
	}

	sites := make([]SiteConfig, 0, len(indexes))
	for _, index := range indexes {
		prefix := fmt.Sprintf("AI_INPUT_SITE_%d_", index)
		rawURL := strings.TrimSpace(firstEnv(prefix+"URL", prefix+"BASE_URL", prefix+"WEBSITE"))
		email := strings.TrimSpace(os.Getenv(prefix + "EMAIL"))
		password := strings.TrimSpace(os.Getenv(prefix + "PASSWORD"))
		name := strings.TrimSpace(os.Getenv(prefix + "NAME"))

		if rawURL == "" {
			return nil, fmt.Errorf("%sURL is required", prefix)
		}
		if email == "" {
			return nil, fmt.Errorf("%sEMAIL is required", prefix)
		}
		if password == "" {
			return nil, fmt.Errorf("%sPASSWORD is required", prefix)
		}

		baseURL, err := normalizeBaseURL(rawURL)
		if err != nil {
			return nil, fmt.Errorf("%sURL: %w", prefix, err)
		}
		if name == "" {
			name = fmt.Sprintf("site_%d", index)
		}

		sites = append(sites, SiteConfig{
			Name:     name,
			BaseURL:  baseURL,
			Email:    email,
			Password: password,
		})
	}
	return sites, nil
}

func siteConfigIndexes() []int {
	indexSet := make(map[int]struct{})
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(key, "AI_INPUT_SITE_") {
			continue
		}

		rest := strings.TrimPrefix(key, "AI_INPUT_SITE_")
		indexText, _, found := strings.Cut(rest, "_")
		if !found {
			continue
		}
		index, err := strconv.Atoi(indexText)
		if err != nil || index <= 0 {
			continue
		}
		indexSet[index] = struct{}{}
	}

	indexes := make([]int, 0, len(indexSet))
	for index := range indexSet {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	return indexes
}

func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", fmt.Errorf("missing URL")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("URL must include scheme and host, for example https://ai.input.im")
	}
	return raw, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func loadDotEnv() error {
	for _, path := range dotEnvCandidates() {
		if err := loadDotEnvFile(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		return nil
	}
	return nil
}

func dotEnvCandidates() []string {
	var paths []string
	seen := make(map[string]struct{})
	add := func(path string) {
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err == nil {
			path = abs
		}
		clean := filepath.Clean(path)
		key := strings.ToLower(clean)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		paths = append(paths, clean)
	}

	if explicit := strings.TrimSpace(os.Getenv("AI_INPUT_ENV_FILE")); explicit != "" {
		add(explicit)
	}
	if cwd, err := os.Getwd(); err == nil {
		add(filepath.Join(cwd, ".env"))
		add(filepath.Join(cwd, "shell", ".env"))
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		add(filepath.Join(filepath.Dir(file), ".env"))
	}
	if exe, err := os.Executable(); err == nil {
		add(filepath.Join(filepath.Dir(exe), ".env"))
	}
	return paths
}

func loadDotEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		key, value, ok, err := parseDotEnvLine(scanner.Text())
		if err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNumber, err)
		}
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("%s:%d: set %s: %w", path, lineNumber, key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func parseDotEnvLine(line string) (string, string, bool, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false, nil
	}
	line = strings.TrimSpace(strings.TrimPrefix(line, "export "))

	key, value, found := strings.Cut(line, "=")
	if !found {
		return "", "", false, fmt.Errorf("missing '='")
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false, fmt.Errorf("missing key")
	}
	if strings.ContainsAny(key, " \t\r\n") {
		return "", "", false, fmt.Errorf("invalid key %q", key)
	}

	value, err := parseDotEnvValue(strings.TrimSpace(value))
	if err != nil {
		return "", "", false, err
	}
	return key, value, true, nil
}

func parseDotEnvValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	if strings.HasPrefix(value, `"`) {
		end := closingDoubleQuote(value)
		if end < 0 {
			return "", fmt.Errorf("unterminated quoted value")
		}
		if err := validateDotEnvValueSuffix(value[end+1:]); err != nil {
			return "", err
		}
		unquoted, err := strconv.Unquote(value[:end+1])
		if err != nil {
			return "", err
		}
		return unquoted, nil
	}
	if strings.HasPrefix(value, "'") {
		end := strings.Index(value[1:], "'")
		if end < 0 {
			return "", fmt.Errorf("unterminated quoted value")
		}
		end++
		if err := validateDotEnvValueSuffix(value[end+1:]); err != nil {
			return "", err
		}
		return value[1:end], nil
	}

	if i := strings.Index(value, " #"); i >= 0 {
		value = value[:i]
	}
	return strings.TrimSpace(value), nil
}

func closingDoubleQuote(value string) int {
	escaped := false
	for i := 1; i < len(value); i++ {
		switch {
		case escaped:
			escaped = false
		case value[i] == '\\':
			escaped = true
		case value[i] == '"':
			return i
		}
	}
	return -1
}

func validateDotEnvValueSuffix(suffix string) error {
	suffix = strings.TrimSpace(suffix)
	if suffix == "" || strings.HasPrefix(suffix, "#") {
		return nil
	}
	return fmt.Errorf("unexpected trailing content %q", suffix)
}

func login(client *http.Client, baseURL, email, password string) (string, error) {
	reqBody := LoginRequest{
		Email:    email,
		Password: password,
	}
	var resp APIResponse[LoginData]
	if err := doJSON(client, baseURL, http.MethodPost, "/api/v1/auth/login", "", reqBody, &resp); err != nil {
		return "", err
	}
	return resp.Data.AccessToken, nil
}

func getMe(client *http.Client, baseURL, token string) (*MeData, error) {
	path := "/api/v1/auth/me?timezone=" + url.QueryEscape(timezone)
	var resp APIResponse[MeData]
	if err := doJSON(client, baseURL, http.MethodGet, path, token, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func getKeys(client *http.Client, baseURL, token string) (*KeysData, error) {
	path := "/api/v1/keys?page=1&page_size=100&sort_by=created_at&sort_order=desc&timezone=" + url.QueryEscape(timezone)
	var resp APIResponse[KeysData]
	if err := doJSON(client, baseURL, http.MethodGet, path, token, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func getGroups(client *http.Client, baseURL, token string) ([]Group, error) {
	path := "/api/v1/groups/available?timezone=" + url.QueryEscape(timezone)
	var resp APIResponse[[]Group]
	if err := doJSON(client, baseURL, http.MethodGet, path, token, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func getSubscriptions(client *http.Client, baseURL, token string) ([]SubscriptionItem, error) {
	path := "/api/v1/subscriptions/active?timezone=" + url.QueryEscape(timezone)
	var resp APIResponse[[]SubscriptionItem]
	if err := doJSON(client, baseURL, http.MethodGet, path, token, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func doJSON(client *http.Client, baseURL, method, path, token string, reqBody any, out any) error {
	var bodyReader io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", baseURL)
		req.Header.Set("Referer", baseURL+"/login")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(raw))
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("解析响应失败: %v, body=%s", err, string(raw))
	}

	switch v := out.(type) {
	case *APIResponse[LoginData]:
		if v.Code != 0 {
			return fmt.Errorf("业务错误: code=%d message=%s", v.Code, v.Message)
		}
	case *APIResponse[MeData]:
		if v.Code != 0 {
			return fmt.Errorf("业务错误: code=%d message=%s", v.Code, v.Message)
		}
	case *APIResponse[KeysData]:
		if v.Code != 0 {
			return fmt.Errorf("业务错误: code=%d message=%s", v.Code, v.Message)
		}
	case *APIResponse[[]Group]:
		if v.Code != 0 {
			return fmt.Errorf("业务错误: code=%d message=%s", v.Code, v.Message)
		}
	case *APIResponse[[]SubscriptionItem]:
		if v.Code != 0 {
			return fmt.Errorf("业务错误: code=%d message=%s", v.Code, v.Message)
		}
	}

	return nil
}

func makeBalanceWindow(limit, used float64) BalanceWindow {
	remain := limit - used
	if remain < 0 {
		remain = 0
	}
	return BalanceWindow{
		LimitUSD:  limit,
		UsedUSD:   used,
		RemainUSD: remain,
	}
}

func maskKey(key string) string {
	if len(key) <= 16 {
		return key
	}
	return key[:8] + "..." + key[len(key)-8:]
}

func exitErr(prefix string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", prefix, err)
	os.Exit(1)
}
