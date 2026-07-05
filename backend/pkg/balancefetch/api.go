package balancefetch

import (
	"context"
	"fmt"
	"time"
)

type Platform string

const (
	PlatformNewAPI  Platform = "newapi"
	PlatformSub2API Platform = "sub2api"
)

type SiteConfig struct {
	Platform Platform
	Name     string
	BaseURL  string
	Username string
	Email    string
	Password string
}

type siteConfig struct {
	Platform platform
	Name     string
	BaseURL  string
	Username string
	Email    string
	Password string
}

type Output = unifiedOutput
type Site = unifiedSite
type Account = unifiedAccount
type Key = unifiedKey
type Group = unifiedGroup
type Subscription = unifiedSubscription
type SubscriptionBalance = subscriptionBalance
type Balance = balance
type SubscriptionWindow = subscriptionWindow

func NormalizeBaseURL(raw string) (string, error) {
	return normalizeBaseURL(raw)
}

func FetchConfiguredSite(site SiteConfig) (*Site, error) {
	return FetchConfiguredSiteContext(context.Background(), site)
}

func FetchConfiguredSiteContext(ctx context.Context, site SiteConfig) (*Site, error) {
	internal := siteConfig{
		Platform: platform(site.Platform),
		Name:     site.Name,
		BaseURL:  site.BaseURL,
		Username: site.Username,
		Email:    site.Email,
		Password: site.Password,
	}
	switch internal.Platform {
	case platformNewAPI:
		return fetchNewAPISite(ctx, internal)
	case platformSub2API:
		return fetchSub2APISite(ctx, internal)
	default:
		return nil, fmt.Errorf("unsupported platform %q", site.Platform)
	}
}

func BuildOutput(sites []Site) Output {
	out := Output{
		Sites:     make([]unifiedSite, 0, len(sites)),
		FetchedAt: time.Now().Format(time.RFC3339),
	}
	for i := range sites {
		out.Sites = append(out.Sites, sites[i])
	}
	return out
}

func FetchSites(sites []SiteConfig) (Output, error) {
	return FetchSitesContext(context.Background(), sites)
}

func FetchSitesContext(ctx context.Context, sites []SiteConfig) (Output, error) {
	out := Output{
		Sites:     make([]unifiedSite, 0, len(sites)),
		FetchedAt: time.Now().Format(time.RFC3339),
	}
	multiSite := len(sites) > 1

	for _, site := range sites {
		result, err := FetchConfiguredSiteContext(ctx, site)
		if err != nil {
			if !multiSite {
				return Output{}, fmt.Errorf("fetch %s site failed: %w", site.Platform, err)
			}
			out.Sites = append(out.Sites, unifiedSite{
				Platform: platform(site.Platform),
				Name:     site.Name,
				BaseURL:  site.BaseURL,
				Error:    err.Error(),
			})
			continue
		}
		out.Sites = append(out.Sites, *result)
	}

	return out, nil
}
