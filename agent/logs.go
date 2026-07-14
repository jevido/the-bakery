package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// maxLogLinesPerContainer bounds how many new lines one container can
// contribute to a single check-in — a chatty container shouldn't be able to
// bloat every check-in payload unboundedly (task 05's explicit concern).
// If a container produced more than this many lines since the last read,
// only the most recent ones are kept; the rest are permanently skipped,
// same trade-off a real live-tail viewer (not a full log archive) implies.
const maxLogLinesPerContainer = 200

// firstReadLookback bounds how far back the very first log read for a
// newly-seen container reaches — without it, a container that's been
// running for days would dump its entire backlog into one check-in the
// first time the agent sees it (e.g. right after an agent restart).
const firstReadLookback = 60 * time.Second

// logLineReport mirrors logLineReportSchema in
// src/lib/server/agent/protocol.ts.
type logLineReport struct {
	UnitName string `json:"unitName"`
	Message  string `json:"message"`
}

// lastLogRead tracks, per unit name, the timestamp up to which its log
// output has already been reported — an in-memory cursor (not persisted)
// so `podman logs --since` only returns lines produced since the previous
// check-in, not the container's entire history each time. Resets to
// firstReadLookback on agent restart, which is an acceptable gap for a v1
// live-tail viewer rather than an exactly-once log archive.
var (
	lastLogReadMu sync.Mutex
	lastLogRead   = map[string]time.Time{}
)

// collectLogLines reports new output for every currently-running,
// Quadlet-managed container reported in containerStats, for inclusion in
// the check-in payload — replacing the app detail page's static LOG_LINES
// mock with real per-app log data. Returns an empty slice (never an error)
// on any single container's failure, same per-metric resilience as
// collectContainerStats/collectVolumeReports.
func collectLogLines(ctx context.Context, containerStats []containerStat) []logLineReport {
	reports := []logLineReport{}
	now := time.Now()

	for _, c := range containerStats {
		lastLogReadMu.Lock()
		since, seenBefore := lastLogRead[c.UnitName]
		lastLogReadMu.Unlock()
		if !seenBefore {
			since = now.Add(-firstReadLookback)
		}

		lines, err := podmanLogsSince(ctx, systemdContainerPrefix+c.UnitName, since)
		if err == nil {
			if len(lines) > maxLogLinesPerContainer {
				lines = lines[len(lines)-maxLogLinesPerContainer:]
			}
			for _, line := range lines {
				reports = append(reports, logLineReport{UnitName: c.UnitName, Message: line})
			}
		}

		lastLogReadMu.Lock()
		lastLogRead[c.UnitName] = now
		lastLogReadMu.Unlock()
	}

	return reports
}

// podmanLogsSince returns each new line of a container's combined
// stdout+stderr output since the given time. Uses CombinedOutput rather
// than runPodman (stdout-only): `podman logs` writes container stdout to
// its own stdout and container stderr to its own stderr (Docker-compatible
// demuxing), and both streams matter for a runtime log viewer.
func podmanLogsSince(ctx context.Context, containerName string, since time.Time) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, podmanTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "podman", "logs", "--since", since.Format(time.RFC3339Nano), containerName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("podman logs %s: %w", containerName, err)
	}

	raw := strings.TrimRight(string(out), "\n")
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}
