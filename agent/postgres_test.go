package main

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

// TestProvisionPostgres is an integration test against real systemd --user
// and rootless Podman, not a unit test — it needs BAKERY_PG_INTEGRATION_TEST
// set (and to run as the bakery user on a host already past `bakery setup`)
// so a casual `go test ./...` never touches real system state.
func TestProvisionPostgres(t *testing.T) {
	if os.Getenv("BAKERY_PG_INTEGRATION_TEST") == "" {
		t.Skip("set BAKERY_PG_INTEGRATION_TEST=1 to run against real systemd --user + rootless Podman")
	}
	if os.Geteuid() == 0 {
		t.Fatal("must run as the bakery user, not root")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolving home directory: %v", err)
	}

	ctx := context.Background()
	url, err := provisionPostgres(ctx, home)
	if err != nil {
		t.Fatalf("provisionPostgres: %v", err)
	}
	if url == "" {
		t.Fatal("expected a non-empty DATABASE_URL")
	}
	t.Logf("provisioned: %s", url)

	// Re-invoke: idempotency requires the exact same DATABASE_URL (no
	// password regeneration, no data loss) and no error.
	url2, err := provisionPostgres(ctx, home)
	if err != nil {
		t.Fatalf("second provisionPostgres call: %v", err)
	}
	if url2 != url {
		t.Fatalf("second call returned a different DATABASE_URL: %q vs %q", url2, url)
	}

	if _, err := exec.LookPath("psql"); err == nil {
		cmd := exec.CommandContext(ctx, "psql", url, "-c", "select 1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("psql smoke query failed: %v\n%s", err, out)
		}
	}
}
