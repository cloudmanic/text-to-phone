// =============================================================================
// File: config_test.go
// Author: Spicer Matthews <spicer@cloudmanic.com>
// Created: 2026-10-24
// Copyright: 2026 Cloudmanic, LLC. All rights reserved.
// =============================================================================

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// configKeys lists every variable LoadConfig reads, so tests can clear them.
var configKeys = []string{
	"PORT", "ACCESS_TOKEN", "TWILIO_ACCOUNT_SID", "TWILIO_AUTH_TOKEN",
	"TWILIO_FROM_NUMBER", "CALL_TO_NUMBER", "TWILIO_VOICE",
}

// clearConfigEnv blanks every config variable for the test. godotenv skips
// keys that are already set, even to "", so each one is unset after t.Setenv
// registers the restore.
func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, k := range configKeys {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

// writeEnvFile writes contents to a .env file in a temp dir and returns its path.
func writeEnvFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestLoadConfigFromFile checks that values come from .env and defaults fill gaps.
func TestLoadConfigFromFile(t *testing.T) {
	clearConfigEnv(t)
	path := writeEnvFile(t, strings.Join([]string{
		"ACCESS_TOKEN=secret",
		"TWILIO_ACCOUNT_SID=AC123",
		"TWILIO_AUTH_TOKEN=auth",
		"TWILIO_FROM_NUMBER=+15550001111",
		"CALL_TO_NUMBER=+15550002222",
	}, "\n"))

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.AccessToken != "secret" || cfg.TwilioAccountSID != "AC123" || cfg.TwilioAuthToken != "auth" {
		t.Errorf("unexpected config: %+v", cfg)
	}
	if cfg.TwilioFromNumber != "+15550001111" || cfg.CallToNumber != "+15550002222" {
		t.Errorf("unexpected numbers: %+v", cfg)
	}
	if cfg.Port != defaultPort {
		t.Errorf("Port = %q, want %q", cfg.Port, defaultPort)
	}
	if cfg.TwilioVoice != defaultVoice {
		t.Errorf("TwilioVoice = %q, want %q", cfg.TwilioVoice, defaultVoice)
	}
}

// TestLoadConfigEnvWinsOverFile checks that real environment variables override .env.
func TestLoadConfigEnvWinsOverFile(t *testing.T) {
	clearConfigEnv(t)
	path := writeEnvFile(t, "ACCESS_TOKEN=from-file\nTWILIO_ACCOUNT_SID=AC1\nTWILIO_AUTH_TOKEN=a\nTWILIO_FROM_NUMBER=+1\nCALL_TO_NUMBER=+2\nPORT=9000\n")
	t.Setenv("ACCESS_TOKEN", "from-env")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.AccessToken != "from-env" {
		t.Errorf("AccessToken = %q, want from-env", cfg.AccessToken)
	}
	if cfg.Port != "9000" {
		t.Errorf("Port = %q, want 9000", cfg.Port)
	}
}

// TestLoadConfigMissingFileUsesEnv checks that a missing .env is not an error.
func TestLoadConfigMissingFileUsesEnv(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("ACCESS_TOKEN", "x")
	t.Setenv("TWILIO_ACCOUNT_SID", "AC1")
	t.Setenv("TWILIO_AUTH_TOKEN", "a")
	t.Setenv("TWILIO_FROM_NUMBER", "+1")
	t.Setenv("CALL_TO_NUMBER", "+2")

	if _, err := LoadConfig(filepath.Join(t.TempDir(), "nope.env")); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
}

// TestLoadConfigReportsAllMissing checks that every missing setting is named.
func TestLoadConfigReportsAllMissing(t *testing.T) {
	clearConfigEnv(t)
	path := writeEnvFile(t, "ACCESS_TOKEN=secret\n")

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, name := range []string{"TWILIO_ACCOUNT_SID", "TWILIO_AUTH_TOKEN", "TWILIO_FROM_NUMBER", "CALL_TO_NUMBER"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not mention %s", err, name)
		}
	}
	if strings.Contains(err.Error(), "ACCESS_TOKEN") {
		t.Errorf("error %q names ACCESS_TOKEN, which was set", err)
	}
}

// TestEnvOr checks the fallback for blank and set variables.
func TestEnvOr(t *testing.T) {
	t.Setenv("TTP_TEST_VALUE", "  ")
	if got := envOr("TTP_TEST_VALUE", "fallback"); got != "fallback" {
		t.Errorf("blank: got %q", got)
	}
	t.Setenv("TTP_TEST_VALUE", "set")
	if got := envOr("TTP_TEST_VALUE", "fallback"); got != "set" {
		t.Errorf("set: got %q", got)
	}
}
