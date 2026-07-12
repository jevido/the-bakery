package main

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// checkRootless verifies the agent process isn't running as root and that
// Podman itself reports operating in rootless mode. Hosts run arbitrary
// customer-deployed containers in a multi-tenant SaaS, so this is the
// runtime enforcement backstop for that hard security requirement — the
// install script (Phase 02 task 10) is the first line of defense, this is
// the second (task 09).
func checkRootless(ctx context.Context) error {
	if os.Geteuid() == 0 {
		return fmt.Errorf("agent is running as root (UID 0) — Bakery requires a rootless install")
	}

	out, err := runPodman(ctx, "info", "--format", "{{.Host.Security.Rootless}}")
	if err != nil {
		return fmt.Errorf("could not verify Podman is running rootless: %w", err)
	}

	if rootless := strings.TrimSpace(string(out)); rootless != "true" {
		return fmt.Errorf(
			"Podman is not running rootless (reported %q) — Bakery requires a rootless install",
			rootless,
		)
	}

	return nil
}
