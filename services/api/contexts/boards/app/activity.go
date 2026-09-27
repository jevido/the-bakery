package app

import (
	"context"
	"time"
)

// ActivityKind is what happened to a task.
type ActivityKind string

const (
	ActivityCreated      ActivityKind = "created"
	ActivityEdited       ActivityKind = "edited"
	ActivityMoved        ActivityKind = "moved"
	ActivityCommented    ActivityKind = "commented"
	ActivitySubtaskAdded ActivityKind = "subtask_added"
	ActivitySubtaskDone  ActivityKind = "subtask_done"
)

// Activity is one entry of a task's history. It is a projection of domain
// events: the activity projector writes it, nobody else.
type Activity struct {
	ID        uint64
	TaskID    uint64
	Kind      ActivityKind
	ActorID   uint64
	ActorName string
	At        time.Time
	// Data depends on Kind, e.g. {"from": "todo", "to": "doing"} for moved.
	Data map[string]any
}

type ActivityLog interface {
	// OfTask returns up to limit entries of the task, newest first, older
	// than the entry before (0: from the newest).
	OfTask(ctx context.Context, taskID, before uint64, limit int) ([]Activity, error)
}

const (
	activityPage    = 50
	activityPageMax = 100
)

// ListActivity returns a task's history, newest first, a page at a time.
func (s *Service) ListActivity(ctx context.Context, taskID, memberID, before uint64, limit int) ([]Activity, error) {
	t, err := s.readableTask(ctx, taskID, memberID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = activityPage
	}
	limit = min(limit, activityPageMax)
	entries, err := s.activity.OfTask(ctx, t.ID, before, limit)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, len(entries))
	for i, e := range entries {
		ids[i] = e.ActorID
	}
	names, err := s.names.DisplayNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		entries[i].ActorName = names[entries[i].ActorID]
	}
	return entries, nil
}
