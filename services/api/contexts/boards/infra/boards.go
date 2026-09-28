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

// columnRecord stores a board column; position is COLLATE "C" like task
// positions.
type columnRecord struct {
	ID       uint64 `gorm:"primaryKey"`
	BoardID  uint64
	Name     string
	Position string
	orm.Timestamps
}

func (columnRecord) TableName() string { return "board_columns" }

func (r columnRecord) toDomain() domain.Column {
	return domain.Column{ID: r.ID, BoardID: r.BoardID, Name: r.Name, Position: r.Position}
}

// Add stores a new board with its columns in one transaction.
func (Boards) Add(ctx context.Context, b domain.Board) (domain.Board, error) {
	rec := boardRecord{GuildID: b.GuildID, Name: b.Name}
	cols := make([]columnRecord, len(b.Columns))
	err := facades.Orm().WithContext(ctx).Transaction(func(tx contractsorm.Query) error {
		if err := tx.Create(&rec); err != nil {
			return err
		}
		for i, c := range b.Columns {
			cols[i] = columnRecord{BoardID: rec.ID, Name: c.Name, Position: c.Position}
			if err := tx.Create(&cols[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.Board{}, err
	}
	out := rec.toDomain()
	for _, c := range cols {
		out.Columns = append(out.Columns, c.toDomain())
	}
	return out, nil
}

func (Boards) ByID(ctx context.Context, id uint64) (domain.Board, bool, error) {
	var rec boardRecord
	if err := query(ctx).Where("id", id).FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return domain.Board{}, false, nil
		}
		return domain.Board{}, false, err
	}
	var cols []columnRecord
	if err := query(ctx).Where("board_id", id).OrderBy("position").Find(&cols); err != nil {
		return domain.Board{}, false, err
	}
	b := rec.toDomain()
	for _, c := range cols {
		b.Columns = append(b.Columns, c.toDomain())
	}
	return b, true, nil
}

func (Boards) ColumnByID(ctx context.Context, id uint64) (domain.Column, bool, error) {
	var rec columnRecord
	if err := query(ctx).Where("id", id).FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return domain.Column{}, false, nil
		}
		return domain.Column{}, false, err
	}
	return rec.toDomain(), true, nil
}

func (Boards) AddColumn(ctx context.Context, c domain.Column) (domain.Column, error) {
	rec := columnRecord{BoardID: c.BoardID, Name: c.Name, Position: c.Position}
	if err := query(ctx).Create(&rec); err != nil {
		return domain.Column{}, columnWriteError(err)
	}
	return rec.toDomain(), nil
}

func (Boards) SaveColumn(ctx context.Context, c domain.Column) error {
	_, err := query(ctx).Model(&columnRecord{}).Where("id", c.ID).Update(map[string]any{
		"name":     c.Name,
		"position": c.Position,
	})
	if err != nil {
		return columnWriteError(err)
	}
	return nil
}

// DeleteColumn removes a column. The tasks' foreign key refuses it while a
// task still stands there, in case one moved in after the use case looked.
func (Boards) DeleteColumn(ctx context.Context, id uint64) error {
	_, err := query(ctx).Where("id", id).Delete(&columnRecord{})
	if err != nil && strings.Contains(err.Error(), "23503") {
		return domain.ErrColumnNotEmpty
	}
	return err
}

// columnWriteError tells a taken position (retry) from a name another
// column took at the same moment.
func columnWriteError(err error) error {
	switch {
	case strings.Contains(err.Error(), "board_columns_name_unique"):
		return domain.ErrDuplicateColumn
	case isUniqueViolation(err):
		return app.ErrPositionTaken
	}
	return err
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

// taskRecord stores a task. ColumnID is NULL for a subtask; position is
// COLLATE "C" so it sorts by byte.
type taskRecord struct {
	ID          uint64 `gorm:"primaryKey"`
	BoardID     uint64
	ParentID    *uint64
	ColumnID    *uint64
	Done        bool
	Title       string
	Description string
	Position    string
	// WorkType is NULL for none.
	WorkType           *string
	PrioritizedAgentID *uint64
	Forbidden          bool
	orm.Timestamps
}

func (taskRecord) TableName() string { return "tasks" }

func (r taskRecord) toDomain() domain.Task {
	return domain.Task{
		ID: r.ID, BoardID: r.BoardID, ParentID: r.ParentID, ColumnID: deref(r.ColumnID), Title: r.Title,
		Description: r.Description, Position: r.Position, Done: r.Done, WorkType: derefString(r.WorkType),
		PrioritizedAgentID: r.PrioritizedAgentID, Forbidden: r.Forbidden,
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func deref(id *uint64) uint64 {
	if id == nil {
		return 0
	}
	return *id
}

func taskToRecord(t domain.Task) taskRecord {
	rec := taskRecord{
		BoardID: t.BoardID, ParentID: t.ParentID, Title: t.Title, Description: t.Description,
		Position: t.Position, Done: t.Done, PrioritizedAgentID: t.PrioritizedAgentID, Forbidden: t.Forbidden,
	}
	if t.ColumnID != 0 {
		rec.ColumnID = &t.ColumnID
	}
	if t.WorkType != "" {
		rec.WorkType = &t.WorkType
	}
	return rec
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
		"title":                t.Title,
		"description":          t.Description,
		"column_id":            taskToRecord(t).ColumnID,
		"work_type":            taskToRecord(t).WorkType,
		"position":             t.Position,
		"done":                 t.Done,
		"prioritized_agent_id": t.PrioritizedAgentID,
		"forbidden":            t.Forbidden,
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
	return findTasks(query(ctx).Where("board_id", boardID).WhereNull("parent_id").OrderBy("column_id").OrderBy("position"))
}

func (Tasks) InColumn(ctx context.Context, columnID uint64) ([]domain.Task, error) {
	return findTasks(query(ctx).Where("column_id", columnID).WhereNull("parent_id").OrderBy("position"))
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

// LogEvents writes boards' domain events to the log.
type LogEvents struct{}

func (LogEvents) Handle(ctx context.Context, event any) {
	facades.Log().WithContext(ctx).Infof("%T %+v", event, event)
}
