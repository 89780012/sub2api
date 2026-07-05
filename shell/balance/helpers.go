package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/pkg/balancefetch"
)

func encodeJSON(out any, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func normalizeBaseURL(raw string) (string, error) {
	return balancefetch.NormalizeBaseURL(raw)
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
