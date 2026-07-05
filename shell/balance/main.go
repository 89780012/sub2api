package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Wei-Shaw/sub2api/pkg/balancefetch"
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

	out, err := balancefetch.FetchSitesContext(context.Background(), toFetcherConfigs(sites))
	if err != nil {
		return err
	}
	return encodeJSON(out, os.Stdout)
}

func fetchConfiguredSite(site siteConfig) (*unifiedSite, error) {
	return balancefetch.FetchConfiguredSiteContext(context.Background(), toFetcherConfig(site))
}

func toFetcherConfigs(sites []siteConfig) []balancefetch.SiteConfig {
	configs := make([]balancefetch.SiteConfig, 0, len(sites))
	for _, site := range sites {
		configs = append(configs, toFetcherConfig(site))
	}
	return configs
}

func toFetcherConfig(site siteConfig) balancefetch.SiteConfig {
	return balancefetch.SiteConfig{
		Platform: balancefetch.Platform(site.Platform),
		Name:     site.Name,
		BaseURL:  site.BaseURL,
		Username: site.Username,
		Email:    site.Email,
		Password: site.Password,
	}
}
