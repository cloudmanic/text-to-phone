// =============================================================================
// File: config.go
// Author: Spicer Matthews <spicer@cloudmanic.com>
// Created: 2026-10-24
// Copyright: 2026 Cloudmanic, LLC. All rights reserved.
// =============================================================================

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Default values for the settings that are safe to leave out of .env.
const (
	defaultPort  = "8080"
	defaultVoice = "Polly.Joanna"
)

// Config holds every setting the app reads from the environment.
type Config struct {
	Port             string
	AccessToken      string
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string
	CallToNumber     string
	TwilioVoice      string
}

// LoadConfig reads envFile (if it exists) and then the process environment.
// Variables already set in the environment win over the file, so a deploy can
// override any value without editing .env.
func LoadConfig(envFile string) (Config, error) {
	if err := godotenv.Load(envFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("read %s: %w", envFile, err)
	}

	cfg := Config{
		Port:             envOr("PORT", defaultPort),
		AccessToken:      os.Getenv("ACCESS_TOKEN"),
		TwilioAccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioFromNumber: os.Getenv("TWILIO_FROM_NUMBER"),
		CallToNumber:     os.Getenv("CALL_TO_NUMBER"),
		TwilioVoice:      envOr("TWILIO_VOICE", defaultVoice),
	}

	return cfg, cfg.validate()
}

// validate reports every missing required setting at once, so a bad .env is
// fixed in one pass instead of one restart per missing value.
func (c Config) validate() error {
	required := []struct {
		name  string
		value string
	}{
		{"ACCESS_TOKEN", c.AccessToken},
		{"TWILIO_ACCOUNT_SID", c.TwilioAccountSID},
		{"TWILIO_AUTH_TOKEN", c.TwilioAuthToken},
		{"TWILIO_FROM_NUMBER", c.TwilioFromNumber},
		{"CALL_TO_NUMBER", c.CallToNumber},
	}

	var missing []string
	for _, r := range required {
		if strings.TrimSpace(r.value) == "" {
			missing = append(missing, r.name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required settings: %s", strings.Join(missing, ", "))
	}
	return nil
}

// envOr returns the named environment variable, or fallback when it is empty.
func envOr(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}
