package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// volumeNamePrefix matches podmanVolumeName() in quadlet.ts — only
// Bakery-managed volumes are reported, not arbitrary volumes a host
// operator created by hand.
const volumeNamePrefix = "bakery-vol-"

// volumeReport mirrors volumeReportSchema in src/lib/server/agent/protocol.ts.
type volumeReport struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"sizeBytes"`
}

// collectVolumeReports lists every Bakery-managed Podman volume on this host
// and measures its on-disk size, for inclusion in the check-in payload
// (Phase 05 task 07) — replacing the control plane's mock `Volume.size`
// field with real, host-reported usage. Returns an empty slice (never an
// error) on any failure, matching collectPayload's per-metric resilience:
// a host mid-setup shouldn't fail its whole check-in over volume reporting.
func collectVolumeReports(ctx context.Context) []volumeReport {
	// Never nil: a nil slice marshals to JSON `null`, which the control
	// plane's schema (an array with a *default*, not nullable) would reject
	// outright, failing the whole check-in over an unrelated metric.
	reports := []volumeReport{}

	out, err := runPodman(ctx, "volume", "ls", "--format", "json")
	if err != nil {
		return reports
	}

	var vols []struct {
		Name       string `json:"Name"`
		Mountpoint string `json:"Mountpoint"`
	}
	if err := json.Unmarshal(out, &vols); err != nil {
		return reports
	}

	for _, v := range vols {
		if !strings.HasPrefix(v.Name, volumeNamePrefix) || v.Mountpoint == "" {
			continue
		}

		size, err := dirSizeBytes(v.Mountpoint)
		if err != nil {
			continue
		}

		reports = append(reports, volumeReport{Name: v.Name, SizeBytes: size})
	}

	return reports
}

// dirSizeBytes sums the size of every regular file under path. Used instead
// of shelling out to `du` to avoid depending on a binary that isn't
// guaranteed to be present on every distro the agent runs on.
func dirSizeBytes(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walk %s: %w", path, err)
	}
	return total, nil
}
