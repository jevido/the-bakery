package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

const defaultCheckinInterval = 45 * time.Second

type config struct {
	token           string
	controlPlaneURL string
	interval        time.Duration
}

func loadConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("bakery-agent", flag.ContinueOnError)
	token := fs.String("token", os.Getenv("BAKERY_TOKEN"), "per-host bearer token (env BAKERY_TOKEN)")
	url := fs.String("url", os.Getenv("BAKERY_URL"), "control-plane base URL (env BAKERY_URL)")
	interval := fs.Duration("interval", defaultCheckinInterval, "check-in interval")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	cfg := config{
		token:           *token,
		controlPlaneURL: *url,
		interval:        *interval,
	}

	if cfg.token == "" {
		return config{}, fmt.Errorf("missing token: set -token or BAKERY_TOKEN")
	}
	if cfg.controlPlaneURL == "" {
		return config{}, fmt.Errorf("missing control-plane URL: set -url or BAKERY_URL")
	}

	return cfg, nil
}
