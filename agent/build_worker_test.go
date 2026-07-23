package main

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

// TestProvisionBuildWorker is a real, heavy integration test (clones this
// repo, installs Bun if missing, runs a real `bun install`, and enables two
// real systemd --user units) against real systemd --user + rootless
// Podman — gated behind BAKERY_BUILD_WORKER_INTEGRATION_TEST, same
// convention as this phase's other integration tests, since it's slow and
// installs real software on the host running it.
func TestProvisionBuildWorker(t *testing.T) {
	if os.Getenv("BAKERY_BUILD_WORKER_INTEGRATION_TEST") == "" {
		t.Skip("set BAKERY_BUILD_WORKER_INTEGRATION_TEST=1 to run a real (slow) build-worker provisioning pass")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolving home directory: %v", err)
	}

	t.Cleanup(func() {
		_ = exec.Command("systemctl", "--user", "disable", "--now", buildWorkerDaemonUnit).Run()
		_ = exec.Command("systemctl", "--user", "disable", "--now", buildWorkerSyncTimer).Run()
		_ = exec.Command("systemctl", "--user", "reset-failed", buildWorkerDaemonUnit).Run()
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	})

	source := "file://" + repoRootForTest(t)
	env := map[string]string{
		"DATABASE_URL": "postgres://root:test@127.0.0.1:1/nonexistent", // never actually connected to in this test
	}

	ctx := context.Background()
	if err := provisionBuildWorker(ctx, home, source, env); err != nil {
		t.Fatalf("provisionBuildWorker: %v", err)
	}

	status := exec.Command("systemctl", "--user", "is-enabled", buildWorkerDaemonUnit)
	if out, err := status.CombinedOutput(); err != nil {
		t.Fatalf("%s is not enabled: %v: %s", buildWorkerDaemonUnit, err, out)
	}

	timerStatus := exec.Command("systemctl", "--user", "is-active", buildWorkerSyncTimer)
	if out, err := timerStatus.CombinedOutput(); err != nil {
		t.Fatalf("%s is not active: %v: %s", buildWorkerSyncTimer, err, out)
	}

	// Re-run: idempotency — must not re-clone (checkout already has a
	// .git dir) or fail because the units already exist.
	if err := provisionBuildWorker(ctx, home, source, env); err != nil {
		t.Fatalf("second provisionBuildWorker call: %v", err)
	}
}
