package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if err := loadDotEnv(); err != nil {
		return fmt.Errorf("load .env failed: %w", err)
	}

	sites, err := loadSiteConfigs()
	if err != nil {
		return fmt.Errorf("load site config failed: %w", err)
	}

	out := unifiedOutput{
		Sites:     make([]unifiedSite, 0, len(sites)),
		FetchedAt: time.Now().Format(time.RFC3339),
	}
	multiSite := len(sites) > 1

	for _, site := range sites {
		result, err := fetchConfiguredSite(site)
		if err != nil {
			if !multiSite {
				return fmt.Errorf("fetch %s site failed: %w", site.Platform, err)
			}
			out.Sites = append(out.Sites, unifiedSite{
				Platform: site.Platform,
				Name:     site.Name,
				BaseURL:  site.BaseURL,
				Error:    err.Error(),
			})
			continue
		}
		out.Sites = append(out.Sites, *result)
	}

	return encodeJSON(out, os.Stdout)
}

func fetchConfiguredSite(site siteConfig) (*unifiedSite, error) {
	switch site.Platform {
	case platformNewAPI:
		return fetchNewAPISite(site)
	case platformSub2API:
		return fetchSub2APISite(site)
	default:
		return nil, fmt.Errorf("unsupported platform %q", site.Platform)
	}
}
