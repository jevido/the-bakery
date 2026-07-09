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

// deployPayload/unitNamePayload mirror the `payload` shapes documented in
// src/lib/server/agent/protocol.ts — wire-format contract, not to be changed
// unilaterally on either side.
type deployPayload struct {
	UnitName       string `json:"unitName"`
	UnitContent    string `json:"unitContent"`
	EnvFileContent string `json:"envFileContent"`
}

type unitNamePayload struct {
	UnitName string `json:"unitName"`
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
	switch cmd.Type {
	case "deploy":
		return executeDeploy(ctx, cmd.Payload)
	case "stop":
		return executeUnitAction(ctx, cmd.Payload, "stop")
	case "restart":
		return executeUnitAction(ctx, cmd.Payload, "restart")
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

	if err := runSystemctl(ctx, "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	if err := runSystemctl(ctx, "start", p.UnitName+".service"); err != nil {
		return fmt.Errorf("start %s.service: %w", p.UnitName, err)
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
