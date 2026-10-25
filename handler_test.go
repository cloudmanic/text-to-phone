// =============================================================================
// File: handler_test.go
// Author: Spicer Matthews <spicer@cloudmanic.com>
// Created: 2026-10-24
// Copyright: 2026 Cloudmanic, LLC. All rights reserved.
// =============================================================================

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// fakeCaller records calls instead of contacting Twilio.
type fakeCaller struct {
	calls  int
	to     string
	text   string
	result CallResult
	err    error
}

// Call records the request and returns the canned result.
func (f *fakeCaller) Call(_ context.Context, to, text string) (CallResult, error) {
	f.calls++
	f.to = to
	f.text = text
	return f.result, f.err
}

// newTestHandler returns a handler with a known token and a fake caller.
func newTestHandler() (*CallHandler, *fakeCaller) {
	fc := &fakeCaller{result: CallResult{SID: "CA1", Status: "queued"}}
	return &CallHandler{AccessToken: "secret", To: "+15550002222", Caller: fc}, fc
}

// serve runs one request through the handler and returns the recorder.
func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// formRequest builds a form-encoded POST to target.
func formRequest(target string, form url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// TestMissingTokenReturns400 checks that a request without access_token gets 400.
func TestMissingTokenReturns400(t *testing.T) {
	h, fc := newTestHandler()
	rec := serve(h, formRequest("/call", url.Values{"text": {"hi"}}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if fc.calls != 0 {
		t.Error("call placed without a token")
	}
}

// TestWrongTokenReturns401 checks that a bad token is refused.
func TestWrongTokenReturns401(t *testing.T) {
	h, fc := newTestHandler()
	rec := serve(h, formRequest("/call", url.Values{"text": {"hi"}, "access_token": {"nope"}}))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if fc.calls != 0 {
		t.Error("call placed with a wrong token")
	}
}

// TestTokenInPostBody checks a form post with the token in the body.
func TestTokenInPostBody(t *testing.T) {
	h, fc := newTestHandler()
	rec := serve(h, formRequest("/call", url.Values{"text": {"hello"}, "access_token": {"secret"}}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if fc.to != "+15550002222" || fc.text != "hello." {
		t.Errorf("caller got to=%q text=%q", fc.to, fc.text)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["call_sid"] != "CA1" || body["status"] != "queued" {
		t.Errorf("body = %v", body)
	}
}

// TestTokenInQueryString checks the token as a GET-style query parameter on a POST.
func TestTokenInQueryString(t *testing.T) {
	h, fc := newTestHandler()
	rec := serve(h, formRequest("/call?access_token=secret", url.Values{"text": {"hello"}}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if fc.calls != 1 {
		t.Errorf("calls = %d", fc.calls)
	}
}

// TestJSONBody checks a JSON post with both fields in the body.
func TestJSONBody(t *testing.T) {
	h, fc := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/call", strings.NewReader(`{"text":"from json","access_token":"secret"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := serve(h, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if fc.text != "from json." {
		t.Errorf("text = %q", fc.text)
	}
}

// TestBadJSONReturns400 checks that broken JSON is refused.
func TestBadJSONReturns400(t *testing.T) {
	h, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/call?access_token=secret", strings.NewReader(`{"text":`))
	req.Header.Set("Content-Type", "application/json")
	if rec := serve(h, req); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// TestRawTextBody checks a plain-text body with the token in the query string.
func TestRawTextBody(t *testing.T) {
	h, fc := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/call?access_token=secret", strings.NewReader("raw words"))
	req.Header.Set("Content-Type", "text/plain")
	rec := serve(h, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if fc.text != "raw words." {
		t.Errorf("text = %q", fc.text)
	}
}

// TestMultipartBody checks a multipart form post.
func TestMultipartBody(t *testing.T) {
	h, fc := newTestHandler()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("text", "multi")
	mw.WriteField("access_token", "secret")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/call", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := serve(h, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if fc.text != "multi." {
		t.Errorf("text = %q", fc.text)
	}
}

// TestMissingTextReturns400 checks that blank text is refused.
func TestMissingTextReturns400(t *testing.T) {
	h, fc := newTestHandler()
	rec := serve(h, formRequest("/call", url.Values{"text": {"   "}, "access_token": {"secret"}}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if fc.calls != 0 {
		t.Error("call placed with no text")
	}
}

// TestGetReturns405 checks that GET cannot place a call.
func TestGetReturns405(t *testing.T) {
	h, fc := newTestHandler()
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/call?access_token=secret&text=hi", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if rec.Header().Get("Allow") != http.MethodPost {
		t.Errorf("Allow = %q", rec.Header().Get("Allow"))
	}
	if fc.calls != 0 {
		t.Error("GET placed a call")
	}
}

// TestTextTooLongReturns400 checks that the caller's size error becomes a 400.
func TestTextTooLongReturns400(t *testing.T) {
	h, fc := newTestHandler()
	fc.err = ErrTextTooLong
	rec := serve(h, formRequest("/call", url.Values{"text": {"hi"}, "access_token": {"secret"}}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// TestTwilioFailureReturns502 checks that a Twilio error becomes a 502.
func TestTwilioFailureReturns502(t *testing.T) {
	h, fc := newTestHandler()
	fc.err = errors.New("twilio down")
	rec := serve(h, formRequest("/call", url.Values{"text": {"hi"}, "access_token": {"secret"}}))
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "twilio down") {
		t.Error("internal error leaked to the client")
	}
}

// TestFirstNonEmpty checks that blank values are skipped.
func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", "b", "c"); got != "b" {
		t.Errorf("got %q", got)
	}
	if got := firstNonEmpty("", " "); got != "" {
		t.Errorf("got %q", got)
	}
}
