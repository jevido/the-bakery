package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestGenerateAndPersistRegistryCredentials covers the parts of
// provisionRegistry that don't need a live container/systemd session:
// credential generation, htpasswd bcrypt hashing (verifiable against the
// same plaintext), and the round-trip through writeRegistryCredentials /
// existingRegistryCredentials that makes re-running bakery bootstrap
// idempotent. Runs unconditionally (no port/systemd dependency), unlike
// TestProvisionRegistry below.
func TestGenerateAndPersistRegistryCredentials(t *testing.T) {
	creds, err := generateRegistryCredentials("example.test")
	if err != nil {
		t.Fatalf("generateRegistryCredentials: %v", err)
	}
	if creds.Host != "registry.example.test" {
		t.Fatalf("expected host registry.example.test, got %q", creds.Host)
	}
	if creds.PushPassword == "" || creds.PullPassword == "" {
		t.Fatal("expected non-empty generated passwords")
	}
	if creds.PushPassword == creds.PullPassword {
		t.Fatal("push and pull passwords must not collide")
	}

	dir := t.TempDir()
	htpasswdPath := filepath.Join(dir, "htpasswd")
	if err := writeHtpasswd(htpasswdPath, creds); err != nil {
		t.Fatalf("writeHtpasswd: %v", err)
	}
	data, err := os.ReadFile(htpasswdPath)
	if err != nil {
		t.Fatalf("reading htpasswd: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 htpasswd lines, got %d: %q", len(lines), data)
	}
	verifyHtpasswdLine(t, lines[0], creds.PushUsername, creds.PushPassword)
	verifyHtpasswdLine(t, lines[1], creds.PullUsername, creds.PullPassword)

	credsPath := filepath.Join(dir, "credentials.env")
	if err := writeRegistryCredentials(credsPath, creds); err != nil {
		t.Fatalf("writeRegistryCredentials: %v", err)
	}
	roundTripped, err := existingRegistryCredentials(credsPath)
	if err != nil {
		t.Fatalf("existingRegistryCredentials: %v", err)
	}
	if roundTripped == nil || *roundTripped != creds {
		t.Fatalf("round-tripped credentials don't match: got %+v, want %+v", roundTripped, creds)
	}

	// A path that doesn't exist yet must mean "not provisioned", not an
	// error — this is what makes the first bootstrap run generate fresh
	// credentials instead of failing.
	missing, err := existingRegistryCredentials(filepath.Join(dir, "does-not-exist"))
	if err != nil {
		t.Fatalf("existingRegistryCredentials on a missing file: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for a missing credentials file, got %+v", missing)
	}
}

func verifyHtpasswdLine(t *testing.T, line, wantUser, wantPassword string) {
	t.Helper()
	user, hash, ok := strings.Cut(line, ":")
	if !ok {
		t.Fatalf("malformed htpasswd line: %q", line)
	}
	if user != wantUser {
		t.Fatalf("expected user %q, got %q", wantUser, user)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(wantPassword)); err != nil {
		t.Fatalf("bcrypt hash for %q does not verify against its own plaintext password: %v", user, err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("definitely-wrong")); err == nil {
		t.Fatalf("bcrypt hash for %q incorrectly verified a wrong password", user)
	}
}

// TestProvisionRegistry is an integration test against real systemd --user
// and rootless Podman, not a unit test — it needs BAKERY_REGISTRY_INTEGRATION_TEST
// set, mirroring postgres_test.go's TestProvisionPostgres, so a casual
// `go test ./...` never touches real system state.
func TestProvisionRegistry(t *testing.T) {
	if os.Getenv("BAKERY_REGISTRY_INTEGRATION_TEST") == "" {
		t.Skip("set BAKERY_REGISTRY_INTEGRATION_TEST=1 to run against real systemd --user + rootless Podman")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolving home directory: %v", err)
	}

	ctx := context.Background()
	creds, err := provisionRegistry(ctx, home, "test.invalid")
	if err != nil {
		t.Fatalf("provisionRegistry: %v", err)
	}
	if creds.Host != "registry.test.invalid" {
		t.Fatalf("expected host registry.test.invalid, got %q", creds.Host)
	}
	t.Logf("provisioned: push=%s pull=%s", creds.PushUsername, creds.PullUsername)

	// Re-invoke: idempotency requires the exact same credentials (no
	// regeneration) and no error.
	creds2, err := provisionRegistry(ctx, home, "test.invalid")
	if err != nil {
		t.Fatalf("second provisionRegistry call: %v", err)
	}
	if creds2 != creds {
		t.Fatalf("second call returned different credentials: %+v vs %+v", creds2, creds)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/v2/", registryPort)

	t.Run("anonymous request is rejected", func(t *testing.T) {
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("GET %s: %v", url, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for anonymous request, got %d", resp.StatusCode)
		}
	})

	t.Run("authenticated request succeeds", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.SetBasicAuth(creds.PushUsername, creds.PushPassword)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("authenticated GET %s: %v", url, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for authenticated request, got %d", resp.StatusCode)
		}
	})

	t.Run("push then pull a real image with push/pull creds, wrong creds fail", func(t *testing.T) {
		if _, err := exec.LookPath("podman"); err != nil {
			t.Skip("podman not on PATH")
		}

		imageRef := fmt.Sprintf("127.0.0.1:%d/bakery-test/registry-check:1", registryPort)

		pull := exec.CommandContext(ctx, "podman", "pull", "--quiet", "docker.io/library/alpine:latest")
		if out, err := pull.CombinedOutput(); err != nil {
			t.Skipf("no network access to pull a base test image: %v: %s", err, out)
		}

		tag := exec.CommandContext(ctx, "podman", "tag", "docker.io/library/alpine:latest", imageRef)
		if out, err := tag.CombinedOutput(); err != nil {
			t.Fatalf("podman tag: %v: %s", err, out)
		}

		push := exec.CommandContext(ctx, "podman", "push", "--tls-verify=false",
			"--creds", creds.PushUsername+":"+creds.PushPassword, imageRef)
		if out, err := push.CombinedOutput(); err != nil {
			t.Fatalf("podman push with push creds: %v: %s", err, out)
		}

		// Remove the local copy so the pull below can't silently pass by
		// finding it already present rather than actually fetching it.
		_ = exec.CommandContext(ctx, "podman", "rmi", imageRef).Run()

		pullBack := exec.CommandContext(ctx, "podman", "pull", "--tls-verify=false",
			"--creds", creds.PullUsername+":"+creds.PullPassword, imageRef)
		if out, err := pullBack.CombinedOutput(); err != nil {
			t.Fatalf("podman pull with pull creds: %v: %s", err, out)
		}
		_ = exec.CommandContext(ctx, "podman", "rmi", imageRef).Run()

		badPull := exec.CommandContext(ctx, "podman", "pull", "--tls-verify=false",
			"--creds", "wrong:credentials", imageRef)
		if out, err := badPull.CombinedOutput(); err == nil {
			t.Fatalf("expected pull with wrong credentials to fail, but it succeeded: %s", out)
		}
	})

	t.Run("data survives a unit restart", func(t *testing.T) {
		restart := exec.CommandContext(ctx, "systemctl", "--user", "restart", registryServiceName)
		if out, err := restart.CombinedOutput(); err != nil {
			t.Fatalf("systemctl --user restart %s: %v: %s", registryServiceName, err, out)
		}

		if err := waitForRegistryReady(ctx, creds); err != nil {
			t.Fatalf("registry not ready after restart: %v", err)
		}

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/v2/bakery-test/registry-check/tags/list", registryPort), nil)
		req.SetBasicAuth(creds.PullUsername, creds.PullPassword)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET tags list: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected the previously-pushed image to still be listed after restart, got status %d", resp.StatusCode)
		}
	})
}
