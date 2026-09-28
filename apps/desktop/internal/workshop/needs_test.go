package workshop

import (
	"testing"
	"time"
)

var needsNow = time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)

func ago(d time.Duration) time.Time { return needsNow.Add(-d) }

func TestBudget(t *testing.T) {
	tests := []struct {
		name  string
		runs  []NeedsRun
		limit float64
		want  int
	}{
		{"no limit is always full", []NeedsRun{{StartedAt: ago(time.Hour), CostUSD: 50}}, 0, 100},
		{"nothing spent", nil, 4, 100},
		{"a quarter spent", []NeedsRun{{StartedAt: ago(time.Hour), CostUSD: 1}}, 4, 75},
		{"yesterday does not count", []NeedsRun{{StartedAt: ago(16 * time.Hour), CostUSD: 3}}, 4, 100},
		{"over the limit is empty", []NeedsRun{{StartedAt: ago(time.Hour), CostUSD: 3}, {StartedAt: ago(2 * time.Hour), CostUSD: 3}}, 4, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeNeeds(NeedsInput{Now: needsNow, Runs: tt.runs, DailyLimitUSD: tt.limit}).Budget; got != tt.want {
				t.Fatalf("budget = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFocus(t *testing.T) {
	tests := []struct {
		name string
		runs []NeedsRun
		want int
	}{
		{"no runs", nil, 100},
		{"unknown usage", []NeedsRun{{StartedAt: ago(time.Hour)}}, 100},
		{"the latest run counts", []NeedsRun{
			{StartedAt: ago(2 * time.Hour), ContextTokens: 190_000, ContextWindow: 200_000},
			{StartedAt: ago(time.Hour), ContextTokens: 50_000, ContextWindow: 200_000},
		}, 75},
		{"nearly full", []NeedsRun{{StartedAt: ago(time.Minute), Status: "running", ContextTokens: 180_000, ContextWindow: 200_000}}, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeNeeds(NeedsInput{Now: needsNow, Runs: tt.runs}).Focus; got != tt.want {
				t.Fatalf("focus = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMorale(t *testing.T) {
	run := func(status string, age time.Duration) NeedsRun {
		return NeedsRun{Status: status, StartedAt: ago(age), EndedAt: ago(age - time.Minute)}
	}
	tests := []struct {
		name string
		runs []NeedsRun
		want int
	}{
		{"a fresh agent", nil, 80},
		{"two stopped runs", []NeedsRun{run("stopped", time.Hour), run("failed", 2*time.Hour)}, 30},
		{"successes lift it, to the top", []NeedsRun{run("succeeded", time.Hour), run("succeeded", 2*time.Hour), run("succeeded", 3*time.Hour)}, 100},
		{"older than a day is forgotten", []NeedsRun{run("failed", 25*time.Hour)}, 80},
		{"it cannot go below empty", []NeedsRun{run("failed", time.Hour), run("failed", 2*time.Hour), run("failed", 3*time.Hour), run("failed", 4*time.Hour)}, 0},
		{"a running run counts for nothing yet", []NeedsRun{{Status: "running", StartedAt: ago(time.Minute)}}, 80},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeNeeds(NeedsInput{Now: needsNow, Runs: tt.runs}).Morale; got != tt.want {
				t.Fatalf("morale = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRest(t *testing.T) {
	tests := []struct {
		name string
		runs []NeedsRun
		want int
	}{
		{"idle all day", nil, 100},
		{"two hours in, still running", []NeedsRun{{Status: "running", StartedAt: ago(2 * time.Hour)}}, 50},
		{"four hours without a break", []NeedsRun{{Status: "running", StartedAt: ago(4 * time.Hour)}}, 0},
		{"short gaps are no break", []NeedsRun{
			{Status: "succeeded", StartedAt: ago(3 * time.Hour), EndedAt: ago(2*time.Hour + 5*time.Minute)},
			{Status: "running", StartedAt: ago(2 * time.Hour)},
		}, 25},
		{"half an hour idle after two hours of work", []NeedsRun{
			{Status: "succeeded", StartedAt: ago(150 * time.Minute), EndedAt: ago(30 * time.Minute)},
		}, 100},
		{"a quarter hour idle after four hours", []NeedsRun{
			{Status: "succeeded", StartedAt: ago(4*time.Hour + 15*time.Minute), EndedAt: ago(15 * time.Minute)},
		}, 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeNeeds(NeedsInput{Now: needsNow, Runs: tt.runs}).Rest; got != tt.want {
				t.Fatalf("rest = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMood(t *testing.T) {
	tests := []struct {
		needs Needs
		want  MoodLevel
	}{
		{Needs{100, 100, 80, 100}, MoodContent},
		{Needs{100, 100, 55, 100}, MoodOkay},
		{Needs{100, 100, 30, 100}, MoodStressed},
		{Needs{60, 50, 30, 40}, MoodStressed},
		{Needs{100, 100, 0, 100}, MoodBreaking},
		{Needs{20, 10, 0, 30}, MoodBreaking},
	}
	for _, tt := range tests {
		if got := Mood(tt.needs); got != tt.want {
			t.Errorf("Mood(%+v) = %s, want %s", tt.needs, got, tt.want)
		}
	}
}

func TestMoodReason(t *testing.T) {
	in := NeedsInput{Now: needsNow, DailyLimitUSD: 4, Runs: []NeedsRun{
		{Status: "stopped", StartedAt: ago(time.Hour), EndedAt: ago(50 * time.Minute)},
		{Status: "stopped", StartedAt: ago(2 * time.Hour), EndedAt: ago(110 * time.Minute)},
	}}
	n := ComputeNeeds(in)
	if got := MoodReason(in, n); got != "two failed or stopped runs in the last day" {
		t.Fatalf("reason = %q", got)
	}
	if got := MoodReason(NeedsInput{Now: needsNow}, ComputeNeeds(NeedsInput{Now: needsNow})); got != "" {
		t.Fatalf("a content agent has a reason: %q", got)
	}
}

func TestContextWindow(t *testing.T) {
	if ContextWindow("claude-opus-5-5[1m]") != 1_000_000 || ContextWindow("haiku") != 200_000 {
		t.Fatal("wrong context window")
	}
}
