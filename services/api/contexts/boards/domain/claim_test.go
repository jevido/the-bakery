package domain

import (
	"errors"
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }

func TestClaim(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	task := Task{ID: 1, BoardID: 2}
	held := &Claim{ID: 5, TaskID: 1, AgentID: 3, MemberID: 7, ExpiresAt: now.Add(time.Minute)}
	expired := &Claim{ID: 5, TaskID: 1, AgentID: 3, MemberID: 7, ExpiresAt: now.Add(-time.Second)}
	released := &Claim{ID: 5, TaskID: 1, AgentID: 3, MemberID: 7, ExpiresAt: now.Add(time.Minute), ReleasedAt: &now}
	tests := []struct {
		name    string
		task    Task
		agent   uint64
		machine string
		current *Claim
		want    error
	}{
		{"free", task, 3, "m1", nil, nil},
		{"held", task, 4, "m1", held, ErrTaskClaimed},
		{"expired frees it", task, 4, "m1", expired, nil},
		{"released frees it", task, 4, "m1", released, nil},
		{"forbidden", Task{ID: 1, Forbidden: true}, 3, "m1", nil, ErrTaskForbidden},
		{"prioritized for another", Task{ID: 1, PrioritizedAgentID: ptr(uint64(9))}, 3, "m1", nil, ErrPrioritizedForOther},
		{"prioritized for this one", Task{ID: 1, PrioritizedAgentID: ptr(uint64(3))}, 3, "m1", nil, nil},
		{"subtask", Task{ID: 1, ParentID: ptr(uint64(8))}, 3, "m1", nil, ErrClaimSubtask},
		{"no machine", task, 3, "  ", nil, ErrInvalidMachineID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := tt.task.Claim(tt.agent, 7, tt.machine, tt.current, now)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if err == nil && (!c.ActiveAt(now) || c.ExpiresAt != now.Add(ClaimTTL) || c.AgentID != tt.agent) {
				t.Fatalf("claim = %+v", c)
			}
		})
	}
}

func TestHeartbeatAndRelease(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	c, _ := Task{ID: 1}.Claim(3, 7, "m1", nil, now)
	later := now.Add(90 * time.Second)
	if err := c.Heartbeat(8, later); !errors.Is(err, ErrNotClaimant) {
		t.Fatalf("someone else's heartbeat: %v", err)
	}
	if err := c.Heartbeat(7, later); err != nil || c.ExpiresAt != later.Add(ClaimTTL) {
		t.Fatalf("heartbeat: %v, expires %v", err, c.ExpiresAt)
	}
	if err := c.Heartbeat(7, later.Add(ClaimTTL+time.Second)); !errors.Is(err, ErrClaimEnded) {
		t.Fatalf("heartbeat after expiry: %v", err)
	}
	if _, err := c.Release(8, later); !errors.Is(err, ErrNotClaimant) {
		t.Fatalf("someone else's release: %v", err)
	}
	if changed, err := c.Release(7, later); err != nil || !changed || c.ActiveAt(later) {
		t.Fatalf("release: %v %v", changed, err)
	}
	if changed, _ := c.Release(7, later); changed {
		t.Fatal("releasing twice changed something")
	}
	if err := c.Heartbeat(7, later); !errors.Is(err, ErrClaimEnded) {
		t.Fatalf("heartbeat after release: %v", err)
	}
}

func TestDraft(t *testing.T) {
	task := Task{ID: 1}
	if changed, err := task.Draft(ptr(uint64(3)), nil); err != nil || !changed || *task.PrioritizedAgentID != 3 {
		t.Fatalf("prioritize: %v %v", changed, err)
	}
	if changed, _ := task.Draft(ptr(uint64(3)), nil); changed {
		t.Fatal("prioritizing the same agent again changed something")
	}
	if changed, _ := task.Draft(ptr(uint64(0)), ptr(true)); !changed || task.PrioritizedAgentID != nil || !task.Forbidden {
		t.Fatalf("clear and forbid: %+v", task)
	}
	sub := Task{ID: 2, ParentID: ptr(uint64(1))}
	if _, err := sub.Draft(nil, ptr(true)); !errors.Is(err, ErrDraftSubtask) {
		t.Fatalf("forbid a subtask: %v", err)
	}
}
