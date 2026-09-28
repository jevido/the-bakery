package infra

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

// notifyChannel is the Postgres channel board events travel on, from the
// API process that made a change to every API process with a stream open.
const notifyChannel = "board_events"

// BoardEvent is the published language of the board event stream (see
// docs/domain/contexts/boards/README.md). Changing a type or removing a
// field breaks the desktop app.
type BoardEvent struct {
	Type    string         `json:"type"`
	BoardID uint64         `json:"board_id"`
	At      time.Time      `json:"at"`
	ActorID uint64         `json:"actor_id"`
	Data    map[string]any `json:"data"`
}

// boardEventOf translates a domain event into a board event. Changes to a
// subtask or a comment are a change to the task on the board. ok is false
// for events the stream does not carry.
func boardEventOf(event any) (ev BoardEvent, ok bool) {
	updated := func(taskID, boardID, actorID uint64, data map[string]any) (BoardEvent, bool) {
		data["task_id"] = taskID
		return BoardEvent{Type: "task.updated", BoardID: boardID, ActorID: actorID, Data: data}, true
	}
	switch e := event.(type) {
	case domain.TaskCreated:
		return BoardEvent{Type: "task.created", BoardID: e.BoardID, ActorID: e.ActorID,
			Data: map[string]any{"task_id": e.TaskID, "column_id": e.ColumnID, "position": e.Position}}, true
	case domain.TaskEdited:
		if e.ParentID != nil {
			return updated(*e.ParentID, e.BoardID, e.ActorID, map[string]any{"subtask_id": e.TaskID})
		}
		return updated(e.TaskID, e.BoardID, e.ActorID, map[string]any{"title": e.Title, "description": e.Description, "work_type": e.WorkType, "drafting": e.Drafting})
	case domain.TaskMoved:
		return BoardEvent{Type: "task.moved", BoardID: e.BoardID, ActorID: e.ActorID,
			Data: map[string]any{"task_id": e.TaskID, "from": e.From, "to": e.To, "position": e.Position}}, true
	case domain.TaskDeleted:
		if e.ParentID != nil {
			return updated(*e.ParentID, e.BoardID, e.ActorID, map[string]any{"subtask_id": e.TaskID})
		}
		return BoardEvent{Type: "task.deleted", BoardID: e.BoardID, ActorID: e.ActorID,
			Data: map[string]any{"task_id": e.TaskID}}, true
	case domain.TaskCommented:
		return updated(e.TaskID, e.BoardID, e.ActorID, map[string]any{"comment_id": e.CommentID})
	case domain.CommentChanged:
		return updated(e.TaskID, e.BoardID, e.ActorID, map[string]any{"comment_id": e.CommentID})
	case domain.SubtaskAdded:
		return updated(e.ParentID, e.BoardID, e.ActorID, map[string]any{"subtask_id": e.SubtaskID})
	case domain.SubtaskCompleted:
		return updated(e.ParentID, e.BoardID, e.ActorID, map[string]any{"subtask_id": e.SubtaskID})
	case domain.SubtaskReopened:
		return updated(e.ParentID, e.BoardID, e.ActorID, map[string]any{"subtask_id": e.SubtaskID})
	case domain.SubtaskMoved:
		return updated(e.ParentID, e.BoardID, e.ActorID, map[string]any{"subtask_id": e.SubtaskID})
	case domain.TaskClaimed:
		return BoardEvent{Type: "task.claimed", BoardID: e.BoardID, ActorID: e.ActorID,
			Data: map[string]any{"task_id": e.TaskID, "claim_id": e.ClaimID, "agent_id": e.AgentID, "member_id": e.ActorID, "expires_at": e.ExpiresAt}}, true
	case domain.TaskReleased:
		return BoardEvent{Type: "task.released", BoardID: e.BoardID, ActorID: e.ActorID,
			Data: map[string]any{"task_id": e.TaskID, "claim_id": e.ClaimID}}, true
	case domain.RunStarted:
		return BoardEvent{Type: "run.started", BoardID: e.BoardID, ActorID: e.ActorID,
			Data: map[string]any{"run_id": e.RunID, "task_id": e.TaskID, "agent_id": e.AgentID, "agent_name": e.AgentName}}, true
	case domain.RunFinished:
		return BoardEvent{Type: "run.finished", BoardID: e.BoardID, ActorID: e.ActorID,
			Data: map[string]any{"run_id": e.RunID, "task_id": e.TaskID, "agent_name": e.AgentName, "status": string(e.Status), "cost_usd": e.CostUSD}}, true
	case domain.ColumnCreated:
		return BoardEvent{Type: "column.created", BoardID: e.BoardID, ActorID: e.ActorID,
			Data: map[string]any{"column_id": e.ColumnID, "name": e.Name, "position": e.Position}}, true
	case domain.ColumnRenamed:
		return BoardEvent{Type: "column.updated", BoardID: e.BoardID, ActorID: e.ActorID,
			Data: map[string]any{"column_id": e.ColumnID, "name": e.Name}}, true
	case domain.ColumnMoved:
		return BoardEvent{Type: "column.moved", BoardID: e.BoardID, ActorID: e.ActorID,
			Data: map[string]any{"column_id": e.ColumnID, "position": e.Position}}, true
	case app.PresenceChanged:
		data := map[string]any{"state": e.State, "member_id": e.ActorID, "conn_id": e.ConnID}
		if e.DisplayName != "" {
			data["display_name"] = e.DisplayName
		}
		return BoardEvent{Type: "presence", BoardID: e.BoardID, ActorID: e.ActorID, Data: data}, true
	case domain.ColumnDeleted:
		return BoardEvent{Type: "column.deleted", BoardID: e.BoardID, ActorID: e.ActorID,
			Data: map[string]any{"column_id": e.ColumnID}}, true
	}
	return BoardEvent{}, false
}

// BoardEventPublisher sends boards' domain events, as board events, to
// every API process through Postgres NOTIFY.
type BoardEventPublisher struct{}

func (BoardEventPublisher) Handle(ctx context.Context, event any) {
	ev, ok := boardEventOf(event)
	if !ok {
		return
	}
	ev.At = time.Now().UTC()
	b, err := json.Marshal(ev)
	if err == nil {
		// Payloads must stay under Postgres' 8000-byte NOTIFY limit; events
		// carry ids and small fields only.
		_, err = query(ctx).Exec("SELECT pg_notify(?, ?)", notifyChannel, string(b))
	}
	if err != nil {
		// The change itself is stored; watchers catch up when they refetch.
		facades.Log().WithContext(ctx).Errorf("board event for %T: %v", event, err)
	}
}
