package workshop

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// Alert kinds, most urgent first when sorted.
const (
	AlertQuestionWaiting = "question_waiting"
	AlertRunFailed       = "run_failed"
	AlertAwaitingReview  = "awaiting_review"
	AlertUncoveredWork   = "uncovered_work"
	AlertIdleAgent       = "idle_agent"
)

const (
	// idleFor is how long an agent may stand idle with work nowhere in
	// reach before it is an alert.
	idleFor = 5 * time.Minute
	// reviewAfter is how long a task may wait in the review column.
	reviewAfter = time.Hour
	// failedFor is how long a failed run stays an alert unless opened.
	failedFor = 24 * time.Hour
)

// Alert is a standing condition that needs a person, shown until the
// condition clears. Its ID stays the same while it lasts (kind and
// target), so the list does not flicker.
type Alert struct {
	ID string `json:"id"`
	// Kind is one of the Alert* kinds.
	Kind string `json:"kind"`
	// Severity is "high" or "normal".
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	// Target is what a click opens: a board (in a guild), and a task, run,
	// letter or agent on it.
	GuildID uint64 `json:"guild_id,omitempty"`
	BoardID uint64 `json:"board_id"`
	TaskID  uint64 `json:"task_id,omitempty"`
	RunID   string `json:"run_id,omitempty"`
	Letter  string `json:"letter,omitempty"`
	Agent   string `json:"agent,omitempty"`
}

// AlertAgent is an agent enabled on a board on this machine.
type AlertAgent struct {
	Slug, Name string
	// IdleSince is when the agent became idle with no task it would take
	// on this board; zero while it works or has work in reach.
	IdleSince time.Time
	// Priorities map a work type key to 1–4; missing is off.
	Priorities map[string]int
}

// AlertTask is a task in a board's ready or review column.
type AlertTask struct {
	ID       uint64
	Title    string
	WorkType string
	// Since is when the task was first seen in the review column.
	Since time.Time
}

// AlertBoard is one board this machine runs agents on.
type AlertBoard struct {
	ID     uint64
	Name   string
	Paused bool
	Agents []AlertAgent
	Ready  []AlertTask
	Review []AlertTask
	// WorkTypes names each of the guild's work types by key.
	WorkTypes map[string]string
}

// AlertRun is a run this machine made.
type AlertRun struct {
	ID, AgentName, TaskTitle string
	BoardID, TaskID          uint64
	Status                   string
	EndedAt                  time.Time
	// Seen is true once someone opened the run after it ended.
	Seen bool
}

// AlertLetter is a letter from a run, waiting for an answer.
type AlertLetter struct {
	ID, RunID, AgentName, TaskTitle string
	BoardID, TaskID                 uint64
	// Kind is "permission" or "question".
	Kind string
}

// AlertState is everything alerts are worked out from.
type AlertState struct {
	Now     time.Time
	Boards  []AlertBoard
	Runs    []AlertRun
	Letters []AlertLetter
}

// Alerts lists what needs attention now, most urgent first. It is pure.
func Alerts(st AlertState) []Alert {
	var out []Alert
	for _, l := range st.Letters {
		what := "has a question"
		if l.Kind == "permission" {
			what = "asks for permission"
		}
		out = append(out, Alert{ID: AlertQuestionWaiting + ":" + l.ID, Kind: AlertQuestionWaiting, Severity: "high",
			Title: l.AgentName + " " + what, Detail: "Waiting on " + quoted(l.TaskTitle) + ". The run is paused until you answer.",
			BoardID: l.BoardID, TaskID: l.TaskID, RunID: l.RunID, Letter: l.ID})
	}
	for _, r := range st.Runs {
		if r.Status != "failed" || r.Seen || r.EndedAt.IsZero() || st.Now.Sub(r.EndedAt) > failedFor {
			continue
		}
		out = append(out, Alert{ID: AlertRunFailed + ":" + r.ID, Kind: AlertRunFailed, Severity: "normal",
			Title: "Run failed", Detail: r.AgentName + " could not finish " + quoted(r.TaskTitle) + ".",
			BoardID: r.BoardID, TaskID: r.TaskID, RunID: r.ID})
	}
	for _, b := range st.Boards {
		for _, t := range b.Review {
			if t.Since.IsZero() || st.Now.Sub(t.Since) < reviewAfter {
				continue
			}
			out = append(out, Alert{ID: fmt.Sprintf("%s:%d", AlertAwaitingReview, t.ID), Kind: AlertAwaitingReview, Severity: "normal",
				Title: "Waiting for review", Detail: quoted(t.Title) + " on " + b.Name + " is done and waiting for a look.",
				BoardID: b.ID, TaskID: t.ID})
		}
		for _, key := range uncovered(b) {
			name := b.WorkTypes[key]
			if name == "" {
				name = key
			}
			out = append(out, Alert{ID: fmt.Sprintf("%s:%d:%s", AlertUncoveredWork, b.ID, key), Kind: AlertUncoveredWork, Severity: "normal",
				Title: "No one on " + name, Detail: "Tasks of " + name + " wait on " + b.Name + ", but no agent here does " + name + " work.",
				BoardID: b.ID})
		}
		if b.Paused {
			continue
		}
		for _, a := range b.Agents {
			if a.IdleSince.IsZero() || st.Now.Sub(a.IdleSince) < idleFor {
				continue
			}
			out = append(out, Alert{ID: fmt.Sprintf("%s:%d:%s", AlertIdleAgent, b.ID, a.Slug), Kind: AlertIdleAgent, Severity: "normal",
				Title: a.Name + " is idle", Detail: "Nothing on " + b.Name + " is work " + a.Name + " would take.",
				BoardID: b.ID, Agent: a.Slug})
		}
	}
	order := []string{AlertQuestionWaiting, AlertRunFailed, AlertAwaitingReview, AlertUncoveredWork, AlertIdleAgent}
	slices.SortStableFunc(out, func(a, b Alert) int { return slices.Index(order, a.Kind) - slices.Index(order, b.Kind) })
	return out
}

// uncovered lists the work types with ready tasks that no agent enabled on
// the board has a priority for, in first-seen order.
func uncovered(b AlertBoard) []string {
	var out []string
	for _, t := range b.Ready {
		if t.WorkType == "" || slices.Contains(out, t.WorkType) {
			continue
		}
		covered := false
		for _, a := range b.Agents {
			if p := a.Priorities[t.WorkType]; p >= 1 && p <= 4 {
				covered = true
				break
			}
		}
		if !covered {
			out = append(out, t.WorkType)
		}
	}
	return out
}

func quoted(title string) string {
	if strings.TrimSpace(title) == "" {
		return "a task"
	}
	return "“" + title + "”"
}
