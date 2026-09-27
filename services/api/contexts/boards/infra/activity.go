package infra

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

type activityRecord struct {
	ID      uint64 `gorm:"primaryKey"`
	TaskID  uint64
	BoardID uint64
	Kind    string
	ActorID uint64
	Data    string `gorm:"type:jsonb"`
	At      time.Time
}

func (activityRecord) TableName() string { return "task_activity" }

// ActivityProjector turns boards' domain events into task activity. A
// subtask's events go on its parent's history, since the parent is what
// people open.
type ActivityProjector struct{}

func (ActivityProjector) Handle(ctx context.Context, event any) {
	var rec activityRecord
	var data map[string]any
	switch ev := event.(type) {
	case domain.TaskCreated:
		rec = activityRecord{TaskID: ev.TaskID, BoardID: ev.BoardID, ActorID: ev.ActorID, Kind: string(app.ActivityCreated)}
		data = map[string]any{"column": columnName(ctx, ev.ColumnID)}
	case domain.TaskEdited:
		rec = activityRecord{TaskID: ev.TaskID, BoardID: ev.BoardID, ActorID: ev.ActorID, Kind: string(app.ActivityEdited)}
		data = map[string]any{"title": ev.Title, "description": ev.Description, "work_type": ev.WorkType}
	case domain.TaskMoved:
		rec = activityRecord{TaskID: ev.TaskID, BoardID: ev.BoardID, ActorID: ev.ActorID, Kind: string(app.ActivityMoved)}
		data = map[string]any{"from": columnName(ctx, ev.From), "to": columnName(ctx, ev.To)}
	case domain.TaskCommented:
		rec = activityRecord{TaskID: ev.TaskID, BoardID: ev.BoardID, ActorID: ev.ActorID, Kind: string(app.ActivityCommented)}
		data = map[string]any{"comment_id": ev.CommentID}
	case domain.SubtaskAdded:
		rec = activityRecord{TaskID: ev.ParentID, BoardID: ev.BoardID, ActorID: ev.ActorID, Kind: string(app.ActivitySubtaskAdded)}
		data = map[string]any{"subtask_id": ev.SubtaskID, "title": ev.Title}
	case domain.SubtaskCompleted:
		rec = activityRecord{TaskID: ev.ParentID, BoardID: ev.BoardID, ActorID: ev.ActorID, Kind: string(app.ActivitySubtaskDone)}
		data = map[string]any{"subtask_id": ev.SubtaskID, "title": ev.Title}
	case domain.RunStarted:
		rec = activityRecord{TaskID: ev.TaskID, BoardID: ev.BoardID, ActorID: ev.ActorID, Kind: string(app.ActivityRunStarted)}
		data = map[string]any{"run_id": ev.RunID, "agent_name": ev.AgentName}
	case domain.RunFinished:
		rec = activityRecord{TaskID: ev.TaskID, BoardID: ev.BoardID, ActorID: ev.ActorID, Kind: string(app.ActivityRunFinished)}
		data = map[string]any{"run_id": ev.RunID, "agent_name": ev.AgentName, "status": string(ev.Status), "cost_usd": ev.CostUSD}
	default:
		return
	}
	b, err := json.Marshal(data)
	if err == nil {
		rec.Data, rec.At = string(b), time.Now()
		err = query(ctx).Create(&rec)
	}
	if err != nil {
		// The change itself is stored; only its history entry is lost.
		facades.Log().WithContext(ctx).Errorf("activity for %T: %v", event, err)
	}
}

// columnName is a column's name now, which activity keeps so the history
// still reads after the column is renamed or deleted.
func columnName(ctx context.Context, id uint64) string {
	var names []string
	if err := query(ctx).Table("board_columns").Where("id", id).Pluck("name", &names); err != nil || len(names) == 0 {
		return ""
	}
	return names[0]
}

type ActivityLog struct{}

func (ActivityLog) OfTask(ctx context.Context, taskID, before uint64, limit int) ([]app.Activity, error) {
	q := query(ctx).Where("task_id", taskID)
	if before > 0 {
		q = q.Where("id < ?", before)
	}
	var recs []activityRecord
	if err := q.OrderByDesc("id").Limit(limit).Find(&recs); err != nil {
		return nil, err
	}
	out := make([]app.Activity, len(recs))
	for i, r := range recs {
		var data map[string]any
		if err := json.Unmarshal([]byte(r.Data), &data); err != nil {
			return nil, err
		}
		out[i] = app.Activity{ID: r.ID, TaskID: r.TaskID, Kind: app.ActivityKind(r.Kind), ActorID: r.ActorID, At: r.At, Data: data}
	}
	return out, nil
}
