package main

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadNewAPISiteConfigsReadsIndexedSitesInOrder(t *testing.T) {
	keys := []string{
		"NEWAPI_USERNAME",
		"NEWAPI_PASSWORD",
		"GGNIAO_USERNAME",
		"GGNIAO_PASSWORD",
		"NEWAPI_SITE_1_NAME",
		"NEWAPI_SITE_1_URL",
		"NEWAPI_SITE_1_USERNAME",
		"NEWAPI_SITE_1_PASSWORD",
		"NEWAPI_SITE_2_URL",
		"NEWAPI_SITE_2_USERNAME",
		"NEWAPI_SITE_2_PASSWORD",
	}
	newAPIPreserveAndUnsetEnv(t, keys...)

	newAPISetEnv(t, "NEWAPI_USERNAME", "legacy@example.com")
	newAPISetEnv(t, "NEWAPI_PASSWORD", "legacy-password")
	newAPISetEnv(t, "NEWAPI_SITE_2_URL", "https://two.example.com/")
	newAPISetEnv(t, "NEWAPI_SITE_2_USERNAME", "two@example.com")
	newAPISetEnv(t, "NEWAPI_SITE_2_PASSWORD", "two-password")
	newAPISetEnv(t, "NEWAPI_SITE_1_NAME", "first")
	newAPISetEnv(t, "NEWAPI_SITE_1_URL", "https://one.example.com")
	newAPISetEnv(t, "NEWAPI_SITE_1_USERNAME", "one@example.com")
	newAPISetEnv(t, "NEWAPI_SITE_1_PASSWORD", "one-password")

	sites, err := loadNewAPISiteConfigs()
	if err != nil {
		t.Fatal(err)
	}

	want := []newAPISiteConfig{
		{Name: "first", BaseURL: "https://one.example.com", Username: "one@example.com", Password: "one-password"},
		{Name: "site_2", BaseURL: "https://two.example.com", Username: "two@example.com", Password: "two-password"},
	}
	if !reflect.DeepEqual(sites, want) {
		t.Fatalf("sites = %#v, want %#v", sites, want)
	}
}

func TestLoadNewAPISiteConfigsFallsBackToLegacyGGNIAO(t *testing.T) {
	keys := []string{
		"NEWAPI_NAME",
		"NEWAPI_BASE_URL",
		"NEWAPI_USERNAME",
		"NEWAPI_PASSWORD",
		"NEWAPI_SITE_1_URL",
		"NEWAPI_SITE_1_USERNAME",
		"NEWAPI_SITE_1_PASSWORD",
		"GGNIAO_NAME",
		"GGNIAO_BASE_URL",
		"GGNIAO_USERNAME",
		"GGNIAO_PASSWORD",
		"GGNIAO_SITE_1_URL",
		"GGNIAO_SITE_1_USERNAME",
		"GGNIAO_SITE_1_PASSWORD",
	}
	newAPIPreserveAndUnsetEnv(t, keys...)

	newAPISetEnv(t, "GGNIAO_NAME", "legacy")
	newAPISetEnv(t, "GGNIAO_BASE_URL", "https://legacy.example.com/")
	newAPISetEnv(t, "GGNIAO_USERNAME", "legacy@example.com")
	newAPISetEnv(t, "GGNIAO_PASSWORD", "legacy-password")

	sites, err := loadNewAPISiteConfigs()
	if err != nil {
		t.Fatal(err)
	}

	want := []newAPISiteConfig{{
		Name:     "legacy",
		BaseURL:  "https://legacy.example.com",
		Username: "legacy@example.com",
		Password: "legacy-password",
		Legacy:   true,
	}}
	if !reflect.DeepEqual(sites, want) {
		t.Fatalf("sites = %#v, want %#v", sites, want)
	}
}

func newAPIPreserveAndUnsetEnv(t *testing.T, keys ...string) {
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

func newAPISetEnv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatal(err)
	}
}
