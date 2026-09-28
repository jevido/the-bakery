package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// RunStatus is where a run stands.
type RunStatus string

const (
	RunRunning   RunStatus = "running"
	RunSucceeded RunStatus = "succeeded"
	RunFailed    RunStatus = "failed"
	RunStopped   RunStatus = "stopped"
	// RunLost is never stored: a run still running after runLostAfter
	// reads as lost, because the app that started it went away.
	RunLost RunStatus = "lost"
)

// runLostAfter is how long a run may run before it reads as lost.
const runLostAfter = 24 * time.Hour

const (
	runSummaryMax = 4000
	runTextMax    = 200
)

var (
	ErrRunAlreadyEnded   = errors.New("this run has already ended")
	ErrNotRunOwner       = errors.New("only the member who started a run can end it")
	ErrInvalidRunStatus  = errors.New("a run ends as succeeded, failed or stopped")
	ErrInvalidRunStats   = errors.New("cost, turns and diff stats cannot be negative")
	ErrInvalidRunAgent   = errors.New("a run needs its agent's id and a name of 1 to 200 characters")
	ErrInvalidRunDetails = errors.New("machine and branch must be at most 200 characters")
	ErrInvalidRunKind    = errors.New("a run is work or plan")
)

// Run kinds: work on the task, or plan it (propose subtasks, change
// nothing).
const (
	RunWork = "work"
	RunPlan = "plan"
)

// Run is one attempt by an agent to work a task on a member's machine. It
// is its own aggregate: it belongs to a task by id, and ends once, only by
// the member who started it.
type Run struct {
	ID        uint64
	TaskID    uint64
	MemberID  uint64
	AgentID   uint64
	AgentName string
	// Kind is RunWork or RunPlan.
	Kind      string
	Machine   string
	Branch    string
	Status    RunStatus
	StartedAt time.Time
	EndedAt   *time.Time
	RunStats
}

// RunStats is what a run reports when it ends.
type RunStats struct {
	CostUSD      float64
	Turns        int
	Summary      string
	FilesChanged int
	Additions    int
	Deletions    int
}

// RunStarted is announced when a member starts a run on a task.
type RunStarted struct {
	RunID     uint64
	TaskID    uint64
	BoardID   uint64
	ActorID   uint64
	AgentID   uint64
	AgentName string
}

// RunFinished is announced when a run ends.
type RunFinished struct {
	RunID     uint64
	TaskID    uint64
	BoardID   uint64
	ActorID   uint64
	AgentName string
	Status    RunStatus
	CostUSD   float64
}

// StartRun makes a running run of the agent on the task by memberID.
func StartRun(taskID, memberID, agentID uint64, agentName, kind, machine, branch string, now time.Time) (Run, error) {
	if kind == "" {
		kind = RunWork
	}
	if kind != RunWork && kind != RunPlan {
		return Run{}, ErrInvalidRunKind
	}
	agentName = strings.TrimSpace(agentName)
	if agentID == 0 || agentName == "" || utf8.RuneCountInString(agentName) > runTextMax {
		return Run{}, ErrInvalidRunAgent
	}
	machine, branch = strings.TrimSpace(machine), strings.TrimSpace(branch)
	if utf8.RuneCountInString(machine) > runTextMax || utf8.RuneCountInString(branch) > runTextMax {
		return Run{}, ErrInvalidRunDetails
	}
	return Run{
		TaskID: taskID, MemberID: memberID, AgentID: agentID, AgentName: agentName, Kind: kind,
		Machine: machine, Branch: branch, Status: RunRunning, StartedAt: now,
	}, nil
}

// Finish ends the run with its outcome. Only the member who started it can,
// and only once.
func (r *Run) Finish(by uint64, status RunStatus, stats RunStats, now time.Time) error {
	if by != r.MemberID {
		return ErrNotRunOwner
	}
	if r.Status != RunRunning {
		return ErrRunAlreadyEnded
	}
	switch status {
	case RunSucceeded, RunFailed, RunStopped:
	default:
		return ErrInvalidRunStatus
	}
	if stats.CostUSD < 0 || stats.Turns < 0 || stats.FilesChanged < 0 || stats.Additions < 0 || stats.Deletions < 0 {
		return ErrInvalidRunStats
	}
	stats.Summary = strings.TrimSpace(stats.Summary)
	if utf8.RuneCountInString(stats.Summary) > runSummaryMax {
		stats.Summary = string([]rune(stats.Summary)[:runSummaryMax])
	}
	r.Status, r.RunStats, r.EndedAt = status, stats, &now
	return nil
}

// StatusAt is the run's status as read at now: a run still running after a
// day is lost.
func (r Run) StatusAt(now time.Time) RunStatus {
	if r.Status == RunRunning && now.Sub(r.StartedAt) > runLostAfter {
		return RunLost
	}
	return r.Status
}
