package Logger

import (
	"log"

	"github.com/getsentry/sentry-go"
)

func Init() {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: "https://d735b9fec79664425c2a08556e15406a@o4508080703864832.ingest.de.sentry.io/4508080710025296",
		// Set TracesSampleRate to 1.0 to capture 100%
		// of transactions for tracing.
		// We recommend adjusting this value in production,
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
}
