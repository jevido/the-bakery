// Package infra stores boards and tasks with the Goravel ORM.
package infra

import (
	"context"
	"errors"
	"strings"

	contractsorm "github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/database/orm"
	frameworkerrors "github.com/goravel/framework/errors"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

func query(ctx context.Context) contractsorm.Query {
	return facades.Orm().WithContext(ctx).Query()
}

func notFound(err error) bool {
	return errors.Is(err, frameworkerrors.OrmRecordNotFound)
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key")
}

type boardRecord struct {
	ID      uint64 `gorm:"primaryKey"`
	GuildID uint64
	Name    string
	orm.Timestamps
}

func (boardRecord) TableName() string { return "boards" }

func (r boardRecord) toDomain() domain.Board {
	return domain.Board{ID: r.ID, GuildID: r.GuildID, Name: r.Name}
}

type Boards struct{}

func (Boards) Add(ctx context.Context, b domain.Board) (domain.Board, error) {
	rec := boardRecord{GuildID: b.GuildID, Name: b.Name}
	if err := query(ctx).Create(&rec); err != nil {
		return domain.Board{}, err
	}
	return rec.toDomain(), nil
}

func (Boards) ByID(ctx context.Context, id uint64) (domain.Board, bool, error) {
	var rec boardRecord
	if err := query(ctx).Where("id", id).FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return domain.Board{}, false, nil
		}
		return domain.Board{}, false, err
	}
	return rec.toDomain(), true, nil
}

func (Boards) OfGuild(ctx context.Context, guildID uint64) ([]domain.Board, error) {
	var recs []boardRecord
	if err := query(ctx).Where("guild_id", guildID).OrderBy("name").OrderBy("id").Find(&recs); err != nil {
		return nil, err
	}
	boards := make([]domain.Board, len(recs))
	for i, r := range recs {
		boards[i] = r.toDomain()
	}
	return boards, nil
}

// ByName finds a guild's board by name, for the dev seeder only.
func (Boards) ByName(ctx context.Context, guildID uint64, name string) (domain.Board, bool, error) {
	var rec boardRecord
	if err := query(ctx).Where("guild_id", guildID).Where("name", name).OrderBy("id").FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return domain.Board{}, false, nil
		}
		return domain.Board{}, false, err
	}
	return rec.toDomain(), true, nil
}

// taskRecord stores a task. The column is stored as column_key because
// "column" is an SQL keyword; position is COLLATE "C" so it sorts by byte.
type taskRecord struct {
	ID          uint64 `gorm:"primaryKey"`
	BoardID     uint64
	ParentID    *uint64
	Done        bool
	Title       string
	Description string
	ColumnKey   string
	Position    string
	orm.Timestamps
}

func (taskRecord) TableName() string { return "tasks" }

func (r taskRecord) toDomain() domain.Task {
	return domain.Task{
		ID: r.ID, BoardID: r.BoardID, ParentID: r.ParentID, Title: r.Title, Description: r.Description,
		Column: domain.Column(r.ColumnKey), Position: r.Position, Done: r.Done,
	}
}

func taskToRecord(t domain.Task) taskRecord {
	return taskRecord{
		BoardID: t.BoardID, ParentID: t.ParentID, Title: t.Title, Description: t.Description,
		ColumnKey: string(t.Column), Position: t.Position, Done: t.Done,
	}
}

type Tasks struct{}

func (Tasks) Add(ctx context.Context, t domain.Task) (domain.Task, error) {
	rec := taskToRecord(t)
	if err := query(ctx).Create(&rec); err != nil {
		if isUniqueViolation(err) {
			return domain.Task{}, app.ErrPositionTaken
		}
		return domain.Task{}, err
	}
	return rec.toDomain(), nil
}

func (Tasks) Save(ctx context.Context, t domain.Task) error {
	_, err := query(ctx).Model(&taskRecord{}).Where("id", t.ID).Update(map[string]any{
		"title":       t.Title,
		"description": t.Description,
		"column_key":  string(t.Column),
		"position":    t.Position,
		"done":        t.Done,
	})
	if err != nil && isUniqueViolation(err) {
		return app.ErrPositionTaken
	}
	return err
}

func (Tasks) Delete(ctx context.Context, id uint64) error {
	_, err := query(ctx).Where("id", id).Delete(&taskRecord{})
	return err
}

func (Tasks) ByID(ctx context.Context, id uint64) (domain.Task, bool, error) {
	var rec taskRecord
	if err := query(ctx).Where("id", id).FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return domain.Task{}, false, nil
		}
		return domain.Task{}, false, err
	}
	return rec.toDomain(), true, nil
}

// OfBoard and InColumn return top-level tasks only; subtasks are not in a
// column.
func (Tasks) OfBoard(ctx context.Context, boardID uint64) ([]domain.Task, error) {
	return findTasks(query(ctx).Where("board_id", boardID).WhereNull("parent_id").OrderBy("column_key").OrderBy("position"))
}

func (Tasks) InColumn(ctx context.Context, boardID uint64, column domain.Column) ([]domain.Task, error) {
	return findTasks(query(ctx).Where("board_id", boardID).WhereNull("parent_id").Where("column_key", string(column)).OrderBy("position"))
}

func (Tasks) AddAll(ctx context.Context, ts []domain.Task) ([]domain.Task, error) {
	recs := make([]taskRecord, len(ts))
	for i, t := range ts {
		recs[i] = taskToRecord(t)
	}
	err := facades.Orm().WithContext(ctx).Transaction(func(tx contractsorm.Query) error {
		for i := range recs {
			if err := tx.Create(&recs[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, app.ErrPositionTaken
		}
		return nil, err
	}
	added := make([]domain.Task, len(recs))
	for i, r := range recs {
		added[i] = r.toDomain()
	}
	return added, nil
}

func (Tasks) SubtasksOf(ctx context.Context, parentID uint64) ([]domain.Task, error) {
	return findTasks(query(ctx).Where("parent_id", parentID).OrderBy("position"))
}

func (Tasks) SubtaskCounts(ctx context.Context, parentIDs []uint64) (map[uint64]app.SubtaskCount, error) {
	counts := map[uint64]app.SubtaskCount{}
	if len(parentIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		ParentID uint64
		Total    int
		Done     int
	}
	err := query(ctx).Raw(
		`SELECT parent_id, count(*) AS total, count(*) FILTER (WHERE done) AS done
		 FROM tasks WHERE parent_id IN ? GROUP BY parent_id`, parentIDs,
	).Scan(&rows)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		counts[r.ParentID] = app.SubtaskCount{Total: r.Total, Done: r.Done}
	}
	return counts, nil
}

func findTasks(q contractsorm.Query) ([]domain.Task, error) {
	var recs []taskRecord
	if err := q.Find(&recs); err != nil {
		return nil, err
	}
	tasks := make([]domain.Task, len(recs))
	for i, r := range recs {
		tasks[i] = r.toDomain()
	}
	return tasks, nil
}

// LogEvents writes boards' domain events to the log; nothing subscribes to
// them yet.
type LogEvents struct{}

func (LogEvents) TaskCreated(ctx context.Context, ev domain.TaskCreated) {
	facades.Log().WithContext(ctx).Infof("TaskCreated task=%d board=%d column=%s position=%s", ev.TaskID, ev.BoardID, ev.Column, ev.Position)
}

func (LogEvents) TaskMoved(ctx context.Context, ev domain.TaskMoved) {
	facades.Log().WithContext(ctx).Infof("TaskMoved task=%d board=%d from=%s to=%s position=%s", ev.TaskID, ev.BoardID, ev.From, ev.To, ev.Position)
}

func (LogEvents) SubtaskAdded(ctx context.Context, ev domain.SubtaskAdded) {
	facades.Log().WithContext(ctx).Infof("SubtaskAdded subtask=%d parent=%d board=%d", ev.SubtaskID, ev.ParentID, ev.BoardID)
}

func (LogEvents) TaskCommented(ctx context.Context, ev domain.TaskCommented) {
	facades.Log().WithContext(ctx).Infof("TaskCommented task=%d board=%d comment=%d author=%d", ev.TaskID, ev.BoardID, ev.CommentID, ev.AuthorID)
}

func (LogEvents) SubtaskCompleted(ctx context.Context, ev domain.SubtaskCompleted) {
	facades.Log().WithContext(ctx).Infof("SubtaskCompleted subtask=%d parent=%d board=%d", ev.SubtaskID, ev.ParentID, ev.BoardID)
}
