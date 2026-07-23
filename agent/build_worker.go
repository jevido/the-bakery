package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	buildWorkerDaemonUnit = "bakery-build-worker.service"
	buildWorkerSyncUnit   = "bakery-build-worker-sync.service"
	buildWorkerSyncTimer  = "bakery-build-worker-sync.timer"
	buildWorkerSyncPeriod = "2min"
)

// provisionBuildWorker closes the gap the old CI deploy job's
// `cd /opt/bakery/app && git pull && ... && systemctl --user restart
// bakery-build-worker` side effect quietly depended on: a checkout that
// someone set up by hand, outside any automated flow, at some point in this
// project's history. Nothing before this (`bakery setup`/`join`/`bootstrap`)
// actually provisioned one. This does: clones `source` into a fixed
// checkout, installs deps, writes its env file (same values the control
// plane container itself gets, minus the container-only DATABASE_URL
// rewrite — this runs natively on the host, not in a container namespace),
// and installs two systemd --user units — the long-running process, and a
// oneshot+timer pair that re-syncs it from source periodically (Phase 08
// task 14).
func provisionBuildWorker(ctx context.Context, home, source string, env map[string]string) error {
	bunBin, err := ensureBunInstalled(ctx, home)
	if err != nil {
		return fmt.Errorf("ensuring bun is installed: %w", err)
	}

	checkoutDir := filepath.Join(home, "bakery", "build-worker")
	if _, statErr := os.Stat(filepath.Join(checkoutDir, ".git")); os.IsNotExist(statErr) {
		fmt.Printf("Cloning %s for the build worker...\n", source)
		if err := os.MkdirAll(filepath.Dir(checkoutDir), 0o755); err != nil {
			return fmt.Errorf("creating parent dir: %w", err)
		}
		clone := exec.CommandContext(ctx, "git", "clone", "--quiet", "--branch", sourceRepoRef, source, checkoutDir)
		clone.Stdout = os.Stdout
		clone.Stderr = os.Stderr
		if err := clone.Run(); err != nil {
			return fmt.Errorf("git clone: %w", err)
		}
	}

	if err := runBunInstall(ctx, bunBin, checkoutDir); err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "bakery")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	envPath := filepath.Join(configDir, "build-worker.env")
	if err := writePersistentEnvFile(envPath, env); err != nil {
		return fmt.Errorf("writing build-worker env file: %w", err)
	}

	unitDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return fmt.Errorf("creating systemd user unit dir: %w", err)
	}

	if err := writeBuildWorkerServiceUnit(unitDir, checkoutDir, envPath, bunBin); err != nil {
		return err
	}

	syncScriptPath := filepath.Join(configDir, "sync-build-worker.sh")
	if err := writeBuildWorkerSyncScript(syncScriptPath, checkoutDir, bunBin); err != nil {
		return err
	}
	if err := writeBuildWorkerSyncUnits(unitDir, syncScriptPath); err != nil {
		return err
	}

	if err := runCmd("systemctl", "--user", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl --user daemon-reload: %w", err)
	}
	if err := runCmd("systemctl", "--user", "enable", "--now", buildWorkerDaemonUnit); err != nil {
		return fmt.Errorf("systemctl --user enable --now %s: %w", buildWorkerDaemonUnit, err)
	}
	if err := runCmd("systemctl", "--user", "enable", "--now", buildWorkerSyncTimer); err != nil {
		return fmt.Errorf("systemctl --user enable --now %s: %w", buildWorkerSyncTimer, err)
	}

	return nil
}

// ensureBunInstalled installs Bun via its own official installer if not
// already on this account — nothing in `bakery setup` installs it today
// (that step only covers rootless-Podman prerequisites), but the build
// worker is a Bun script, not a Go binary or container. Returns the full
// path to the binary rather than relying on PATH, since a systemd --user
// unit's default PATH may not include `~/.bun/bin`.
func ensureBunInstalled(ctx context.Context, home string) (string, error) {
	bunBin := filepath.Join(home, ".bun", "bin", "bun")
	if _, err := os.Stat(bunBin); err == nil {
		return bunBin, nil
	}

	fmt.Println("Installing Bun (needed to run the build worker)...")
	install := exec.CommandContext(ctx, "sh", "-c", "curl -fsSL https://bun.sh/install | bash")
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("installing bun: %w", err)
	}

	if _, err := os.Stat(bunBin); err != nil {
		return "", fmt.Errorf("bun installer completed but %s was not found: %w", bunBin, err)
	}
	return bunBin, nil
}

// runBunInstall installs node-gyp first — found live: better-sqlite3 (an
// optional peer dep pulled in transitively by better-auth/drizzle-orm's
// multi-dialect support, even though this app only ever uses Postgres) has
// a native addon whose install script directly invokes `node-gyp` by name,
// which isn't present anywhere on a fresh box (no Node.js/npm at all, only
// Bun) and fails the whole `bun install` outright ("node-gyp: command not
// found") without it. `bun install -g` puts it in the same bin directory as
// bun itself; `make`/`g++`/`python3` (the actual compiler toolchain
// node-gyp drives) are OS packages `bakery setup` now installs — same
// reasoning the Containerfile's own build stage already applies for its
// (containerized, so unaffected by any of this) `bun install`.
func runBunInstall(ctx context.Context, bunBin, dir string) error {
	bunDir := filepath.Dir(bunBin)
	env := append(os.Environ(), "PATH="+bunDir+":"+os.Getenv("PATH"))

	nodeGyp := exec.CommandContext(ctx, bunBin, "install", "-g", "node-gyp")
	nodeGyp.Env = env
	nodeGyp.Stdout = os.Stdout
	nodeGyp.Stderr = os.Stderr
	if err := nodeGyp.Run(); err != nil {
		return fmt.Errorf("installing node-gyp: %w", err)
	}

	cmd := exec.CommandContext(ctx, bunBin, "install", "--frozen-lockfile")
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bun install: %w", err)
	}
	return nil
}

// writePersistentEnvFile is writeEnvFile's (cmd_bootstrap.go) persistent-path
// counterpart — that one deliberately writes to a throwaway temp path for a
// single `podman run --env-file`; this one is read by a long-lived systemd
// unit's own `EnvironmentFile=`, so it needs a fixed, stable path instead.
func writePersistentEnvFile(path string, env map[string]string) error {
	var content string
	for key, value := range env {
		content += fmt.Sprintf("%s=%s\n", key, value)
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func writeBuildWorkerServiceUnit(unitDir, checkoutDir, envPath, bunBin string) error {
	unit := fmt.Sprintf(`[Unit]
Description=Bakery Build Worker
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=%s
EnvironmentFile=%s
ExecStart=%s run build-worker
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, checkoutDir, envPath, bunBin)

	return os.WriteFile(filepath.Join(unitDir, buildWorkerDaemonUnit), []byte(unit), 0o644)
}

// writeBuildWorkerSyncScript is a real shell script (not inlined into the
// unit's ExecStart=) so the fetch/compare/reset/reinstall/restart sequence
// stays readable and independently runnable for manual debugging — same
// reasoning install.sh itself is a standalone script rather than a one-liner.
func writeBuildWorkerSyncScript(path, checkoutDir, bunBin string) error {
	script := fmt.Sprintf(`#!/bin/sh
set -eu
cd %s
git fetch --quiet origin %s
LOCAL="$(git rev-parse HEAD)"
REMOTE="$(git rev-parse origin/%s)"
if [ "$LOCAL" != "$REMOTE" ]; then
	git reset --quiet --hard "origin/%s"
	%s install --frozen-lockfile
	systemctl --user restart %s
fi
`, checkoutDir, sourceRepoRef, sourceRepoRef, sourceRepoRef, bunBin, buildWorkerDaemonUnit)

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return err
	}
	return nil
}

func writeBuildWorkerSyncUnits(unitDir, syncScriptPath string) error {
	service := fmt.Sprintf(`[Unit]
Description=Sync the Bakery build worker checkout from source

[Service]
Type=oneshot
ExecStart=%s
`, syncScriptPath)
	if err := os.WriteFile(filepath.Join(unitDir, buildWorkerSyncUnit), []byte(service), 0o644); err != nil {
		return err
	}

	timer := fmt.Sprintf(`[Unit]
Description=Periodically sync and restart the Bakery build worker

[Timer]
OnBootSec=%s
OnUnitActiveSec=%s

[Install]
WantedBy=timers.target
`, buildWorkerSyncPeriod, buildWorkerSyncPeriod)
	return os.WriteFile(filepath.Join(unitDir, buildWorkerSyncTimer), []byte(timer), 0o644)
}
