package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDotEnvFilePreservesExistingEnvironment(t *testing.T) {
	preserveEnv(t, "TEST_DOTENV_KEEP")
	preserveEnv(t, "TEST_DOTENV_SET")
	preserveEnv(t, "TEST_DOTENV_SINGLE")
	preserveEnv(t, "TEST_DOTENV_UNQUOTED")

	requireSetEnv(t, "TEST_DOTENV_KEEP", "from-env")
	requireUnsetEnv(t, "TEST_DOTENV_SET")
	requireUnsetEnv(t, "TEST_DOTENV_SINGLE")
	requireUnsetEnv(t, "TEST_DOTENV_UNQUOTED")

	envPath := filepath.Join(t.TempDir(), ".env")
	content := "" +
		"TEST_DOTENV_KEEP=from-file\n" +
		"TEST_DOTENV_SET=\"quoted value\" # comment\n" +
		"export TEST_DOTENV_SINGLE='single value'\n" +
		"TEST_DOTENV_UNQUOTED=value # comment\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := loadDotEnvFile(envPath); err != nil {
		t.Fatal(err)
	}

	assertEnv(t, "TEST_DOTENV_KEEP", "from-env")
	assertEnv(t, "TEST_DOTENV_SET", "quoted value")
	assertEnv(t, "TEST_DOTENV_SINGLE", "single value")
	assertEnv(t, "TEST_DOTENV_UNQUOTED", "value")
}

func TestLoadSiteConfigsReadsIndexedSitesInOrder(t *testing.T) {
	keys := []string{
		"AI_INPUT_EMAIL",
		"AI_INPUT_PASSWORD",
		"AI_INPUT_SITE_1_NAME",
		"AI_INPUT_SITE_1_URL",
		"AI_INPUT_SITE_1_EMAIL",
		"AI_INPUT_SITE_1_PASSWORD",
		"AI_INPUT_SITE_2_URL",
		"AI_INPUT_SITE_2_EMAIL",
		"AI_INPUT_SITE_2_PASSWORD",
	}
	for _, key := range keys {
		preserveEnv(t, key)
		requireUnsetEnv(t, key)
	}

	requireSetEnv(t, "AI_INPUT_EMAIL", "legacy@example.com")
	requireSetEnv(t, "AI_INPUT_PASSWORD", "legacy-password")
	requireSetEnv(t, "AI_INPUT_SITE_2_URL", "https://two.example.com/")
	requireSetEnv(t, "AI_INPUT_SITE_2_EMAIL", "two@example.com")
	requireSetEnv(t, "AI_INPUT_SITE_2_PASSWORD", "two-password")
	requireSetEnv(t, "AI_INPUT_SITE_1_NAME", "first")
	requireSetEnv(t, "AI_INPUT_SITE_1_URL", "https://one.example.com")
	requireSetEnv(t, "AI_INPUT_SITE_1_EMAIL", "one@example.com")
	requireSetEnv(t, "AI_INPUT_SITE_1_PASSWORD", "one-password")

	sites, err := loadSiteConfigs()
	if err != nil {
		t.Fatal(err)
	}

	want := []SiteConfig{
		{Name: "first", BaseURL: "https://one.example.com", Email: "one@example.com", Password: "one-password"},
		{Name: "site_2", BaseURL: "https://two.example.com", Email: "two@example.com", Password: "two-password"},
	}
	if !reflect.DeepEqual(sites, want) {
		t.Fatalf("sites = %#v, want %#v", sites, want)
	}
}

func TestLoadSiteConfigsFallsBackToLegacySite(t *testing.T) {
	keys := []string{
		"AI_INPUT_NAME",
		"AI_INPUT_BASE_URL",
		"AI_INPUT_EMAIL",
		"AI_INPUT_PASSWORD",
		"AI_INPUT_SITE_1_URL",
		"AI_INPUT_SITE_1_EMAIL",
		"AI_INPUT_SITE_1_PASSWORD",
	}
	for _, key := range keys {
		preserveEnv(t, key)
		requireUnsetEnv(t, key)
	}

	requireSetEnv(t, "AI_INPUT_NAME", "legacy")
	requireSetEnv(t, "AI_INPUT_BASE_URL", "https://legacy.example.com/")
	requireSetEnv(t, "AI_INPUT_EMAIL", "legacy@example.com")
	requireSetEnv(t, "AI_INPUT_PASSWORD", "legacy-password")

	sites, err := loadSiteConfigs()
	if err != nil {
		t.Fatal(err)
	}

	want := []SiteConfig{{
		Name:     "legacy",
		BaseURL:  "https://legacy.example.com",
		Email:    "legacy@example.com",
		Password: "legacy-password",
		Legacy:   true,
	}}
	if !reflect.DeepEqual(sites, want) {
		t.Fatalf("sites = %#v, want %#v", sites, want)
	}
}

func preserveEnv(t *testing.T, key string) {
	t.Helper()
	oldValue, hadValue := os.LookupEnv(key)
	t.Cleanup(func() {
		if hadValue {
			_ = os.Setenv(key, oldValue)
			return
		}
		_ = os.Unsetenv(key)
	})
}

func requireSetEnv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatal(err)
	}
}

func requireUnsetEnv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
}

func assertEnv(t *testing.T, key, want string) {
	t.Helper()
	if got := os.Getenv(key); got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}
