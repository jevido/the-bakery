package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const podmanTimeout = 5 * time.Second

type podmanMetrics struct {
	version        string
	containerCount int
}

// collectPodmanMetrics shells out to podman. Podman may not be installed
// yet at initial agent bring-up (or the host may be mid-setup), so any
// failure here is returned as an error for the caller to log and fall back
// on rather than crashing the agent.
func collectPodmanMetrics(ctx context.Context) (podmanMetrics, error) {
	version, err := podmanVersion(ctx)
	if err != nil {
		return podmanMetrics{}, fmt.Errorf("podman version: %w", err)
	}

	count, err := podmanRunningContainerCount(ctx)
	if err != nil {
		return podmanMetrics{}, fmt.Errorf("podman container count: %w", err)
	}

	return podmanMetrics{version: version, containerCount: count}, nil
}

func podmanVersion(ctx context.Context) (string, error) {
	out, err := runPodman(ctx, "info", "--format", "json")
	if err != nil {
		return "", err
	}

	// Same shape under both rootful and rootless invocations.
	var info struct {
		Version struct {
			Version string `json:"Version"`
		} `json:"version"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return "", fmt.Errorf("parse podman info: %w", err)
	}
	if info.Version.Version == "" {
		return "", fmt.Errorf("podman info did not report a version")
	}

	return info.Version.Version, nil
}

// podmanRunningContainerCount uses `podman stats --no-stream`, which (unlike
// `podman ps -a`) only reports currently-running containers, so its entry
// count doubles as the running-container count.
func podmanRunningContainerCount(ctx context.Context) (int, error) {
	out, err := runPodman(ctx, "stats", "--no-stream", "--format", "json")
	if err != nil {
		return 0, err
	}

	var stats []json.RawMessage
	if err := json.Unmarshal(out, &stats); err != nil {
		return 0, fmt.Errorf("parse podman stats: %w", err)
	}

	return len(stats), nil
}

func runPodman(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, podmanTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "podman", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("podman %s: %w: %s", args[0], err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("podman %s: %w", args[0], err)
	}

	return out, nil
}
