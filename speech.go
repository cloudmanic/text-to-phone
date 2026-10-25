// =============================================================================
// File: speech.go
// Author: Spicer Matthews <spicer@cloudmanic.com>
// Created: 2026-10-24
// Copyright: 2026 Cloudmanic, LLC. All rights reserved.
// =============================================================================

package main

import (
	"net/url"
	"regexp"
	"strings"
)

// Patterns for the Markdown that alert senders (Checkmate, Slack-style
// webhooks) put in their text. Read aloud, these come out as noise.
var (
	markdownLink = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]*)\)`)
	bareURL      = regexp.MustCompile(`https?://[^\s<>()]+`)
	lineMarker   = regexp.MustCompile(`^(?:[-*+]|#{1,6}|>)\s+`)
	emphasis     = regexp.MustCompile("\\*\\*|__|~~|`|\\*")
)

// CleanForSpeech turns chat-style text into something that sounds right on
// a phone call. Links become their label, bare URLs become just the site
// name, formatting marks are dropped, and each line ends with punctuation so
// the voice pauses between lines instead of running them together.
func CleanForSpeech(text string) string {
	text = markdownLink.ReplaceAllString(text, "$1")
	text = bareURL.ReplaceAllStringFunc(text, spokenHost)

	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		line = lineMarker.ReplaceAllString(line, "")
		line = strings.TrimSpace(emphasis.ReplaceAllString(line, ""))
		if line == "" {
			continue
		}
		if !strings.ContainsAny(line[len(line)-1:], ".!?:;,") {
			line += "."
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// spokenHost shortens a URL to its site name, for example
// "https://www.options.cafe/login" becomes "options.cafe". A full URL read
// aloud is long and useless to the listener.
func spokenHost(raw string) string {
	u, err := url.Parse(strings.TrimRight(raw, ".,;:!?"))
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}
