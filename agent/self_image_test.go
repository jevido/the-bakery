package main

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

// TestBuildAndPushSelfImage is a real, heavy integration test (a full
// Containerfile build: bun install + bun run build + agent cross-compile)
// against real systemd --user + rootless Podman — gated behind
// BAKERY_SELF_IMAGE_INTEGRATION_TEST, same convention as
// BAKERY_REGISTRY_INTEGRATION_TEST in registry_test.go, and skipped by
// default since it's slow and not appropriate to run casually.
func TestBuildAndPushSelfImage(t *testing.T) {
	if os.Getenv("BAKERY_SELF_IMAGE_INTEGRATION_TEST") == "" {
		t.Skip("set BAKERY_SELF_IMAGE_INTEGRATION_TEST=1 to run a real (slow) Containerfile build + push")
	}
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman not on PATH")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolving home directory: %v", err)
	}

	ctx := context.Background()

	creds, err := provisionRegistry(ctx, home, "self-image-test.invalid")
	if err != nil {
		t.Fatalf("provisionRegistry: %v", err)
	}
	t.Cleanup(func() {
		_ = exec.Command("systemctl", "--user", "stop", registryServiceName).Run()
		_ = exec.Command("systemctl", "--user", "reset-failed", registryServiceName).Run()
	})

	// file:// so this exercises the exact same git-clone-then-build path
	// production uses, without a network fetch of the public repo — the
	// source under test is this working tree's own committed state.
	source := "file://" + repoRootForTest(t)

	imageRef, err := buildAndPushSelfImage(ctx, home, source, creds)
	if err != nil {
		t.Fatalf("buildAndPushSelfImage: %v", err)
	}
	t.Logf("built and pushed: %s", imageRef)
	t.Cleanup(func() {
		_ = exec.Command("podman", "rmi", "-f", imageRef).Run()
	})

	// Prove it's genuinely pullable back from the registry, not just
	// present in local storage from the build step above.
	if out, err := exec.Command("podman", "rmi", imageRef).CombinedOutput(); err != nil {
		t.Fatalf("removing local copy before pull-back check: %v: %s", err, out)
	}
	pull := exec.CommandContext(ctx, "podman", "pull", imageRef)
	if out, err := pull.CombinedOutput(); err != nil {
		t.Fatalf("pulling back the pushed image failed: %v: %s", err, out)
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel: %v", err)
	}
	dir := string(out)
	for len(dir) > 0 && (dir[len(dir)-1] == '\n' || dir[len(dir)-1] == '\r') {
		dir = dir[:len(dir)-1]
	}
	return dir
}
