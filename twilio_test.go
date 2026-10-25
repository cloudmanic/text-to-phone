// =============================================================================
// File: twilio_test.go
// Author: Spicer Matthews <spicer@cloudmanic.com>
// Created: 2026-10-24
// Copyright: 2026 Cloudmanic, LLC. All rights reserved.
// =============================================================================

package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestBuildTwiML checks the script shape and that text is XML-escaped.
func TestBuildTwiML(t *testing.T) {
	got := BuildTwiML(`Tom & Jerry <say> "hi"`, "Polly.Joanna")
	want := `<Response><Pause length="1"/><Say voice="Polly.Joanna">Tom &amp; Jerry &lt;say&gt; &#34;hi&#34;</Say></Response>`
	if got != want {
		t.Errorf("BuildTwiML:\n got  %s\n want %s", got, want)
	}
}

// TestBuildTwiMLNoVoice checks that an empty voice leaves out the attribute.
func TestBuildTwiMLNoVoice(t *testing.T) {
	got := BuildTwiML("hello", "")
	if strings.Contains(got, "voice=") {
		t.Errorf("unexpected voice attribute: %s", got)
	}
}

// TestCallSendsRequest checks the URL, auth, and form fields sent to Twilio.
func TestCallSendsRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/2010-04-01/Accounts/AC123/Calls.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "AC123" || pass != "auth" {
			t.Errorf("basic auth = %q %q %v", user, pass, ok)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.PostForm.Get("To") != "+15550002222" || r.PostForm.Get("From") != "+15550001111" {
			t.Errorf("numbers = %v", r.PostForm)
		}
		if !strings.Contains(r.PostForm.Get("Twiml"), "<Say") || !strings.Contains(r.PostForm.Get("Twiml"), "hello there") {
			t.Errorf("Twiml = %q", r.PostForm.Get("Twiml"))
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"sid":"CA999","status":"queued"}`))
	}))
	defer srv.Close()

	c := &TwilioClient{AccountSID: "AC123", AuthToken: "auth", From: "+15550001111", BaseURL: srv.URL}
	got, err := c.Call(context.Background(), "+15550002222", "hello there")
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if got.SID != "CA999" || got.Status != "queued" {
		t.Errorf("result = %+v", got)
	}
}

// TestCallTwilioError checks that Twilio's own error message comes through.
func TestCallTwilioError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"code":21211,"message":"Invalid 'To' Phone Number"}`))
	}))
	defer srv.Close()

	c := &TwilioClient{AccountSID: "AC1", AuthToken: "a", From: "+1", BaseURL: srv.URL}
	_, err := c.Call(context.Background(), "bad", "hi")
	if err == nil || !strings.Contains(err.Error(), "Invalid 'To' Phone Number") || !strings.Contains(err.Error(), "21211") {
		t.Errorf("err = %v", err)
	}
}

// TestCallTwilioErrorNoBody checks the fallback when Twilio sends no JSON.
func TestCallTwilioErrorNoBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &TwilioClient{AccountSID: "AC1", AuthToken: "a", From: "+1", BaseURL: srv.URL}
	_, err := c.Call(context.Background(), "+2", "hi")
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("err = %v", err)
	}
}

// TestCallTextTooLong checks that oversize text fails before any request is sent.
func TestCallTextTooLong(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	c := &TwilioClient{AccountSID: "AC1", AuthToken: "a", From: "+1", BaseURL: srv.URL}
	_, err := c.Call(context.Background(), "+2", strings.Repeat("a", maxTwiMLLength))
	if !errors.Is(err, ErrTextTooLong) {
		t.Errorf("err = %v, want ErrTextTooLong", err)
	}
	if called {
		t.Error("Twilio was called for oversize text")
	}
}

// TestBaseURLDefault checks that production uses Twilio's real host.
func TestBaseURLDefault(t *testing.T) {
	if got := (&TwilioClient{}).baseURL(); got != twilioAPIBase {
		t.Errorf("baseURL = %q", got)
	}
	if got := (&TwilioClient{BaseURL: "http://x/"}).baseURL(); got != "http://x" {
		t.Errorf("baseURL trim = %q", got)
	}
}
