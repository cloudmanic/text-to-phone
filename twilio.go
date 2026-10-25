// =============================================================================
// File: twilio.go
// Author: Spicer Matthews <spicer@cloudmanic.com>
// Created: 2026-10-24
// Copyright: 2026 Cloudmanic, LLC. All rights reserved.
// =============================================================================

package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// twilioAPIBase is Twilio's REST API host.
const twilioAPIBase = "https://api.twilio.com"

// maxTwiMLLength is Twilio's limit on the inline call script sent with a call.
const maxTwiMLLength = 4000

// ErrTextTooLong means the message will not fit in one Twilio call script.
var ErrTextTooLong = errors.New("text is too long to read in one call")

// CallResult is the part of Twilio's reply that we pass back to the client.
type CallResult struct {
	SID    string `json:"sid"`
	Status string `json:"status"`
}

// TwilioClient places outbound calls with Twilio's REST API.
type TwilioClient struct {
	AccountSID string
	AuthToken  string
	From       string
	Voice      string

	// BaseURL and HTTP are empty in production; tests set them to a fake server.
	BaseURL string
	HTTP    *http.Client
}

// Call dials `to` from the client's number. Twilio reads `text` aloud when
// the call is answered. The script goes inline with the request, so Twilio
// never needs to reach back to this server and it can run behind a firewall.
func (c *TwilioClient) Call(ctx context.Context, to, text string) (CallResult, error) {
	twiml := BuildTwiML(text, c.Voice)
	if utf8.RuneCountInString(twiml) > maxTwiMLLength {
		return CallResult{}, ErrTextTooLong
	}

	form := url.Values{
		"To":    {to},
		"From":  {c.From},
		"Twiml": {twiml},
	}
	endpoint := fmt.Sprintf("%s/2010-04-01/Accounts/%s/Calls.json", c.baseURL(), url.PathEscape(c.AccountSID))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return CallResult{}, fmt.Errorf("build twilio request: %w", err)
	}
	req.SetBasicAuth(c.AccountSID, c.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return CallResult{}, fmt.Errorf("send twilio request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return CallResult{}, fmt.Errorf("read twilio response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CallResult{}, twilioError(resp.StatusCode, body)
	}

	var result CallResult
	if err := json.Unmarshal(body, &result); err != nil {
		return CallResult{}, fmt.Errorf("decode twilio response: %w", err)
	}
	return result, nil
}

// BuildTwiML returns the call script Twilio runs once the call is answered.
// The one-second pause stops the first words being lost while the person
// lifts the phone to their ear.
func BuildTwiML(text, voice string) string {
	var b strings.Builder
	b.WriteString(`<Response><Pause length="1"/><Say`)
	if voice != "" {
		b.WriteString(` voice="`)
		_ = xml.EscapeText(&b, []byte(voice))
		b.WriteString(`"`)
	}
	b.WriteString(">")
	_ = xml.EscapeText(&b, []byte(text))
	b.WriteString("</Say></Response>")
	return b.String()
}

// twilioError turns a failed Twilio reply into an error. It keeps Twilio's own
// message when there is one, because that message names the actual problem
// (bad number, unverified caller ID, and so on).
func twilioError(status int, body []byte) error {
	var apiErr struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
		return fmt.Errorf("twilio returned %d: %s (code %d)", status, apiErr.Message, apiErr.Code)
	}
	return fmt.Errorf("twilio returned %d", status)
}

// baseURL returns the API host, so tests can point the client at a fake server.
func (c *TwilioClient) baseURL() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return twilioAPIBase
}

// httpClient returns the HTTP client. The default has a timeout so a slow
// Twilio reply cannot hold a request open forever.
func (c *TwilioClient) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 15 * time.Second}
}
