package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// loopbackRegistryHost is deliberately not registryCredentials.Host
// (registry.<domain>) — at the exact moment bootstrap needs to push this
// image, Caddy hasn't been told about the registry's public domain yet
// (that only happens once task 13's env vars land on a real, running
// deployment of this same app), so the only guaranteed-reachable address is
// loopback. Since this app only ever runs on the box it was bootstrapped on,
// a loopback reference stays valid for this specific image's entire
// lifecycle, not just for the duration of bootstrap.
func loopbackRegistryHost() string {
	return fmt.Sprintf("127.0.0.1:%d", registryPort)
}

// buildAndPushSelfImage closes the chicken-and-egg gap `bakery bootstrap`
// would otherwise hit: no build-worker/agent pipeline exists yet to build
// anything the normal way, and there is no external CI publishing an image
// to pull instead (Phase 08, task 12). It clones `source` locally, builds
// the exact same Containerfile every build-worker-driven build uses, and
// pushes the result to the registry task 10 just provisioned — over
// loopback, not the public domain, per loopbackRegistryHost's reasoning.
func buildAndPushSelfImage(ctx context.Context, home, source string, creds registryCredentials) (string, error) {
	if err := ensureLoopbackRegistryInsecure(home); err != nil {
		return "", fmt.Errorf("configuring registry as insecure: %w", err)
	}

	srcDir, err := os.MkdirTemp("", "bakery-bootstrap-src-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(srcDir)

	fmt.Printf("Cloning %s (%s)...\n", source, sourceRepoRef)
	clone := exec.CommandContext(ctx, "git", "clone", "--quiet", "--depth=1",
		"--branch", sourceRepoRef, source, srcDir)
	clone.Stdout = os.Stdout
	clone.Stderr = os.Stderr
	if err := clone.Run(); err != nil {
		return "", fmt.Errorf("git clone: %w", err)
	}

	shortSHA, err := gitShortSHA(ctx, srcDir)
	if err != nil {
		return "", fmt.Errorf("resolving commit sha: %w", err)
	}

	imageRef := fmt.Sprintf("%s/%s/%s:%s", loopbackRegistryHost(), selfImageBuildOrg, selfImageBuildApp, shortSHA)

	fmt.Printf("Building %s...\n", imageRef)
	// Same build-arg shape the old CI `publish` job used
	// (`--build-arg BAKERY_SELF_IMAGE=${IMAGE}:${{ github.sha }}`) — the
	// image has to know its own eventual pushed reference so the running
	// container can report it back via the bootstrap API.
	build := exec.CommandContext(ctx, "podman", "build",
		"-f", "Containerfile",
		"-t", imageRef,
		"--build-arg", "BAKERY_SELF_IMAGE="+imageRef,
		".",
	)
	build.Dir = srcDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return "", fmt.Errorf("podman build: %w", err)
	}

	fmt.Println("Logging in to the registry...")
	if err := podmanLogin(ctx, registryAuth{
		Host:     loopbackRegistryHost(),
		Username: creds.PushUsername,
		Password: creds.PushPassword,
	}); err != nil {
		return "", err
	}

	fmt.Printf("Pushing %s...\n", imageRef)
	push := exec.CommandContext(ctx, "podman", "push", imageRef)
	push.Stdout = os.Stdout
	push.Stderr = os.Stderr
	if err := push.Run(); err != nil {
		return "", fmt.Errorf("podman push: %w", err)
	}

	return imageRef, nil
}

func gitShortSHA(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ensureLoopbackRegistryInsecure marks 127.0.0.1:<registryPort> as HTTP,
// not HTTPS-with-verification, in Podman's registries.conf.d — needed for
// both this function's own push and, later, the real deployment engine's
// *implicit* pull when `systemctl --user start` brings up this app's
// real Quadlet unit (that pull takes no CLI flags of its own, so
// `--tls-verify=false` isn't an option there; this is the only way to make
// an unencrypted loopback registry usable at all). Scoped to this one
// address, not the registry's public domain, which gets real Caddy TLS.
func ensureLoopbackRegistryInsecure(home string) error {
	dir := filepath.Join(home, ".config", "containers", "registries.conf.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf("[[registry]]\nlocation = %q\ninsecure = true\n", loopbackRegistryHost())
	return os.WriteFile(filepath.Join(dir, "bakery-loopback-registry.conf"), []byte(content), 0o644)
}
