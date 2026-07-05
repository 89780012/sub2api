package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultNewAPIBaseURL  = "https://ggniao.com"
	defaultSub2APIBaseURL = "https://ai.input.im"
)

type siteConfig struct {
	Platform platform
	Name     string
	BaseURL  string
	Username string
	Email    string
	Password string
	Legacy   bool
}

func loadSiteConfigs() ([]siteConfig, error) {
	if sites, err := loadUnifiedSiteConfigs(); err != nil || len(sites) > 0 {
		return sites, err
	}

	var sites []siteConfig
	newAPISites, err := loadLegacyNewAPISiteConfigs()
	if err != nil {
		return nil, err
	}
	sites = append(sites, newAPISites...)

	sub2APISites, err := loadLegacySub2APISiteConfigs()
	if err != nil {
		return nil, err
	}
	sites = append(sites, sub2APISites...)

	if len(sites) == 0 {
		return nil, fmt.Errorf("configure BALANCE_SITE_1_PLATFORM/URL/PASSWORD, or legacy NEWAPI_* / AI_INPUT_* variables")
	}
	return sites, nil
}

func loadUnifiedSiteConfigs() ([]siteConfig, error) {
	indexes := configIndexes("BALANCE_SITE_")
	if len(indexes) == 0 {
		return nil, nil
	}

	sites := make([]siteConfig, 0, len(indexes))
	for _, index := range indexes {
		prefix := fmt.Sprintf("BALANCE_SITE_%d_", index)
		platformValue := strings.ToLower(strings.TrimSpace(os.Getenv(prefix + "PLATFORM")))
		if platformValue == "" {
			return nil, fmt.Errorf("%sPLATFORM is required", prefix)
		}

		sitePlatform := platform(platformValue)
		if sitePlatform != platformNewAPI && sitePlatform != platformSub2API {
			return nil, fmt.Errorf("%sPLATFORM must be newapi or sub2api", prefix)
		}

		rawURL := strings.TrimSpace(firstEnv(prefix+"URL", prefix+"BASE_URL", prefix+"WEBSITE"))
		if rawURL == "" {
			return nil, fmt.Errorf("%sURL is required", prefix)
		}
		baseURL, err := normalizeBaseURL(rawURL)
		if err != nil {
			return nil, fmt.Errorf("%sURL: %w", prefix, err)
		}

		password := strings.TrimSpace(os.Getenv(prefix + "PASSWORD"))
		if password == "" {
			return nil, fmt.Errorf("%sPASSWORD is required", prefix)
		}

		username := strings.TrimSpace(firstEnv(prefix+"USERNAME", prefix+"EMAIL"))
		email := strings.TrimSpace(firstEnv(prefix+"EMAIL", prefix+"USERNAME"))
		if sitePlatform == platformNewAPI && username == "" {
			return nil, fmt.Errorf("%sUSERNAME is required for newapi", prefix)
		}
		if sitePlatform == platformSub2API && email == "" {
			return nil, fmt.Errorf("%sEMAIL is required for sub2api", prefix)
		}

		name := strings.TrimSpace(os.Getenv(prefix + "NAME"))
		if name == "" {
			name = fmt.Sprintf("site_%d", index)
		}

		sites = append(sites, siteConfig{
			Platform: sitePlatform,
			Name:     name,
			BaseURL:  baseURL,
			Username: username,
			Email:    email,
			Password: password,
		})
	}
	return sites, nil
}

func loadLegacyNewAPISiteConfigs() ([]siteConfig, error) {
	if sites, err := loadIndexedNewAPISiteConfigs("NEWAPI_SITE_"); err != nil || len(sites) > 0 {
		return sites, err
	}
	if sites, err := loadIndexedNewAPISiteConfigs("GGNIAO_SITE_"); err != nil || len(sites) > 0 {
		return sites, err
	}

	username := strings.TrimSpace(firstEnv("NEWAPI_USERNAME", "GGNIAO_USERNAME"))
	password := strings.TrimSpace(firstEnv("NEWAPI_PASSWORD", "GGNIAO_PASSWORD"))
	if username == "" && password == "" {
		return nil, nil
	}
	if username == "" || password == "" {
		return nil, fmt.Errorf("configure both NEWAPI_USERNAME and NEWAPI_PASSWORD")
	}

	baseURL, err := normalizeBaseURL(envOrDefault("NEWAPI_BASE_URL", envOrDefault("GGNIAO_BASE_URL", defaultNewAPIBaseURL)))
	if err != nil {
		return nil, fmt.Errorf("NEWAPI_BASE_URL: %w", err)
	}

	return []siteConfig{{
		Platform: platformNewAPI,
		Name:     strings.TrimSpace(firstEnv("NEWAPI_NAME", "GGNIAO_NAME")),
		BaseURL:  baseURL,
		Username: username,
		Email:    username,
		Password: password,
		Legacy:   true,
	}}, nil
}

func loadIndexedNewAPISiteConfigs(envPrefix string) ([]siteConfig, error) {
	indexes := configIndexes(envPrefix)
	if len(indexes) == 0 {
		return nil, nil
	}

	sites := make([]siteConfig, 0, len(indexes))
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
			name = fmt.Sprintf("newapi_%d", index)
		}

		sites = append(sites, siteConfig{
			Platform: platformNewAPI,
			Name:     name,
			BaseURL:  baseURL,
			Username: username,
			Email:    username,
			Password: password,
		})
	}
	return sites, nil
}

func loadLegacySub2APISiteConfigs() ([]siteConfig, error) {
	if sites, err := loadIndexedSub2APISiteConfigs(); err != nil || len(sites) > 0 {
		return sites, err
	}

	email := strings.TrimSpace(os.Getenv("AI_INPUT_EMAIL"))
	password := strings.TrimSpace(os.Getenv("AI_INPUT_PASSWORD"))
	if email == "" && password == "" {
		return nil, nil
	}
	if email == "" || password == "" {
		return nil, fmt.Errorf("configure both AI_INPUT_EMAIL and AI_INPUT_PASSWORD")
	}

	baseURL, err := normalizeBaseURL(envOrDefault("AI_INPUT_BASE_URL", defaultSub2APIBaseURL))
	if err != nil {
		return nil, fmt.Errorf("AI_INPUT_BASE_URL: %w", err)
	}

	return []siteConfig{{
		Platform: platformSub2API,
		Name:     strings.TrimSpace(os.Getenv("AI_INPUT_NAME")),
		BaseURL:  baseURL,
		Email:    email,
		Username: email,
		Password: password,
		Legacy:   true,
	}}, nil
}

func loadIndexedSub2APISiteConfigs() ([]siteConfig, error) {
	indexes := configIndexes("AI_INPUT_SITE_")
	if len(indexes) == 0 {
		return nil, nil
	}

	sites := make([]siteConfig, 0, len(indexes))
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
			name = fmt.Sprintf("sub2api_%d", index)
		}

		sites = append(sites, siteConfig{
			Platform: platformSub2API,
			Name:     name,
			BaseURL:  baseURL,
			Email:    email,
			Username: email,
			Password: password,
		})
	}
	return sites, nil
}

func configIndexes(envPrefix string) []int {
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
