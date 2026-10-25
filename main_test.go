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
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("/nope: status = %d, want 404", rec.Code)
	}
}
