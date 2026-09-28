package workshop

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// NeedsRun is one run of an agent, as its needs see it.
type NeedsRun struct {
	// Status is "running", "succeeded", "failed" or "stopped".
	Status    string
	StartedAt time.Time
	// EndedAt is zero while the run goes.
	EndedAt time.Time
	CostUSD float64
	// ContextTokens is how much of the model's context the run's last turn
	// used; ContextWindow how much the model has. Zero when unknown.
	ContextTokens, ContextWindow int
}

// NeedsInput is what an agent's needs are computed from: its runs on this
// machine (any order, any age) and what it may spend in a day.
type NeedsInput struct {
	Now           time.Time
	Runs          []NeedsRun
	DailyLimitUSD float64
}

// Needs are an agent's four bars, each 0 (empty) to 100 (full).
type Needs struct {
	Budget int `json:"budget"`
	Focus  int `json:"focus"`
	Morale int `json:"morale"`
	Rest   int `json:"rest"`
}

// MoodLevel sums up the needs in one word.
type MoodLevel string

const (
	MoodContent   MoodLevel = "content"
	MoodOkay      MoodLevel = "okay"
	MoodStressed  MoodLevel = "stressed"
	MoodBreaking  MoodLevel = "breaking"
	moraleStart             = 80
	moraleFailure           = 25
	moraleSuccess           = 10
	// Rest runs out after four hours of work without a break, and comes
	// back in one idle hour. Runs less than breakGap apart are one stretch.
	restDrain = 4 * time.Hour
	restFill  = time.Hour
	breakGap  = 10 * time.Minute
)

// ComputeNeeds works out an agent's needs. It is pure: the same input gives
// the same needs.
func ComputeNeeds(in NeedsInput) Needs {
	return Needs{Budget: budget(in), Focus: focus(in), Morale: morale(in), Rest: rest(in)}
}

func clamp(v float64) int {
	return int(max(0, min(100, v+0.5)))
}

// today are the runs started since midnight of in.Now's day.
func today(in NeedsInput) []NeedsRun {
	y, m, d := in.Now.Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, in.Now.Location())
	var out []NeedsRun
	for _, r := range in.Runs {
		if !r.StartedAt.Before(midnight) {
			out = append(out, r)
		}
	}
	return out
}

func spentToday(in NeedsInput) float64 {
	var spent float64
	for _, r := range today(in) {
		spent += r.CostUSD
	}
	return spent
}

// budget is the money left today.
func budget(in NeedsInput) int {
	if in.DailyLimitUSD <= 0 {
		return 100
	}
	return clamp(100 - spentToday(in)/in.DailyLimitUSD*100)
}

// latest is the run started last, if any.
func latest(runs []NeedsRun) (NeedsRun, bool) {
	if len(runs) == 0 {
		return NeedsRun{}, false
	}
	return slices.MaxFunc(runs, func(a, b NeedsRun) int { return a.StartedAt.Compare(b.StartedAt) }), true
}

// focus is the context left in the current or last run.
func focus(in NeedsInput) int {
	r, ok := latest(in.Runs)
	if !ok || r.ContextWindow <= 0 || r.ContextTokens <= 0 {
		return 100
	}
	return clamp(100 - float64(r.ContextTokens)/float64(r.ContextWindow)*100)
}

// recentOutcomes counts the last day's failed or stopped runs and its
// successes.
func recentOutcomes(in NeedsInput) (bad, good int) {
	since := in.Now.Add(-24 * time.Hour)
	for _, r := range in.Runs {
		if r.StartedAt.Before(since) {
			continue
		}
		switch r.Status {
		case "failed", "stopped":
			bad++
		case "succeeded":
			good++
		}
	}
	return bad, good
}

// morale starts at 80, drops 25 for every failed or stopped run of the last
// day and rises 10 for every success.
func morale(in NeedsInput) int {
	bad, good := recentOutcomes(in)
	return clamp(moraleStart - float64(bad*moraleFailure) + float64(good*moraleSuccess))
}

// stretches merges the last day's runs into working stretches: runs less
// than breakGap apart count as one.
func stretches(in NeedsInput) [][2]time.Time {
	since := in.Now.Add(-24 * time.Hour)
	var spans [][2]time.Time
	for _, r := range in.Runs {
		end := r.EndedAt
		if end.IsZero() || r.Status == "running" {
			end = in.Now
		}
		if end.Before(since) {
			continue
		}
		spans = append(spans, [2]time.Time{maxTime(r.StartedAt, since), minTime(end, in.Now)})
	}
	slices.SortFunc(spans, func(a, b [2]time.Time) int { return a[0].Compare(b[0]) })
	var out [][2]time.Time
	for _, s := range spans {
		if n := len(out); n > 0 && s[0].Sub(out[n-1][1]) < breakGap {
			out[n-1][1] = maxTime(out[n-1][1], s[1])
			continue
		}
		out = append(out, s)
	}
	return out
}

// rest drains while the agent works and fills while it is idle, over the
// last day.
func rest(in NeedsInput) int {
	level := 100.0
	at := in.Now.Add(-24 * time.Hour)
	for _, s := range stretches(in) {
		level = min(100, level+float64(s[0].Sub(at))/float64(restFill)*100)
		level = max(0, level-float64(s[1].Sub(s[0]))/float64(restDrain)*100)
		at = s[1]
	}
	level = min(100, level+float64(in.Now.Sub(at))/float64(restFill)*100)
	return clamp(level)
}

// Mood sums the needs up. The lowest need weighs most (three quarters),
// the average of all four the rest: one empty bar is enough to stress an
// agent, as one bad need is in a colony.
func Mood(n Needs) MoodLevel {
	lowest := float64(min(n.Budget, n.Focus, n.Morale, n.Rest))
	avg := float64(n.Budget+n.Focus+n.Morale+n.Rest) / 4
	score := 0.75*lowest + 0.25*avg
	switch {
	case score >= 70:
		return MoodContent
	case score >= 45:
		return MoodOkay
	case score >= 20:
		return MoodStressed
	}
	return MoodBreaking
}

// MoodReason says in one line why the agent feels as it does: the lowest
// need, in words. Empty when every need is full enough not to matter.
func MoodReason(in NeedsInput, n Needs) string {
	type need struct {
		value int
		why   string
	}
	bad, _ := recentOutcomes(in)
	needs := []need{
		{n.Morale, fmt.Sprintf("%s in the last day", runsWord(bad, "failed or stopped run"))},
		{n.Budget, fmt.Sprintf("spent $%.2f of $%.2f today", spentToday(in), in.DailyLimitUSD)},
		{n.Focus, "its context is filling up"},
		{n.Rest, "working without a break"},
	}
	lowest := slices.MinFunc(needs, func(a, b need) int { return a.value - b.value })
	if lowest.value >= 70 {
		return ""
	}
	return lowest.why
}

func runsWord(n int, what string) string {
	words := []string{"no", "one", "two", "three", "four", "five"}
	count := fmt.Sprint(n)
	if n < len(words) {
		count = words[n]
	}
	if n != 1 {
		what = strings.Replace(what, "run", "runs", 1)
	}
	return count + " " + what
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// ContextWindow is how many tokens a model can hold: a million for the
// "[1m]" variants, else 200,000.
func ContextWindow(model string) int {
	if strings.HasSuffix(model, "[1m]") {
		return 1_000_000
	}
	return 200_000
}
