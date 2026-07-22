package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	postgresQuadletUnit  = "db.container"
	postgresServiceName  = "db.service"
	postgresContainer    = "systemd-db" // Podman's systemd-<unit-basename> naming for Quadlet-managed containers
	postgresPort         = 5434
	postgresReadyRetries = 30
	postgresReadyDelay   = 2 * time.Second
)

// provisionPostgres ports scripts/bootstrap-host.sh's phase-2 Postgres setup
// into Go for `bakery bootstrap`: writes and starts the db Quadlet unit,
// waits for readiness, and returns the resulting DATABASE_URL. Must run as
// the bakery user (same rootless Podman/systemd --user session every other
// Quadlet-managed unit in this codebase depends on), not root.
func provisionPostgres(ctx context.Context, home string) (string, error) {
	ensureUserSessionEnv()

	quadletDir := filepath.Join(home, ".config", "containers", "systemd")
	bakeryDir := filepath.Join(home, "bakery")
	pgDataDir := filepath.Join(bakeryDir, "pgdata")
	unitPath := filepath.Join(quadletDir, postgresQuadletUnit)

	if err := os.MkdirAll(quadletDir, 0o755); err != nil {
		return "", fmt.Errorf("creating quadlet dir: %w", err)
	}
	if err := os.MkdirAll(pgDataDir, 0o755); err != nil {
		return "", fmt.Errorf("creating pgdata dir: %w", err)
	}

	password, err := existingPassword(unitPath)
	if err != nil {
		return "", fmt.Errorf("reading existing db unit: %w", err)
	}

	// Idempotency: a unit file already on disk means Postgres was already
	// provisioned by a prior `bakery bootstrap` run — re-running must never
	// regenerate the password (would orphan whatever already trusts it) or
	// touch pgdata, so the write below is skipped entirely in that case.
	if password == "" {
		password, err = randomHex(24)
		if err != nil {
			return "", fmt.Errorf("generating password: %w", err)
		}
		if err := writePostgresUnit(unitPath, pgDataDir, password); err != nil {
			return "", fmt.Errorf("writing db unit: %w", err)
		}
	}

	if err := runCmd("systemctl", "--user", "daemon-reload"); err != nil {
		return "", fmt.Errorf("systemctl --user daemon-reload: %w", err)
	}
	// `start`, not `enable --now`: Quadlet-generated units live in systemd's
	// generator output dir, not a persistent unit file location, so `enable`
	// fails ("transient or generated") — same fix as agent/cmd_join.go's
	// Caddy unit. Also idempotent: starting an already-active unit is a
	// no-op success, so this is safe to call unconditionally every run.
	if err := runCmd("systemctl", "--user", "start", postgresServiceName); err != nil {
		return "", fmt.Errorf("systemctl --user start %s: %w", postgresServiceName, err)
	}

	if err := waitForPostgresReady(ctx); err != nil {
		return "", err
	}

	return fmt.Sprintf("postgres://root:%s@127.0.0.1:%d/local", password, postgresPort), nil
}

// existingPassword extracts POSTGRES_PASSWORD from an already-written
// db.container, if one exists. Returns "" (no error) if the unit doesn't
// exist yet.
func existingPassword(unitPath string) (string, error) {
	data, err := os.ReadFile(unitPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	const prefix = "Environment=POSTGRES_PASSWORD="
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix), nil
		}
	}
	return "", fmt.Errorf("%s exists but has no POSTGRES_PASSWORD line", unitPath)
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func writePostgresUnit(unitPath, pgDataDir, password string) error {
	unit := fmt.Sprintf(`[Unit]
Description=Bakery Postgres

[Container]
Image=docker.io/library/postgres:17
Environment=POSTGRES_USER=root
Environment=POSTGRES_DB=local
Environment=POSTGRES_PASSWORD=%s
Volume=%s:/var/lib/postgresql:Z
PublishPort=127.0.0.1:%d:5432

[Service]
Restart=always

[Install]
WantedBy=default.target
`, password, pgDataDir, postgresPort)

	return os.WriteFile(unitPath, []byte(unit), 0o644)
}

func waitForPostgresReady(ctx context.Context) error {
	for i := 0; i < postgresReadyRetries; i++ {
		cmd := exec.CommandContext(ctx, "podman", "exec", postgresContainer, "pg_isready", "-U", "root")
		if err := cmd.Run(); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(postgresReadyDelay):
		}
	}
	return fmt.Errorf("postgres never became ready after %d retries", postgresReadyRetries)
}
