package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const commandTimeout = 30 * time.Second
const healthCheckTimeout = 20 * time.Second
const healthCheckInterval = 500 * time.Millisecond

// caddyAdminBaseURL is Caddy's admin API. The Caddy Quadlet unit runs with
// Network=host (Phase 05 task 01), so Caddy's default admin bind
// (localhost:2019) is genuinely the host's own loopback — reachable here,
// but never beyond it, since the admin API has no auth of its own.
const caddyAdminBaseURL = "http://127.0.0.1:2019"

// deployPayload/unitNamePayload mirror the `payload` shapes documented in
// src/lib/server/agent/protocol.ts — wire-format contract, not to be changed
// unilaterally on either side.
type deployPayload struct {
	UnitName        string        `json:"unitName"`
	UnitContent     string        `json:"unitContent"`
	EnvFileContent  string        `json:"envFileContent"`
	HealthCheckPort int           `json:"healthCheckPort"`
	NetworkName     string        `json:"networkName"`
	Volumes         []volumeMount `json:"volumes"`
}

// volumeMount is one entry of deployPayload.Volumes (task 07) — Name is
// already the real, namespaced Podman volume name (podmanVolumeName() in
// quadlet.ts), ready to hand straight to `podman volume create`.
type volumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
}

type unitNamePayload struct {
	UnitName string `json:"unitName"`
}

type configureProxyPayload struct {
	CaddyfileContent string `json:"caddyfileContent"`
}

type completionPayload struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// executeCommands runs each pending command and reports its outcome back to
// the control plane. One command failing doesn't stop the others — each
// targets an independent unit.
func executeCommands(ctx context.Context, httpClient *http.Client, cfg config, commands []pendingCommand) {
	for _, cmd := range commands {
		err := executeCommand(ctx, cmd)

		status := "succeeded"
		errMsg := ""
		if err != nil {
			status = "failed"
			errMsg = err.Error()
			log.Printf("command %s (%s) failed: %v", cmd.ID, cmd.Type, err)
		} else {
			log.Printf("command %s (%s) succeeded", cmd.ID, cmd.Type)
		}

		if reportErr := reportCompletion(ctx, httpClient, cfg, cmd.ID, status, errMsg); reportErr != nil {
			log.Printf("failed to report completion for command %s: %v", cmd.ID, reportErr)
		}
	}
}

func executeCommand(ctx context.Context, cmd pendingCommand) error {
	if err := checkRootless(ctx); err != nil {
		return fmt.Errorf("refusing to execute %s command: %w", cmd.Type, err)
	}

	switch cmd.Type {
	case "deploy":
		return executeDeploy(ctx, cmd.Payload)
	case "stop":
		return executeUnitAction(ctx, cmd.Payload, "stop")
	case "restart":
		return executeUnitAction(ctx, cmd.Payload, "restart")
	case "configureProxy":
		return executeConfigureProxy(ctx, cmd.Payload)
	default:
		return fmt.Errorf("unknown command type %q", cmd.Type)
	}
}

// executeDeploy writes the unit + env file and starts it. Each step is
// reported as its own failure point (env write vs. unit write vs.
// daemon-reload vs. start) rather than a single generic error, since a
// deliberately broken unit (e.g. an invalid image ref) should fail at
// `start`, not be indistinguishable from a filesystem permissions problem.
func executeDeploy(ctx context.Context, rawPayload json.RawMessage) error {
	var p deployPayload
	if err := json.Unmarshal(rawPayload, &p); err != nil {
		return fmt.Errorf("decode deploy payload: %w", err)
	}
	if p.UnitName == "" {
		return fmt.Errorf("deploy payload missing unitName")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	unitDir := filepath.Join(home, ".config", "containers", "systemd")
	envDir := filepath.Join(home, ".config", "bakery", "env")

	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return fmt.Errorf("create unit dir: %w", err)
	}
	if err := os.MkdirAll(envDir, 0o700); err != nil {
		return fmt.Errorf("create env dir: %w", err)
	}

	// Env file (may contain decrypted secrets) written 600, unit file 644 —
	// only the unit's structure needs to be host-readable, never its values.
	envPath := filepath.Join(envDir, p.UnitName+".env")
	if err := os.WriteFile(envPath, []byte(p.EnvFileContent), 0o600); err != nil {
		return fmt.Errorf("write env file: %w", err)
	}

	unitPath := filepath.Join(unitDir, p.UnitName+".container")
	if err := os.WriteFile(unitPath, []byte(p.UnitContent), 0o644); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}

	// The unit file's own `Network=` line (Phase 05 task 05) doesn't create
	// the network it references — Podman requires it to already exist before
	// a container can join it, unlike the image, which it pulls on demand.
	if p.NetworkName != "" {
		if err := ensureNetwork(ctx, p.NetworkName); err != nil {
			return fmt.Errorf("ensure network: %w", err)
		}
	}

	// Same reasoning as the network above, for each `Volume=` line the unit
	// content references (task 07).
	for _, v := range p.Volumes {
		if err := ensureVolume(ctx, v.Name); err != nil {
			return fmt.Errorf("ensure volume: %w", err)
		}
	}

	if err := runSystemctl(ctx, "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	if err := runSystemctl(ctx, "start", p.UnitName+".service"); err != nil {
		return fmt.Errorf("start %s.service: %w", p.UnitName, err)
	}

	// The control plane's zero-downtime rollover (task 06) treats a `deploy`
	// command's success as proof the new unit is not just started but
	// reachable — it decides whether to stop the old unit based on this
	// outcome alone, so the health probe result has to ride in the same
	// completion report rather than a separate round trip.
	if p.HealthCheckPort > 0 {
		if err := probeHealth(ctx, p.HealthCheckPort); err != nil {
			return fmt.Errorf("unit started but health check failed: %w", err)
		}
	}

	return nil
}

// probeHealth polls the unit's published port until it accepts a connection
// and responds, or healthCheckTimeout elapses. Any HTTP response — even a
// 404/500 — proves the process is up and accepting connections; Bakery
// doesn't assume apps expose a dedicated health endpoint in v1.
//
// Uses the literal loopback IP, not "localhost": on hosts where it resolves
// to ::1 first, rootless Podman's port publishing (pasta) can leave IPv6
// loopback connections hanging/reset even though the port is correctly
// published on IPv4 — matches the same 127.0.0.1 convention `reverse_proxy`
// (Phase 05 task 03) uses, for the same reason.
func probeHealth(ctx context.Context, port int) error {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	client := &http.Client{Timeout: 3 * time.Second}

	var lastErr error
	for {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for %s to become reachable: %w", url, lastErr)
		case <-time.After(healthCheckInterval):
		}
	}
}

// ensureNetwork creates the guild's bridge network (Phase 05 task 05) if it
// doesn't already exist. `--ignore` makes this a single idempotent call
// rather than a check-then-create — two concurrent deploys for the same
// guild racing here would otherwise both see "doesn't exist" and only one
// of the two `create` calls would win.
func ensureNetwork(ctx context.Context, name string) error {
	if _, err := runPodman(ctx, "network", "create", "--ignore", name); err != nil {
		return fmt.Errorf("create network %s: %w", name, err)
	}
	return nil
}

// ensureVolume creates a Bakery-managed named volume (Phase 05 task 07) if
// it doesn't already exist. Same `--ignore` idempotency reasoning as
// ensureNetwork above.
func ensureVolume(ctx context.Context, name string) error {
	if _, err := runPodman(ctx, "volume", "create", "--ignore", name); err != nil {
		return fmt.Errorf("create volume %s: %w", name, err)
	}
	return nil
}

func executeUnitAction(ctx context.Context, rawPayload json.RawMessage, action string) error {
	var p unitNamePayload
	if err := json.Unmarshal(rawPayload, &p); err != nil {
		return fmt.Errorf("decode %s payload: %w", action, err)
	}
	if p.UnitName == "" {
		return fmt.Errorf("%s payload missing unitName", action)
	}

	if err := runSystemctl(ctx, action, p.UnitName+".service"); err != nil {
		return fmt.Errorf("%s %s.service: %w", action, p.UnitName, err)
	}
	return nil
}

// executeConfigureProxy is the real traffic-flip mechanism (Phase 05 task
// 03): it writes the full desired Caddyfile to disk (so a Caddy container
// restart picks up the same config it's running right now, not whatever
// install.sh last wrote) and pushes it to Caddy's admin API for immediate,
// in-place application — no container restart, so requests to routes this
// change doesn't touch are never dropped.
func executeConfigureProxy(ctx context.Context, rawPayload json.RawMessage) error {
	var p configureProxyPayload
	if err := json.Unmarshal(rawPayload, &p); err != nil {
		return fmt.Errorf("decode configureProxy payload: %w", err)
	}
	if p.CaddyfileContent == "" {
		return fmt.Errorf("configureProxy payload missing caddyfileContent")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	caddyfilePath := filepath.Join(home, ".config", "bakery", "caddy", "Caddyfile")
	if err := os.WriteFile(caddyfilePath, []byte(p.CaddyfileContent), 0o644); err != nil {
		return fmt.Errorf("write Caddyfile: %w", err)
	}

	if err := loadCaddyConfig(ctx, p.CaddyfileContent); err != nil {
		return fmt.Errorf("apply Caddy config: %w", err)
	}

	return nil
}

// loadCaddyConfig posts the Caddyfile straight to Caddy's admin API. The
// `text/caddyfile` content type tells Caddy to adapt it via the Caddyfile
// adapter before applying — no need to hand-build Caddy's native JSON config.
func loadCaddyConfig(ctx context.Context, caddyfileContent string) error {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, caddyAdminBaseURL+"/load", strings.NewReader(caddyfileContent),
	)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "text/caddyfile")

	client := &http.Client{Timeout: commandTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("reach Caddy admin API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return fmt.Errorf("Caddy admin API rejected config: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func runSystemctl(ctx context.Context, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", append([]string{"--user"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func reportCompletion(ctx context.Context, httpClient *http.Client, cfg config, commandID, status, errMsg string) error {
	body, err := json.Marshal(completionPayload{Status: status, ErrorMessage: errMsg})
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	endpoint := strings.TrimSuffix(cfg.controlPlaneURL, "/") + "/api/v1/agent/commands/" + commandID + "/complete"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return fmt.Errorf("read response: %w", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("completion rejected: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}
