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
	"strconv"
	"strings"
)

func encodeJSON(out any, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
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
		return "", fmt.Errorf("URL must include scheme and host")
	}
	return raw, nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
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

	if explicit := strings.TrimSpace(firstEnv("BALANCE_ENV_FILE", "NEWAPI_ENV_FILE", "GGNIAO_ENV_FILE", "AI_INPUT_ENV_FILE")); explicit != "" {
		add(explicit)
	}
	if cwd, err := os.Getwd(); err == nil {
		add(filepath.Join(cwd, ".env"))
		add(filepath.Join(cwd, "shell", ".env"))
		add(filepath.Join(cwd, "..", ".env"))
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		dir := filepath.Dir(file)
		add(filepath.Join(dir, ".env"))
		add(filepath.Join(dir, "..", ".env"))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		add(filepath.Join(dir, ".env"))
		add(filepath.Join(dir, "..", ".env"))
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

	parsed, err := parseDotEnvValue(strings.TrimSpace(value))
	if err != nil {
		return "", "", false, err
	}
	return key, parsed, true, nil
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

func doJSON(client *http.Client, req *http.Request, reqBody any, out any) error {
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		req.Body = io.NopCloser(bytes.NewReader(b))
		req.ContentLength = int64(len(b))
		req.Header.Set("Content-Type", "application/json")
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
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func maskKey(key string) string {
	if len(key) <= 16 {
		return key
	}
	return key[:8] + "..." + key[len(key)-8:]
}

func floatPtr(v float64) *float64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func quotaBalance(limit, used, remain float64, unlimited bool) *balance {
	return &balance{
		Unit:      "quota",
		Limit:     floatPtr(limit),
		Used:      floatPtr(used),
		Remain:    floatPtr(remain),
		Unlimited: unlimited,
	}
}

func newAPIAccountBalance(rawQuota int64) *balance {
	return &balance{
		Unit:   "balance",
		Remain: floatPtr(float64(rawQuota) / 500000),
	}
}

func usdBalance(limit, used float64) *balance {
	remain := limit - used
	if remain < 0 {
		remain = 0
	}
	return &balance{
		Unit:   "usd",
		Limit:  floatPtr(limit),
		Used:   floatPtr(used),
		Remain: floatPtr(remain),
	}
}

func accountUSDBalance(remain float64) *balance {
	return &balance{
		Unit:   "usd",
		Remain: floatPtr(remain),
	}
}

func makeSubscriptionWindow(limit, used float64) *subscriptionWindow {
	remain := limit - used
	if remain < 0 {
		remain = 0
	}
	return &subscriptionWindow{
		Unit:   "usd",
		Limit:  limit,
		Used:   used,
		Remain: remain,
	}
}
