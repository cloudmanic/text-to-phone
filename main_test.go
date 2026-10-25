// =============================================================================
// File: main_test.go
// Author: Spicer Matthews <spicer@cloudmanic.com>
// Created: 2026-10-24
// Copyright: 2026 Cloudmanic, LLC. All rights reserved.
// =============================================================================

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewRouter checks that /call is wired up and other paths are not found.
func TestNewRouter(t *testing.T) {
	router := NewRouter(Config{AccessToken: "secret", CallToNumber: "+2"})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/call", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("/call without token: status = %d, want 400", rec.Code)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("/healthz: status = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("/nope: status = %d, want 404", rec.Code)
	}
}

// TestRunHealthcheck checks the exit code for a healthy, unhealthy, and missing server.
func TestRunHealthcheck(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(healthHandler))
	defer ok.Close()
	if code := runHealthcheck(ok.URL); code != 0 {
		t.Errorf("healthy: code = %d, want 0", code)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	if code := runHealthcheck(bad.URL); code != 1 {
		t.Errorf("unhealthy: code = %d, want 1", code)
	}

	if code := runHealthcheck("http://127.0.0.1:1/healthz"); code != 1 {
		t.Errorf("down: code = %d, want 1", code)
	}
}
