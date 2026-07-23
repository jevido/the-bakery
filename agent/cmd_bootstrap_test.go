package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestPublicIPv4 guards against a real regression: ifconfig.me's root path
// only returns plain text when it recognizes the client as a CLI tool
// (curl's default User-Agent) — Go's own default User-Agent isn't
// recognized, so the root path serves a full HTML page instead, which
// silently broke every DNS check (comparing the box's real IP against an
// entire HTML document, always "mismatched"). /ip is a dedicated
// plain-text endpoint, unaffected by User-Agent — this asserts the
// response actually parses as an IPv4 address, not just that the request
// succeeded, so a regression back to the HTML-serving endpoint would fail
// loudly here instead of only at DNS-check time.
func TestPublicIPv4(t *testing.T) {
	ip, err := publicIPv4()
	if err != nil {
		t.Skipf("no network access to verify against: %v", err)
	}
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		t.Fatalf("expected a plain IPv4 address, got: %q", ip)
	}
}

// TestCheckDNSPointsHere hits the real internet (ifconfig.me + real DNS) —
// read-only and safe to run by default, but skipped rather than failed if
// that's unreachable (e.g. an offline dev machine), since network
// availability isn't what this test is actually verifying.
func TestCheckDNSPointsHere(t *testing.T) {
	if _, err := publicIPv4(); err != nil {
		t.Skipf("no network access to verify against: %v", err)
	}

	t.Run("nonexistent domain", func(t *testing.T) {
		err := checkDNSPointsHere("this-definitely-does-not-exist-12345.invalid")
		if err == nil {
			t.Fatal("expected an error for a domain with no DNS record")
		}
		if !strings.Contains(err.Error(), "does not resolve") {
			t.Errorf("expected a \"does not resolve\" error, got: %v", err)
		}
	})

	t.Run("domain resolving elsewhere", func(t *testing.T) {
		err := checkDNSPointsHere("example.com")
		if err == nil {
			t.Fatal("expected a mismatch error — this box is certainly not example.com")
		}
		if !strings.Contains(err.Error(), "resolves to") {
			t.Errorf("expected a \"resolves to X, but this box is Y\" error, got: %v", err)
		}
	})
}

func TestWaitForHTTP(t *testing.T) {
	t.Run("healthy immediately", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		if err := waitForHTTP(context.Background(), srv.URL, 5*time.Second); err != nil {
			t.Fatalf("expected success, got: %v", err)
		}
	})

	t.Run("times out against nothing listening", func(t *testing.T) {
		err := waitForHTTP(context.Background(), "http://127.0.0.1:1/", 2*time.Second)
		if err == nil {
			t.Fatal("expected a timeout error")
		}
	})
}

// TestCallBootstrapAPI is a real integration test against a running Bakery
// dev server, not a mock — set BAKERY_BOOTSTRAP_API_URL (and, if the server
// was started with BAKERY_BOOTSTRAP_SECRET set, BAKERY_BOOTSTRAP_SECRET
// here too) to run it against a fresh, empty database. Skipped otherwise so
// a casual `go test ./...` never makes a real HTTP call.
func TestCallBootstrapAPI(t *testing.T) {
	baseURL := os.Getenv("BAKERY_BOOTSTRAP_API_URL")
	if baseURL == "" {
		t.Skip("set BAKERY_BOOTSTRAP_API_URL to a running dev server against a fresh DB to run this test")
	}

	token, err := callBootstrapAPI(context.Background(), baseURL, bootstrapAPIRequest{
		Email:                "admin@example.com",
		Password:             "supersecret1",
		GuildName:            "Bakery",
		Domain:               "bakery.example.com",
		DatabaseURL:          "postgres://root:test@host.containers.internal:5432/local",
		Origin:               "https://bakery.example.com",
		BetterAuthSecret:     "test-better-auth-secret-value",
		EncryptionKey:        "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		RegistryHost:         "registry.bakery.example.com",
		RegistryPushUsername: "push",
		RegistryPushPassword: "test-push-password",
		RegistryPullUsername: "pull",
		RegistryPullPassword: "test-pull-password",
	}, os.Getenv("BAKERY_BOOTSTRAP_SECRET"))
	if err != nil {
		t.Fatalf("callBootstrapAPI: %v", err)
	}
	if !strings.HasPrefix(token, "bkry_host_") {
		t.Errorf("expected a bkry_host_ prefixed token, got: %q", token)
	}
}
