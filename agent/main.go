// Command bakery-agent runs on a customer's host and periodically reports
// system metrics to the Bakery control plane over outbound HTTP, so it
// works behind NAT/home routers with no inbound access required.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := loadConfig(os.Args[1:])
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("bakery-agent %s starting, control plane %s, interval %s", agentVersion, cfg.controlPlaneURL, cfg.interval)

	run(ctx, cfg)

	log.Println("bakery-agent shutting down")
}
