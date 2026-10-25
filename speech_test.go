// =============================================================================
// File: speech_test.go
// Author: Spicer Matthews <spicer@cloudmanic.com>
// Created: 2026-10-24
// Copyright: 2026 Cloudmanic, LLC. All rights reserved.
// =============================================================================

package main

import "testing"

// TestCleanForSpeech checks each kind of clean-up on small inputs.
func TestCleanForSpeech(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text is kept", "Hello there", "Hello there."},
		{"existing punctuation is kept", "Is it down?", "Is it down?"},
		{"bold is dropped", "**Monitor down**", "Monitor down."},
		{"markdown link becomes its label", "[View Incident](https://x.com/i/1)", "View Incident."},
		{"bare URL becomes the site name", "URL: https://www.options.cafe/login", "URL: options.cafe."},
		{"URL at end of sentence", "See https://cloudmanic.com.", "See cloudmanic.com."},
		{"list markers are dropped", "- Name: Harbor", "Name: Harbor."},
		{"headings are dropped", "## Alert", "Alert."},
		{"blank lines are dropped", "one\n\n\ntwo", "one.\ntwo."},
		{"underscores inside words are kept", "app_server is down", "app_server is down."},
		{"only formatting gives empty", "** **", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanForSpeech(tt.in); got != tt.want {
				t.Errorf("CleanForSpeech(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestCleanForSpeechCheckmateAlert checks a real Checkmate webhook message.
func TestCleanForSpeechCheckmateAlert(t *testing.T) {
	in := "**Monitor Down**\nOptions Cafe is not responding\n\n**Monitor Details:**\n- Name: Options Cafe\n- URL: https://options.cafe\n- Type: http\n- Status: down\n\n[View Incident](https://checkmate.cloudmanic.com/incidents/123)"
	want := "Monitor Down.\nOptions Cafe is not responding.\nMonitor Details:\nName: Options Cafe.\nURL: options.cafe.\nType: http.\nStatus: down.\nView Incident."
	if got := CleanForSpeech(in); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestSpokenHost checks URL shortening, including input that is not a URL.
func TestSpokenHost(t *testing.T) {
	if got := spokenHost("https://www.harbor.my/a/b?c=d"); got != "harbor.my" {
		t.Errorf("got %q", got)
	}
	if got := spokenHost("http://"); got != "" {
		t.Errorf("empty host: got %q", got)
	}
}
