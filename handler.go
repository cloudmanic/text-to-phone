// =============================================================================
// File: handler.go
// Author: Spicer Matthews <spicer@cloudmanic.com>
// Created: 2026-10-24
// Copyright: 2026 Cloudmanic, LLC. All rights reserved.
// =============================================================================

package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
)

// maxBodyBytes caps request bodies. A spoken message never needs more.
const maxBodyBytes = 64 << 10

// Caller places a phone call that reads text aloud. The handler depends on
// this interface so tests can use a fake instead of calling Twilio.
type Caller interface {
	Call(ctx context.Context, to, text string) (CallResult, error)
}

// CallHandler serves the endpoint that turns posted text into a phone call.
type CallHandler struct {
	AccessToken string
	To          string
	Caller      Caller
}

// callRequest is the body shape for clients that post JSON.
type callRequest struct {
	Text        string `json:"text"`
	AccessToken string `json:"access_token"`
}

// ServeHTTP checks the access token, reads the text, and places the call.
// Only POST is allowed, so a link preview or browser prefetch of the URL can
// never ring the phone.
func (h *CallHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	token, text, err := readRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if token == "" {
		writeError(w, http.StatusBadRequest, "access_token is required")
		return
	}

	// Constant-time compare so response timing does not leak the token.
	if subtle.ConstantTimeCompare([]byte(token), []byte(h.AccessToken)) != 1 {
		writeError(w, http.StatusUnauthorized, "access_token is invalid")
		return
	}

	text = strings.TrimSpace(text)
	if text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}

	result, err := h.Caller.Call(r.Context(), h.To, text)
	if errors.Is(err, ErrTextTooLong) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		log.Printf("call failed: %v", err)
		writeError(w, http.StatusBadGateway, "could not place the call")
		return
	}

	log.Printf("call placed: sid=%s status=%s chars=%d", result.SID, result.Status, len(text))
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   result.Status,
		"call_sid": result.SID,
	})
}

// readRequest gets access_token and text from the query string and the body.
// It accepts form posts, JSON, and a raw text body, so curl, scripts, and
// webhooks can all use the endpoint. A value in the body wins over the query.
func readRequest(r *http.Request) (token, text string, err error) {
	query := r.URL.Query()
	token = query.Get("access_token")
	text = query.Get("text")

	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch mediaType {
	case "application/x-www-form-urlencoded", "multipart/form-data":
		if mediaType == "multipart/form-data" {
			err = r.ParseMultipartForm(maxBodyBytes)
		} else {
			err = r.ParseForm()
		}
		if err != nil {
			return "", "", errors.New("could not read the form body")
		}
		token = firstNonEmpty(r.PostForm.Get("access_token"), token)
		text = firstNonEmpty(r.PostForm.Get("text"), text)

	case "application/json":
		var body callRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			return "", "", errors.New("body is not valid JSON")
		}
		token = firstNonEmpty(body.AccessToken, token)
		text = firstNonEmpty(body.Text, text)

	default:
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			return "", "", errors.New("could not read the body")
		}
		text = firstNonEmpty(string(raw), text)
	}

	return token, text, nil
}

// firstNonEmpty returns the first value that is not blank.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// writeError sends a JSON error body with the given status code.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeJSON sends v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
