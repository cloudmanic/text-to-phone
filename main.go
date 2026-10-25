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
