package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultNewAPIBaseURL = "https://ggniao.com"

type newAPISiteConfig struct {
	Name     string
	BaseURL  string
	Username string
	Password string
	Legacy   bool
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		ID int `json:"id"`
	} `json:"data"`
}

type userSelfResponse struct {
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

type tokenResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
		Total    int `json:"total"`
		Items    []struct {
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

type groupInfo struct {
	Desc  string  `json:"desc"`
	Ratio float64 `json:"ratio"`
}

type groupsResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message"`
	Data    map[string]groupInfo `json:"data"`
}

type output struct {
	User     outputUser           `json:"user"`
	Keys     []outputKey          `json:"keys"`
	GroupMap map[string]groupInfo `json:"group_map"`
}

type multiNewAPIOutput struct {
	Sites     []newAPISiteOutput `json:"sites"`
	FetchedAt string             `json:"fetched_at"`
}

type newAPISiteOutput struct {
	Name    string  `json:"name,omitempty"`
	BaseURL string  `json:"base_url"`
	Error   string  `json:"error,omitempty"`
	Data    *output `json:"data,omitempty"`
}

type outputUser struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Display   string `json:"display_name"`
	Group     string `json:"group"`
	Quota     int64  `json:"quota"`
	UsedQuota int64  `json:"used_quota"`
	LeftQuota int64  `json:"left_quota"`
}

type outputKey struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	Key            string  `json:"key"`
	Group          string  `json:"group"`
	GroupRatio     float64 `json:"group_ratio"`
	GroupDesc      string  `json:"group_desc"`
	Status         int     `json:"status"`
	UnlimitedQuota bool    `json:"unlimited_quota"`
	RemainQuota    int64   `json:"remain_quota"`
	UsedQuota      int64   `json:"used_quota"`
}

func main() {
	if err := loadDotEnv(); err != nil {
		exitf("load .env failed: %v", err)
	}

	sites, err := loadNewAPISiteConfigs()
	if err != nil {
		exitf("load site config failed: %v", err)
	}

	if len(sites) == 1 && sites[0].Legacy {
		result, err := fetchNewAPISite(sites[0])
		if err != nil {
			exitf("fetch site failed: %v", err)
		}
		if err := encodeJSON(result); err != nil {
			exitf("encode output failed: %v", err)
		}
		return
	}

	result := multiNewAPIOutput{
		Sites:     make([]newAPISiteOutput, 0, len(sites)),
		FetchedAt: time.Now().Format(time.RFC3339),
	}
	for _, site := range sites {
		siteResult := newAPISiteOutput{
			Name:    site.Name,
			BaseURL: site.BaseURL,
		}
		data, err := fetchNewAPISite(site)
		if err != nil {
			siteResult.Error = err.Error()
		} else {
			siteResult.Data = data
		}
		result.Sites = append(result.Sites, siteResult)
	}

	if err := encodeJSON(result); err != nil {
		exitf("encode output failed: %v", err)
	}
}

func fetchNewAPISite(site newAPISiteConfig) (*output, error) {
	client, err := newHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("init http client: %w", err)
	}

	userID, err := login(client, site.BaseURL, site.Username, site.Password)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	userSelf, err := getUserSelf(client, site.BaseURL, userID)
	if err != nil {
		return nil, fmt.Errorf("get user self: %w", err)
	}

	groups, err := getGroups(client, site.BaseURL, userID)
	if err != nil {
		return nil, fmt.Errorf("get groups: %w", err)
	}

	tokens, err := getTokens(client, site.BaseURL, userID)
	if err != nil {
		return nil, fmt.Errorf("get tokens: %w", err)
	}

	result := buildOutput(userSelf, groups, tokens)
	return &result, nil
}

func encodeJSON(result any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(result)
}

func loadNewAPISiteConfigs() ([]newAPISiteConfig, error) {
	if sites, err := loadIndexedNewAPISiteConfigs("NEWAPI_SITE_"); err != nil || len(sites) > 0 {
		return sites, err
	}
	if sites, err := loadIndexedNewAPISiteConfigs("GGNIAO_SITE_"); err != nil || len(sites) > 0 {
		return sites, err
	}

	username := strings.TrimSpace(firstEnv("NEWAPI_USERNAME", "GGNIAO_USERNAME"))
	password := strings.TrimSpace(firstEnv("NEWAPI_PASSWORD", "GGNIAO_PASSWORD"))
	if username == "" || password == "" {
		return nil, fmt.Errorf("configure NEWAPI_USERNAME and NEWAPI_PASSWORD, or use NEWAPI_SITE_1_URL/USERNAME/PASSWORD")
	}

	baseURL, err := normalizeBaseURL(envOrDefault("NEWAPI_BASE_URL", envOrDefault("GGNIAO_BASE_URL", defaultNewAPIBaseURL)))
	if err != nil {
		return nil, fmt.Errorf("NEWAPI_BASE_URL: %w", err)
	}

	return []newAPISiteConfig{{
		Name:     strings.TrimSpace(firstEnv("NEWAPI_NAME", "GGNIAO_NAME")),
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		Legacy:   true,
	}}, nil
}

func loadIndexedNewAPISiteConfigs(envPrefix string) ([]newAPISiteConfig, error) {
	indexes := newAPISiteConfigIndexes(envPrefix)
	if len(indexes) == 0 {
		return nil, nil
	}

	sites := make([]newAPISiteConfig, 0, len(indexes))
	for _, index := range indexes {
		prefix := fmt.Sprintf("%s%d_", envPrefix, index)
		rawURL := strings.TrimSpace(firstEnv(prefix+"URL", prefix+"BASE_URL", prefix+"WEBSITE"))
		username := strings.TrimSpace(firstEnv(prefix+"USERNAME", prefix+"EMAIL"))
		password := strings.TrimSpace(os.Getenv(prefix + "PASSWORD"))
		name := strings.TrimSpace(os.Getenv(prefix + "NAME"))

		if rawURL == "" {
			return nil, fmt.Errorf("%sURL is required", prefix)
		}
		if username == "" {
			return nil, fmt.Errorf("%sUSERNAME is required", prefix)
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

		sites = append(sites, newAPISiteConfig{
			Name:     name,
			BaseURL:  baseURL,
			Username: username,
			Password: password,
		})
	}
	return sites, nil
}

func newAPISiteConfigIndexes(envPrefix string) []int {
	indexSet := make(map[int]struct{})
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(key, envPrefix) {
			continue
		}

		rest := strings.TrimPrefix(key, envPrefix)
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
		return "", fmt.Errorf("URL must include scheme and host, for example https://ggniao.com")
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

	if explicit := strings.TrimSpace(firstEnv("NEWAPI_ENV_FILE", "GGNIAO_ENV_FILE")); explicit != "" {
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

func newHTTPClient() (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Jar:     jar,
		Timeout: 20 * time.Second,
	}, nil
}

func login(client *http.Client, baseURL, username, password string) (int, error) {
	reqBody := loginRequest{
		Username: username,
		Password: password,
	}

	var resp loginResponse
	if err := doJSON(client, http.MethodPost, baseURL+"/api/user/login?turnstile=", 0, reqBody, &resp); err != nil {
		return 0, err
	}
	if !resp.Success {
		return 0, fmt.Errorf("%s", resp.Message)
	}
	return resp.Data.ID, nil
}

func getUserSelf(client *http.Client, baseURL string, userID int) (*userSelfResponse, error) {
	var resp userSelfResponse
	if err := doJSON(client, http.MethodGet, baseURL+"/api/user/self", userID, nil, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("%s", resp.Message)
	}
	return &resp, nil
}

func getGroups(client *http.Client, baseURL string, userID int) (*groupsResponse, error) {
	var resp groupsResponse
	if err := doJSON(client, http.MethodGet, baseURL+"/api/user/self/groups", userID, nil, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("%s", resp.Message)
	}
	return &resp, nil
}

func getTokens(client *http.Client, baseURL string, userID int) (*tokenResponse, error) {
	var resp tokenResponse
	url := baseURL + "/api/token/?p=1&size=20"
	if err := doJSON(client, http.MethodGet, url, userID, nil, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("%s", resp.Message)
	}
	return &resp, nil
}

func doJSON(client *http.Client, method, url string, userID int, reqBody any, out any) error {
	var body io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Cache-Control", "no-store")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if userID > 0 {
		req.Header.Set("New-Api-User", fmt.Sprintf("%d", userID))
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func buildOutput(userSelf *userSelfResponse, groups *groupsResponse, tokens *tokenResponse) output {
	out := output{
		User: outputUser{
			ID:        userSelf.Data.ID,
			Username:  userSelf.Data.Username,
			Display:   userSelf.Data.Display,
			Group:     userSelf.Data.Group,
			Quota:     userSelf.Data.Quota,
			UsedQuota: userSelf.Data.UsedQuota,
			LeftQuota: userSelf.Data.Quota - userSelf.Data.UsedQuota,
		},
		GroupMap: groups.Data,
		Keys:     make([]outputKey, 0, len(tokens.Data.Items)),
	}

	for _, item := range tokens.Data.Items {
		g := groups.Data[item.Group]
		out.Keys = append(out.Keys, outputKey{
			ID:             item.ID,
			Name:           item.Name,
			Key:            item.Key,
			Group:          item.Group,
			GroupRatio:     g.Ratio,
			GroupDesc:      g.Desc,
			Status:         item.Status,
			UnlimitedQuota: item.UnlimitedQuota,
			RemainQuota:    item.RemainQuota,
			UsedQuota:      item.UsedQuota,
		})
	}

	return out
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
