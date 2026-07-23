package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	registryQuadletUnit  = "registry.container"
	registryServiceName  = "registry.service"
	registryPort         = 5050 // fixed — matches src/lib/server/deploy/proxy.ts's REGISTRY_LOCAL_PORT
	registryReadyRetries = 30
	registryReadyDelay   = 2 * time.Second
)

type registryCredentials struct {
	Host         string
	PushUsername string
	PushPassword string
	PullUsername string
	PullPassword string
}

// provisionRegistry ports compose.yaml's dev-only `registry` service into a
// native Quadlet unit for `bakery bootstrap`, the same way provisionPostgres
// already ported the `db` service: idempotent, runs as the bakery user, no
// dependency on podman-compose. `domain` is only used to compute the public
// hostname persisted alongside the credentials — this function itself never
// touches Caddy (see this task's Notes for why that's someone else's job).
func provisionRegistry(ctx context.Context, home, domain string) (registryCredentials, error) {
	ensureUserSessionEnv()

	quadletDir := filepath.Join(home, ".config", "containers", "systemd")
	dataDir := filepath.Join(home, "bakery", "registry-data")
	authDir := filepath.Join(home, ".config", "bakery", "registry-auth")
	credsPath := filepath.Join(authDir, "credentials.env")
	htpasswdPath := filepath.Join(authDir, "htpasswd")
	unitPath := filepath.Join(quadletDir, registryQuadletUnit)

	for _, dir := range []string{quadletDir, dataDir, authDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return registryCredentials{}, fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	creds, err := existingRegistryCredentials(credsPath)
	if err != nil {
		return registryCredentials{}, fmt.Errorf("reading existing registry credentials: %w", err)
	}

	// Idempotency: credentials on disk mean a prior run already provisioned
	// the registry — regenerating here would orphan the htpasswd file's
	// hashes (and anything already relying on the old plaintext), same
	// reasoning as postgres.go's existingPassword check.
	if creds == nil {
		generated, err := generateRegistryCredentials(domain)
		if err != nil {
			return registryCredentials{}, fmt.Errorf("generating credentials: %w", err)
		}
		if err := writeHtpasswd(htpasswdPath, generated); err != nil {
			return registryCredentials{}, fmt.Errorf("writing htpasswd: %w", err)
		}
		if err := writeRegistryCredentials(credsPath, generated); err != nil {
			return registryCredentials{}, fmt.Errorf("persisting credentials: %w", err)
		}
		creds = &generated
	}

	if err := writeRegistryUnit(unitPath, dataDir, authDir); err != nil {
		return registryCredentials{}, fmt.Errorf("writing registry unit: %w", err)
	}

	if err := runCmd("systemctl", "--user", "daemon-reload"); err != nil {
		return registryCredentials{}, fmt.Errorf("systemctl --user daemon-reload: %w", err)
	}
	// `start`, not `enable --now` — same Quadlet-generator reasoning as
	// postgres.go/cmd_join.go's Caddy unit. Idempotent: starting an
	// already-active unit is a no-op success.
	if err := runCmd("systemctl", "--user", "start", registryServiceName); err != nil {
		return registryCredentials{}, fmt.Errorf("systemctl --user start %s: %w", registryServiceName, err)
	}

	if err := waitForRegistryReady(ctx, *creds); err != nil {
		return registryCredentials{}, err
	}

	return *creds, nil
}

func generateRegistryCredentials(domain string) (registryCredentials, error) {
	pushPassword, err := randomHex(24)
	if err != nil {
		return registryCredentials{}, err
	}
	pullPassword, err := randomHex(24)
	if err != nil {
		return registryCredentials{}, err
	}

	return registryCredentials{
		Host:         "registry." + domain,
		PushUsername: "push",
		PushPassword: pushPassword,
		PullUsername: "pull",
		PullPassword: pullPassword,
	}, nil
}

// writeHtpasswd bcrypt-hashes both credential pairs (cost 10, matching
// scripts/generate-registry-htpasswd.ts's Bun.password call — the
// registry:2 image's htpasswd auth handler only accepts bcrypt) into the
// file its Quadlet unit mounts read-only.
func writeHtpasswd(path string, creds registryCredentials) error {
	var lines []string
	for _, pair := range [][2]string{
		{creds.PushUsername, creds.PushPassword},
		{creds.PullUsername, creds.PullPassword},
	} {
		hash, err := bcrypt.GenerateFromPassword([]byte(pair[1]), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hashing credential for %s: %w", pair[0], err)
		}
		lines = append(lines, fmt.Sprintf("%s:%s", pair[0], hash))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

// writeRegistryCredentials persists the plaintext (bcrypt is one-way, so
// this is the only place the real values survive) for two consumers: this
// function's own idempotency check on the next run, and task 12's local
// image build/push, which needs to `podman login` with these same push
// credentials.
func writeRegistryCredentials(path string, c registryCredentials) error {
	content := fmt.Sprintf(
		"BAKERY_REGISTRY_HOST=%s\nBAKERY_REGISTRY_PUSH_USERNAME=%s\nBAKERY_REGISTRY_PUSH_PASSWORD=%s\nBAKERY_REGISTRY_PULL_USERNAME=%s\nBAKERY_REGISTRY_PULL_PASSWORD=%s\n",
		c.Host, c.PushUsername, c.PushPassword, c.PullUsername, c.PullPassword,
	)
	return os.WriteFile(path, []byte(content), 0o600)
}

func existingRegistryCredentials(path string) (*registryCredentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	c := registryCredentials{}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "BAKERY_REGISTRY_HOST":
			c.Host = value
		case "BAKERY_REGISTRY_PUSH_USERNAME":
			c.PushUsername = value
		case "BAKERY_REGISTRY_PUSH_PASSWORD":
			c.PushPassword = value
		case "BAKERY_REGISTRY_PULL_USERNAME":
			c.PullUsername = value
		case "BAKERY_REGISTRY_PULL_PASSWORD":
			c.PullPassword = value
		}
	}
	if c.Host == "" || c.PushPassword == "" || c.PullPassword == "" {
		return nil, fmt.Errorf("%s exists but is incomplete", path)
	}
	return &c, nil
}

func writeRegistryUnit(unitPath, dataDir, authDir string) error {
	unit := fmt.Sprintf(`[Unit]
Description=Bakery Registry

[Container]
Image=docker.io/library/registry:2
Environment=REGISTRY_AUTH=htpasswd
Environment=REGISTRY_AUTH_HTPASSWD_REALM=Bakery Registry
Environment=REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd
Volume=%s:/var/lib/registry:Z
Volume=%s:/auth:Z,ro
PublishPort=127.0.0.1:%d:5000

[Service]
Restart=always

[Install]
WantedBy=default.target
`, dataDir, authDir, registryPort)

	return os.WriteFile(unitPath, []byte(unit), 0o644)
}

// waitForRegistryReady polls with the push credentials rather than an
// unauthenticated request — a healthy, correctly-configured registry
// returns 401 for anonymous /v2/ requests, so that response alone can't
// distinguish "up and enforcing auth" from "not listening yet"; only an
// authenticated 200 proves both at once.
func waitForRegistryReady(ctx context.Context, creds registryCredentials) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/v2/", registryPort)
	client := &http.Client{Timeout: 3 * time.Second}

	var lastErr error
	for i := 0; i < registryReadyRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err == nil {
			req.SetBasicAuth(creds.PushUsername, creds.PushPassword)
			resp, doErr := client.Do(req)
			if doErr == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
				lastErr = fmt.Errorf("status %d", resp.StatusCode)
			} else {
				lastErr = doErr
			}
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(registryReadyDelay):
		}
	}
	return fmt.Errorf("registry never became ready after %d retries: %w", registryReadyRetries, lastErr)
}
