package main

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadSiteConfigsReadsUnifiedSitesInOrder(t *testing.T) {
	preserveAndUnsetEnv(t,
		"BALANCE_SITE_1_PLATFORM",
		"BALANCE_SITE_1_NAME",
		"BALANCE_SITE_1_URL",
		"BALANCE_SITE_1_USERNAME",
		"BALANCE_SITE_1_PASSWORD",
		"BALANCE_SITE_2_PLATFORM",
		"BALANCE_SITE_2_URL",
		"BALANCE_SITE_2_EMAIL",
		"BALANCE_SITE_2_PASSWORD",
		"NEWAPI_USERNAME",
		"NEWAPI_PASSWORD",
		"AI_INPUT_EMAIL",
		"AI_INPUT_PASSWORD",
	)

	setEnv(t, "BALANCE_SITE_2_PLATFORM", "sub2api")
	setEnv(t, "BALANCE_SITE_2_URL", "https://two.example.com/")
	setEnv(t, "BALANCE_SITE_2_EMAIL", "two@example.com")
	setEnv(t, "BALANCE_SITE_2_PASSWORD", "two-password")
	setEnv(t, "BALANCE_SITE_1_PLATFORM", "newapi")
	setEnv(t, "BALANCE_SITE_1_NAME", "first")
	setEnv(t, "BALANCE_SITE_1_URL", "https://one.example.com")
	setEnv(t, "BALANCE_SITE_1_USERNAME", "one@example.com")
	setEnv(t, "BALANCE_SITE_1_PASSWORD", "one-password")

	sites, err := loadSiteConfigs()
	if err != nil {
		t.Fatal(err)
	}

	want := []siteConfig{
		{Platform: platformNewAPI, Name: "first", BaseURL: "https://one.example.com", Username: "one@example.com", Email: "one@example.com", Password: "one-password"},
		{Platform: platformSub2API, Name: "site_2", BaseURL: "https://two.example.com", Username: "two@example.com", Email: "two@example.com", Password: "two-password"},
	}
	if !reflect.DeepEqual(sites, want) {
		t.Fatalf("sites = %#v, want %#v", sites, want)
	}
}

func TestLoadSiteConfigsFallsBackToLegacyNewAPIAndSub2API(t *testing.T) {
	preserveAndUnsetEnv(t,
		"BALANCE_SITE_1_PLATFORM",
		"NEWAPI_NAME",
		"NEWAPI_BASE_URL",
		"NEWAPI_USERNAME",
		"NEWAPI_PASSWORD",
		"AI_INPUT_NAME",
		"AI_INPUT_BASE_URL",
		"AI_INPUT_EMAIL",
		"AI_INPUT_PASSWORD",
	)

	setEnv(t, "NEWAPI_NAME", "new")
	setEnv(t, "NEWAPI_BASE_URL", "https://new.example.com/")
	setEnv(t, "NEWAPI_USERNAME", "new@example.com")
	setEnv(t, "NEWAPI_PASSWORD", "new-password")
	setEnv(t, "AI_INPUT_NAME", "sub")
	setEnv(t, "AI_INPUT_BASE_URL", "https://sub.example.com/")
	setEnv(t, "AI_INPUT_EMAIL", "sub@example.com")
	setEnv(t, "AI_INPUT_PASSWORD", "sub-password")

	sites, err := loadSiteConfigs()
	if err != nil {
		t.Fatal(err)
	}

	want := []siteConfig{
		{Platform: platformNewAPI, Name: "new", BaseURL: "https://new.example.com", Username: "new@example.com", Email: "new@example.com", Password: "new-password", Legacy: true},
		{Platform: platformSub2API, Name: "sub", BaseURL: "https://sub.example.com", Username: "sub@example.com", Email: "sub@example.com", Password: "sub-password", Legacy: true},
	}
	if !reflect.DeepEqual(sites, want) {
		t.Fatalf("sites = %#v, want %#v", sites, want)
	}
}

func preserveAndUnsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		oldValue, hadValue := os.LookupEnv(key)
		key := key
		t.Cleanup(func() {
			if hadValue {
				_ = os.Setenv(key, oldValue)
				return
			}
			_ = os.Unsetenv(key)
		})
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatal(err)
	}
}
