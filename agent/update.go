package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// updateCheckEveryNCycles throttles version polling to avoid hammering the
// endpoint on every 45s check-in.
const updateCheckEveryNCycles = 10

type releaseInfo struct {
	Version     string `json:"version"`
	DownloadURL string `json:"downloadUrl"`
	SHA256      string `json:"sha256"`
}

// checkForUpdate compares the latest published release against the
// compiled-in agentVersion and self-updates if they differ. Any failure is
// logged and left for the next check — an update hiccup should never take
// the agent down.
func checkForUpdate(ctx context.Context, httpClient *http.Client, cfg config) {
	platform := "linux-" + runtime.GOARCH
	endpoint := strings.TrimSuffix(cfg.controlPlaneURL, "/") + "/api/v1/agent/version?platform=" + platform

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		log.Printf("update check: build request: %v", err)
		return
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("update check: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return // no release published for this platform yet
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("update check: unexpected status %d", resp.StatusCode)
		return
	}

	var release releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Printf("update check: decode response: %v", err)
		return
	}

	if release.Version == "" || release.Version == agentVersion {
		return
	}

	log.Printf("update available: %s -> %s", agentVersion, release.Version)
	if err := applyUpdate(ctx, httpClient, release); err != nil {
		log.Printf("update failed, staying on %s: %v", agentVersion, err)
	}
}

// applyUpdate downloads the new binary, verifies its checksum, atomically
// swaps it into place, and re-execs into it.
func applyUpdate(ctx context.Context, httpClient *http.Client, release releaseInfo) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve executable symlink: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, release.DownloadURL, nil)
	if err != nil {
		return fmt.Errorf("build download request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: unexpected status %d", resp.StatusCode)
	}

	// Same directory as the running binary so the final rename lands on the
	// same filesystem and is therefore atomic.
	tmpFile, err := os.CreateTemp(filepath.Dir(execPath), ".bakery-agent-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // no-op once successfully renamed away

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmpFile, hasher), resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write download: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(sum, release.SHA256) {
		return fmt.Errorf("checksum mismatch: got %s, expected %s", sum, release.SHA256)
	}

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		return fmt.Errorf("swap binary: %w", err)
	}

	log.Printf("updated to %s, re-executing", release.Version)

	// Re-exec in place (same PID) rather than exiting: the install.sh unit
	// sets Restart=on-failure, which does NOT restart a process that exits
	// 0. Re-execing keeps the systemd service continuously "active" with
	// no check-in gap beyond the download time itself, and needs no unit
	// file change.
	if err := syscall.Exec(execPath, os.Args, os.Environ()); err != nil {
		return fmt.Errorf("re-exec: %w", err)
	}
	return nil // unreachable when syscall.Exec succeeds
}
