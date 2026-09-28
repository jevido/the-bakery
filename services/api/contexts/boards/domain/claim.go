package domain

import (
	"errors"
	"strings"
	"time"
)

// ClaimTTL is how long a claim lasts without a heartbeat. The desktop
// heartbeats every 30 seconds.
const ClaimTTL = 2 * time.Minute

const machineIDMax = 100

var (
	ErrTaskClaimed         = errors.New("an agent is already working on this task")
	ErrTaskForbidden       = errors.New("this task is forbidden for agents")
	ErrPrioritizedForOther = errors.New("this task is prioritized for another agent")
	ErrClaimSubtask        = errors.New("agents work tasks on the board, not subtasks")
	ErrInvalidMachineID    = errors.New("machine_id must be 1 to 100 characters")
	ErrNotClaimant         = errors.New("only the member who claimed a task can change the claim")
	ErrClaimEnded          = errors.New("this claim has expired or was released")
	ErrDraftSubtask        = errors.New("a subtask is worked with its task; prioritize or forbid the task")
)

// Claim is a time-limited hold a member's agent has on a task, so no other
// machine starts it. It belongs to the Task aggregate: a task has at most
// one active claim.
type Claim struct {
	ID         uint64
	TaskID     uint64
	AgentID    uint64
	MemberID   uint64
	MachineID  string
	ClaimedAt  time.Time
	ExpiresAt  time.Time
	ReleasedAt *time.Time
}

// ActiveAt reports whether the claim still holds the task at now.
func (c Claim) ActiveAt(now time.Time) bool {
	return c.ReleasedAt == nil && now.Before(c.ExpiresAt)
}

// TaskClaimed is announced when an agent takes a task.
type TaskClaimed struct {
	TaskID    uint64
	BoardID   uint64
	ActorID   uint64
	ClaimID   uint64
	AgentID   uint64
	ExpiresAt time.Time
}

// TaskReleased is announced when a claim is let go.
type TaskReleased struct {
	TaskID  uint64
	BoardID uint64
	ActorID uint64
	ClaimID uint64
}

// Claim makes a claim on the task for the member's agent from one machine.
// current is the task's latest unreleased claim, if any; an expired one no
// longer holds the task. The caller has made sure the agent is the member's.
func (t Task) Claim(agentID, memberID uint64, machineID string, current *Claim, now time.Time) (Claim, error) {
	machineID = strings.TrimSpace(machineID)
	switch {
	case t.IsSubtask():
		return Claim{}, ErrClaimSubtask
	case machineID == "" || len(machineID) > machineIDMax:
		return Claim{}, ErrInvalidMachineID
	case t.Forbidden:
		return Claim{}, ErrTaskForbidden
	case t.PrioritizedAgentID != nil && *t.PrioritizedAgentID != agentID:
		return Claim{}, ErrPrioritizedForOther
	case current != nil && current.ActiveAt(now):
		return Claim{}, ErrTaskClaimed
	}
	return Claim{TaskID: t.ID, AgentID: agentID, MemberID: memberID, MachineID: machineID, ClaimedAt: now, ExpiresAt: now.Add(ClaimTTL)}, nil
}

// Heartbeat extends a claim that still holds its task.
func (c *Claim) Heartbeat(by uint64, now time.Time) error {
	if by != c.MemberID {
		return ErrNotClaimant
	}
	if !c.ActiveAt(now) {
		return ErrClaimEnded
	}
	c.ExpiresAt = now.Add(ClaimTTL)
	return nil
}

// Release lets the task go. Releasing twice changes nothing; changed is
// false then.
func (c *Claim) Release(by uint64, now time.Time) (changed bool, err error) {
	if by != c.MemberID {
		return false, ErrNotClaimant
	}
	if c.ReleasedAt != nil {
		return false, nil
	}
	c.ReleasedAt = &now
	return true, nil
}

// Draft prioritizes the task for an agent (0 clears it) and/or forbids it
// for agents (nil leaves either as it is). It reports what changed.
func (t *Task) Draft(prioritized *uint64, forbidden *bool) (changed bool, err error) {
	if t.IsSubtask() && (prioritized != nil || forbidden != nil) {
		return false, ErrDraftSubtask
	}
	if prioritized != nil {
		var next *uint64
		if *prioritized != 0 {
			v := *prioritized
			next = &v
		}
		if (t.PrioritizedAgentID == nil) != (next == nil) || (next != nil && *t.PrioritizedAgentID != *next) {
			changed = true
		}
		t.PrioritizedAgentID = next
	}
	if forbidden != nil && t.Forbidden != *forbidden {
		t.Forbidden = *forbidden
		changed = true
	}
	return changed, nil
}
