package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvFilePreservesExistingEnvironment(t *testing.T) {
	preserveAndUnsetEnv(t, "TEST_BALANCE_KEEP", "TEST_BALANCE_SET", "TEST_BALANCE_SINGLE", "TEST_BALANCE_UNQUOTED")
	setEnv(t, "TEST_BALANCE_KEEP", "from-env")

	envPath := filepath.Join(t.TempDir(), ".env")
	content := "" +
		"TEST_BALANCE_KEEP=from-file\n" +
		"TEST_BALANCE_SET=\"quoted value\" # comment\n" +
		"export TEST_BALANCE_SINGLE='single value'\n" +
		"TEST_BALANCE_UNQUOTED=value # comment\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := loadDotEnvFile(envPath); err != nil {
		t.Fatal(err)
	}

	assertEnv(t, "TEST_BALANCE_KEEP", "from-env")
	assertEnv(t, "TEST_BALANCE_SET", "quoted value")
	assertEnv(t, "TEST_BALANCE_SINGLE", "single value")
	assertEnv(t, "TEST_BALANCE_UNQUOTED", "value")
}

func TestMakeSubscriptionWindowClampsRemain(t *testing.T) {
	window := makeSubscriptionWindow(10, 12)
	if window.Remain != 0 {
		t.Fatalf("remain = %v, want 0", window.Remain)
	}
}

func assertEnv(t *testing.T, key, want string) {
	t.Helper()
	if got := os.Getenv(key); got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}
