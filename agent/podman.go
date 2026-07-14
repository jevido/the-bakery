package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const podmanTimeout = 5 * time.Second

// systemdContainerPrefix is the default Podman container name Quadlet
// assigns a unit that doesn't set ContainerName= explicitly (`systemd-%N`,
// per podman-systemd.unit(5)) — matches `versionedUnitName` in quadlet.ts,
// which every Bakery-deployed unit goes through.
const systemdContainerPrefix = "systemd-"

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

// containerStat mirrors containerStatSchema in
// src/lib/server/agent/protocol.ts.
type containerStat struct {
	UnitName string  `json:"unitName"`
	CPUPct   float64 `json:"cpuPct"`
	MemBytes int64   `json:"memBytes"`
}

// collectContainerStats reports per-container CPU/memory usage for every
// Quadlet-managed (i.e. `systemd-`-prefixed) running container, for
// inclusion in the check-in payload (task 01) — replacing the app detail
// page's mock `wobble()` charts with real per-app data. Containers not
// managed by Quadlet (a host operator's own, or Bakery infra like Caddy
// itself) are skipped since they can't be mapped back to an app. Returns an
// empty slice (never an error) on any failure, matching
// collectVolumeReports's per-metric resilience.
//
// Uses `--format "{{json .}}"` rather than `--format json`: the latter's
// JSON mode renders CPU/memory as human-formatted strings ("31.98MB / 67.12GB"),
// while the Go-template JSON encoder exposes the underlying raw numeric
// fields (CPU as a 0-100 percentage, MemUsage in bytes) needed here.
func collectContainerStats(ctx context.Context) []containerStat {
	stats := []containerStat{}

	out, err := runPodman(ctx, "stats", "--no-stream", "--format", "{{json .}}")
	if err != nil {
		return stats
	}

	// One JSON object per running container, not wrapped in an array — a
	// streaming decoder reads each value off in turn regardless of whether
	// they're newline- or otherwise-separated.
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var raw struct {
			Name     string  `json:"Name"`
			CPU      float64 `json:"CPU"`
			MemUsage int64   `json:"MemUsage"`
		}
		if err := dec.Decode(&raw); err != nil {
			break
		}

		unitName, ok := strings.CutPrefix(raw.Name, systemdContainerPrefix)
		if !ok {
			continue
		}

		stats = append(stats, containerStat{
			UnitName: unitName,
			CPUPct:   clampPct(raw.CPU),
			MemBytes: raw.MemUsage,
		})
	}

	return stats
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
