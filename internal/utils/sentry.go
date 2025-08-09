package utils

import (
	"log"
	"os"

	"github.com/getsentry/sentry-go"
)

// InitSentry initializes Sentry for error tracking
func InitSentry() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		dsn = "https://5552a4b2de69c6268847f9cb802b8285@o576394.ingest.us.sentry.io/4508987729707008"
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		EnableTracing:    true,
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	source := "environment"
	if os.Getenv("SENTRY_DSN") == "" {
		source = "fallback"
	}
	log.Printf("Sentry initialized with DSN from: %s", source)
}
