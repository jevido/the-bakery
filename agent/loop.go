package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

// run executes the check-in loop until ctx is cancelled. Network and
// server errors are logged and retried on the next interval rather than
// crashing the process, since a host being briefly unreachable or a
// control-plane hiccup shouldn't take the agent down.
func run(ctx context.Context, cfg config) {
	httpClient := newHTTPClient()

	doCheckin(ctx, httpClient, cfg)

	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			doCheckin(ctx, httpClient, cfg)
		}
	}
}

func doCheckin(ctx context.Context, httpClient *http.Client, cfg config) {
	payload := collectPayload(ctx)

	resp, err := checkin(ctx, httpClient, cfg, payload)
	if err != nil {
		log.Printf("check-in failed: %v", err)
		return
	}

	log.Printf("check-in ok, %d pending command(s)", len(resp.PendingCommands))
}
