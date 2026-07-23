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

	cycle := 0
	tick := func() {
		doCheckin(ctx, httpClient, cfg)
		if cycle%updateCheckEveryNCycles == 0 {
			checkForUpdate(ctx, httpClient, cfg)
		}
		cycle++
	}

	tick()

	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick()
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

	// Cheap and idempotent (a repeat `podman login` to the same host just
	// re-confirms the credential), so it's simplest to just do this on every
	// check-in that carries it rather than tracking whether it changed —
	// self-healing if auth.json is ever cleared/corrupted between check-ins.
	if resp.RegistryAuth != nil {
		if err := podmanLogin(ctx, *resp.RegistryAuth); err != nil {
			log.Printf("registry login failed: %v", err)
		}
	}

	if len(resp.PendingCommands) > 0 {
		executeCommands(ctx, httpClient, cfg, resp.PendingCommands)
	}
}
