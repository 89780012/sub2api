package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchNewAPISiteUsesEndpointFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			writeJSON(t, w, map[string]any{
				"success": true,
				"data":    map[string]any{"id": 42},
			})
		case "/api/user/self":
			if got := r.Header.Get("New-Api-User"); got != "42" {
				t.Fatalf("New-Api-User = %q, want 42", got)
			}
			writeJSON(t, w, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":           42,
					"username":     "new@example.com",
					"display_name": "New",
					"group":        "default",
					"quota":        100,
					"used_quota":   30,
				},
			})
		case "/api/user/self/groups":
			writeJSON(t, w, map[string]any{
				"success": true,
				"data": map[string]any{
					"plus": map[string]any{"desc": "Plus", "ratio": 0.08},
				},
			})
		case "/api/token/":
			writeJSON(t, w, map[string]any{
				"success": true,
				"data": map[string]any{
					"items": []map[string]any{
						{
							"id":              7,
							"name":            "key",
							"key":             "1234567890abcdef1234",
							"group":           "plus",
							"status":          1,
							"unlimited_quota": false,
							"remain_quota":    70,
							"used_quota":      5,
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	out, err := fetchNewAPISite(siteConfig{
		Platform: platformNewAPI,
		Name:     "new",
		BaseURL:  server.URL,
		Username: "new@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Account.Balance.Limit != nil {
		t.Fatalf("account limit = %#v, want nil", out.Account.Balance.Limit)
	}
	if out.Account.Balance.Remain == nil || *out.Account.Balance.Remain != 0.0002 {
		t.Fatalf("account remain = %#v, want 0.0002", out.Account.Balance.Remain)
	}
	if len(out.Keys) != 1 || out.Keys[0].RateMultiplier == nil || *out.Keys[0].RateMultiplier != 0.08 {
		t.Fatalf("keys = %#v, want one key with 0.08 rate", out.Keys)
	}
}

func TestFetchSub2APISiteUsesEndpointFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			writeJSON(t, w, map[string]any{
				"code": 0,
				"data": map[string]any{"access_token": "token"},
			})
		case "/api/v1/auth/me":
			requireBearer(t, r)
			writeJSON(t, w, map[string]any{
				"code": 0,
				"data": map[string]any{
					"id":              99,
					"email":           "sub@example.com",
					"status":          "active",
					"balance":         12.5,
					"total_recharged": 20,
					"concurrency":     10,
					"rpm_limit":       0,
					"role":            "user",
				},
			})
		case "/api/v1/keys":
			requireBearer(t, r)
			writeJSON(t, w, map[string]any{
				"code": 0,
				"data": map[string]any{
					"items": []map[string]any{
						{
							"id":       1,
							"name":     "key",
							"key":      "sk-1234567890abcdef1234",
							"group_id": 3,
							"status":   "active",
						},
					},
				},
			})
		case "/api/v1/groups/available":
			requireBearer(t, r)
			writeJSON(t, w, map[string]any{
				"code": 0,
				"data": []map[string]any{
					{
						"id":                3,
						"name":              "plus",
						"platform":          "openai",
						"rate_multiplier":   0.1,
						"subscription_type": "standard",
						"status":            "active",
						"daily_limit_usd":   10,
						"weekly_limit_usd":  50,
						"monthly_limit_usd": 100,
					},
				},
			})
		case "/api/v1/subscriptions/active":
			requireBearer(t, r)
			writeJSON(t, w, map[string]any{
				"code": 0,
				"data": []map[string]any{
					{
						"id":                77,
						"group_id":          3,
						"status":            "active",
						"daily_usage_usd":   4,
						"weekly_usage_usd":  5,
						"monthly_usage_usd": 20,
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	out, err := fetchSub2APISite(siteConfig{
		Platform: platformSub2API,
		Name:     "sub",
		BaseURL:  server.URL,
		Email:    "sub@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Account.Balance.Remain == nil || *out.Account.Balance.Remain != 12.5 {
		t.Fatalf("account balance = %#v, want 12.5", out.Account.Balance.Remain)
	}
	if len(out.Keys) != 1 || out.Keys[0].SubscriptionBalance == nil {
		t.Fatalf("keys = %#v, want one key with subscription", out.Keys)
	}
	if out.Keys[0].SubscriptionBalance.Daily.Remain != 6 {
		t.Fatalf("daily remain = %v, want 6", out.Keys[0].SubscriptionBalance.Daily.Remain)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}

func requireBearer(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "Bearer token" {
		t.Fatalf("Authorization = %q, want Bearer token", got)
	}
}
