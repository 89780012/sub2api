package balancefetch

type platform string

const (
	platformNewAPI  platform = "newapi"
	platformSub2API platform = "sub2api"
)

type unifiedOutput struct {
	Sites     []unifiedSite `json:"sites"`
	FetchedAt string        `json:"fetched_at"`
}

type unifiedSite struct {
	Platform      platform              `json:"platform"`
	Name          string                `json:"name,omitempty"`
	BaseURL       string                `json:"base_url"`
	Error         string                `json:"error,omitempty"`
	Account       *unifiedAccount       `json:"account,omitempty"`
	Keys          []unifiedKey          `json:"keys,omitempty"`
	Groups        []unifiedGroup        `json:"groups,omitempty"`
	Subscriptions []unifiedSubscription `json:"subscriptions,omitempty"`
	FetchedAt     string                `json:"fetched_at,omitempty"`
}

type unifiedAccount struct {
	ID             string   `json:"id,omitempty"`
	Login          string   `json:"login,omitempty"`
	DisplayName    string   `json:"display_name,omitempty"`
	GroupName      string   `json:"group_name,omitempty"`
	Role           string   `json:"role,omitempty"`
	Status         string   `json:"status,omitempty"`
	Balance        *balance `json:"balance,omitempty"`
	TotalRecharged *float64 `json:"total_recharged,omitempty"`
	Concurrency    *int     `json:"concurrency,omitempty"`
	RPMLimit       *int     `json:"rpm_limit,omitempty"`
}

type unifiedKey struct {
	ID                  string               `json:"id,omitempty"`
	Name                string               `json:"name,omitempty"`
	MaskedKey           string               `json:"masked_key,omitempty"`
	Status              string               `json:"status,omitempty"`
	GroupID             string               `json:"group_id,omitempty"`
	GroupName           string               `json:"group_name,omitempty"`
	GroupDescription    string               `json:"group_description,omitempty"`
	RateMultiplier      *float64             `json:"rate_multiplier,omitempty"`
	KeyBalance          *balance             `json:"key_balance,omitempty"`
	SubscriptionBalance *subscriptionBalance `json:"subscription_balance,omitempty"`
}

type unifiedGroup struct {
	ID               string   `json:"id,omitempty"`
	Name             string   `json:"name"`
	Platform         string   `json:"platform,omitempty"`
	RateMultiplier   *float64 `json:"rate_multiplier,omitempty"`
	SubscriptionType string   `json:"subscription_type,omitempty"`
	Status           string   `json:"status,omitempty"`
	Description      string   `json:"description,omitempty"`
	DailyLimitUSD    *float64 `json:"daily_limit_usd,omitempty"`
	WeeklyLimitUSD   *float64 `json:"weekly_limit_usd,omitempty"`
	MonthlyLimitUSD  *float64 `json:"monthly_limit_usd,omitempty"`
}

type unifiedSubscription struct {
	ID             string              `json:"id,omitempty"`
	Status         string              `json:"status,omitempty"`
	GroupID        string              `json:"group_id,omitempty"`
	GroupName      string              `json:"group_name,omitempty"`
	RateMultiplier *float64            `json:"rate_multiplier,omitempty"`
	StartsAt       string              `json:"starts_at,omitempty"`
	ExpiresAt      string              `json:"expires_at,omitempty"`
	Daily          *subscriptionWindow `json:"daily,omitempty"`
	Weekly         *subscriptionWindow `json:"weekly,omitempty"`
	Monthly        *subscriptionWindow `json:"monthly,omitempty"`
}

type subscriptionBalance unifiedSubscription

type balance struct {
	Unit      string   `json:"unit"`
	Limit     *float64 `json:"limit,omitempty"`
	Used      *float64 `json:"used,omitempty"`
	Remain    *float64 `json:"remain,omitempty"`
	Unlimited bool     `json:"unlimited,omitempty"`
}

type subscriptionWindow struct {
	Unit   string  `json:"unit"`
	Limit  float64 `json:"limit"`
	Used   float64 `json:"used"`
	Remain float64 `json:"remain"`
}
