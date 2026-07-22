package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// cmdDaemon runs the check-in loop against the control plane until it
// receives SIGINT/SIGTERM. This is today's original bakery-agent behavior,
// unchanged — just moved under the `daemon` subcommand.
func cmdDaemon(args []string) {
	cfg, err := loadConfig(args)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("bakery-agent %s starting, control plane %s, interval %s", agentVersion, cfg.controlPlaneURL, cfg.interval)

	// Fails loudly up front rather than only discovering this the first time
	// a deploy is attempted — metrics reporting still works either way, but
	// the operator needs to know their install isn't rootless (Phase 02
	// task 10's install script) well before that first deploy is queued.
	if err := checkRootless(ctx); err != nil {
		log.Printf("WARNING: rootless check failed at startup: %v", err)
		log.Printf("WARNING: deploy/stop/restart commands will be refused until this host is reinstalled rootless")
	}

	run(ctx, cfg)

	log.Println("bakery-agent shutting down")
}
