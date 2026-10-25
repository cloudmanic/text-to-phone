// =============================================================================
// File: main.go
// Author: Spicer Matthews <spicer@cloudmanic.com>
// Created: 2026-10-24
// Copyright: 2026 Cloudmanic, LLC. All rights reserved.
// =============================================================================

// Command text-to-phone runs an HTTP server. Text posted to /call is read
// aloud by Twilio in a phone call to the number set in .env.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// main loads settings, starts the server, and stops it cleanly on Ctrl+C or
// SIGTERM so an in-flight call request is not cut off.
func main() {
	// The container image has no shell or curl, so Docker's health check runs
	// the binary itself in this mode.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck("http://127.0.0.1:" + envOr("PORT", defaultPort) + "/healthz"))
	}

	cfg, err := LoadConfig(".env")
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           NewRouter(cfg),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s (POST /call)", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// NewRouter maps URL paths to handlers. It is separate from main so tests can
// exercise the real routes without opening a port.
func NewRouter(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/call", &CallHandler{
		AccessToken: cfg.AccessToken,
		To:          cfg.CallToNumber,
		Caller: &TwilioClient{
			AccountSID: cfg.TwilioAccountSID,
			AuthToken:  cfg.TwilioAuthToken,
			From:       cfg.TwilioFromNumber,
			Voice:      cfg.TwilioVoice,
		},
	})
	return mux
}

// healthHandler reports that the server is up. It needs no token and never
// calls Twilio, so Docker and Checkmate can check it as often as they like.
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// runHealthcheck asks the running server for /healthz and returns the exit
// code Docker expects: 0 when healthy, 1 when not.
func runHealthcheck(url string) int {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
