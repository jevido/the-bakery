package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func startedRun(t *testing.T) Run {
	t.Helper()
	r, err := StartRun(1, 7, 3, "Vera", "ada-laptop", "bakery/1-bench", t0)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestStartRun(t *testing.T) {
	r := startedRun(t)
	if r.Status != RunRunning || r.EndedAt != nil || r.AgentName != "Vera" {
		t.Fatalf("run = %+v", r)
	}
	tests := []struct {
		name            string
		agentID         uint64
		agentName, host string
		want            error
	}{
		{"no agent id", 0, "Vera", "", ErrInvalidRunAgent},
		{"no agent name", 3, "  ", "", ErrInvalidRunAgent},
		{"long machine", 3, "Vera", strings.Repeat("x", 201), ErrInvalidRunDetails},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := StartRun(1, 7, tt.agentID, tt.agentName, tt.host, "", t0); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestFinishRun(t *testing.T) {
	stats := RunStats{CostUSD: 0.42, Turns: 5, Summary: "  Walls up.  ", FilesChanged: 2, Additions: 10, Deletions: 1}
	tests := []struct {
		name   string
		by     uint64
		status RunStatus
		stats  RunStats
		want   error
	}{
		{"succeeded", 7, RunSucceeded, stats, nil},
		{"stopped", 7, RunStopped, RunStats{}, nil},
		{"failed", 7, RunFailed, RunStats{}, nil},
		{"someone else", 8, RunSucceeded, stats, ErrNotRunOwner},
		{"still running", 7, RunRunning, stats, ErrInvalidRunStatus},
		{"lost is not an end", 7, RunLost, stats, ErrInvalidRunStatus},
		{"made-up status", 7, "exploded", stats, ErrInvalidRunStatus},
		{"negative cost", 7, RunSucceeded, RunStats{CostUSD: -1}, ErrInvalidRunStats},
		{"negative lines", 7, RunSucceeded, RunStats{Deletions: -1}, ErrInvalidRunStats},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := startedRun(t)
			err := r.Finish(tt.by, tt.status, tt.stats, t0.Add(time.Minute))
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if err == nil && (r.Status != tt.status || r.EndedAt == nil) {
				t.Fatalf("run = %+v", r)
			}
		})
	}
}

func TestRunEndsOnce(t *testing.T) {
	r := startedRun(t)
	if err := r.Finish(7, RunSucceeded, RunStats{Summary: "  done  "}, t0); err != nil {
		t.Fatal(err)
	}
	if r.Summary != "done" {
		t.Errorf("summary = %q", r.Summary)
	}
	if err := r.Finish(7, RunFailed, RunStats{}, t0); !errors.Is(err, ErrRunAlreadyEnded) {
		t.Fatalf("second finish: %v", err)
	}
}

func TestRunSummaryIsCapped(t *testing.T) {
	r := startedRun(t)
	if err := r.Finish(7, RunSucceeded, RunStats{Summary: strings.Repeat("é", 5000)}, t0); err != nil {
		t.Fatal(err)
	}
	if n := len([]rune(r.Summary)); n != runSummaryMax {
		t.Fatalf("summary has %d characters", n)
	}
}

func TestRunReadsAsLost(t *testing.T) {
	r := startedRun(t)
	if got := r.StatusAt(t0.Add(23 * time.Hour)); got != RunRunning {
		t.Errorf("after 23h: %s", got)
	}
	if got := r.StatusAt(t0.Add(25 * time.Hour)); got != RunLost {
		t.Errorf("after 25h: %s", got)
	}
	_ = r.Finish(7, RunStopped, RunStats{}, t0.Add(time.Hour))
	if got := r.StatusAt(t0.Add(48 * time.Hour)); got != RunStopped {
		t.Errorf("an ended run reads as %s", got)
	}
}
