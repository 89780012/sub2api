package balancefetch

import "testing"

func TestNormalizeNewAPIMapsGroupRatioAndQuota(t *testing.T) {
	user := &newAPIUserSelfResponse{}
	user.Data.ID = 316
	user.Data.Username = "user@example.com"
	user.Data.Display = "User"
	user.Data.Group = "default"
	user.Data.Quota = 100
	user.Data.UsedQuota = 25

	groups := &newAPIGroupsResponse{Data: map[string]newAPIGroupInfo{
		"plus": {Desc: "plus group", Ratio: 0.08},
	}}
	tokens := &newAPITokenResponse{}
	tokens.Data.Items = append(tokens.Data.Items, struct {
		ID             int    `json:"id"`
		Name           string `json:"name"`
		Key            string `json:"key"`
		Group          string `json:"group"`
		Status         int    `json:"status"`
		UnlimitedQuota bool   `json:"unlimited_quota"`
		RemainQuota    int64  `json:"remain_quota"`
		UsedQuota      int64  `json:"used_quota"`
	}{
		ID: 1, Name: "k", Key: "1234567890abcdef1234", Group: "plus", Status: 1, RemainQuota: 50, UsedQuota: 10,
	})

	out := normalizeNewAPI(siteConfig{Platform: platformNewAPI, Name: "n", BaseURL: "https://example.com"}, user, groups, tokens)
	if out.Account.Balance.Limit != nil {
		t.Fatalf("account limit = %#v, want nil", out.Account.Balance.Limit)
	}
	if out.Account.Balance.Used != nil {
		t.Fatalf("account used = %#v, want nil", out.Account.Balance.Used)
	}
	if out.Account.Balance.Remain == nil || *out.Account.Balance.Remain != 0.0002 {
		t.Fatalf("account remain = %#v, want 0.0002", out.Account.Balance.Remain)
	}
	if len(out.Keys) != 1 {
		t.Fatalf("keys len = %d, want 1", len(out.Keys))
	}
	if out.Keys[0].RateMultiplier == nil || *out.Keys[0].RateMultiplier != 0.08 {
		t.Fatalf("rate = %#v, want 0.08", out.Keys[0].RateMultiplier)
	}
	if out.Keys[0].MaskedKey != "12345678...cdef1234" {
		t.Fatalf("masked key = %q", out.Keys[0].MaskedKey)
	}
}

func TestNormalizeSub2APIAttachesSubscriptionByGroupID(t *testing.T) {
	me := &sub2APIMeData{ID: 7, Email: "user@example.com", Balance: 12.5, Status: "active"}
	keys := &sub2APIKeysData{Items: []sub2APIKeyItem{
		{ID: 1, Name: "with-sub", Key: "sk-1234567890abcdef", GroupID: 3, Status: "active"},
		{ID: 2, Name: "without-sub", Key: "sk-abcdef1234567890", GroupID: 4, Status: "active"},
	}}
	groups := []sub2APIGroup{
		{ID: 3, Name: "plus", RateMultiplier: 0.1, DailyLimitUSD: 10, WeeklyLimitUSD: 50, MonthlyLimitUSD: 100, Status: "active"},
		{ID: 4, Name: "pro", RateMultiplier: 0.2, Status: "active"},
	}
	subs := []sub2APISubscriptionItem{
		{ID: 99, GroupID: 3, Status: "active", DailyUsageUSD: 3, WeeklyUsageUSD: 10, MonthlyUsageUSD: 40},
	}

	out := normalizeSub2API(siteConfig{Platform: platformSub2API, Name: "s", BaseURL: "https://example.com"}, me, keys, groups, subs)
	if len(out.Keys) != 2 {
		t.Fatalf("keys len = %d, want 2", len(out.Keys))
	}
	if out.Keys[0].SubscriptionBalance == nil {
		t.Fatal("first key subscription balance is nil")
	}
	if out.Keys[0].SubscriptionBalance.Daily == nil || out.Keys[0].SubscriptionBalance.Daily.Remain != 7 {
		t.Fatalf("daily remain = %#v, want 7", out.Keys[0].SubscriptionBalance.Daily)
	}
	if out.Keys[1].SubscriptionBalance != nil {
		t.Fatalf("second key subscription balance = %#v, want nil", out.Keys[1].SubscriptionBalance)
	}
	if out.Keys[0].RateMultiplier == nil || *out.Keys[0].RateMultiplier != 0.1 {
		t.Fatalf("rate = %#v, want 0.1", out.Keys[0].RateMultiplier)
	}
}
