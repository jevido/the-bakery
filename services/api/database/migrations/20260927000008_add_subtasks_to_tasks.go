package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

// M20260927000008AddSubtasksToTasks lets a task have subtasks: rows in the
// same table with a parent_id. A subtask's column_key is a copy of its
// parent's at creation and means nothing; subtasks are not in a column and
// are ordered by position under their parent instead.
type M20260927000008AddSubtasksToTasks struct{}

func (r *M20260927000008AddSubtasksToTasks) Signature() string {
	return "20260927000008_add_subtasks_to_tasks"
}

func (r *M20260927000008AddSubtasksToTasks) Up() error {
	if err := facades.Schema().Table("tasks", func(table schema.Blueprint) {
		table.UnsignedBigInteger("parent_id").Nullable()
		table.Boolean("done").Default(false)
		table.Foreign("parent_id").References("id").On("tasks").CascadeOnDelete()
	}); err != nil {
		return err
	}
	// Positions are unique per column among top-level tasks, and per parent
	// among subtasks. One index over both would make them collide.
	for _, sql := range []string{
		`ALTER TABLE tasks DROP CONSTRAINT tasks_board_id_column_key_position_unique`,
		`CREATE UNIQUE INDEX tasks_column_position_unique ON tasks (board_id, column_key, position) WHERE parent_id IS NULL`,
		`CREATE UNIQUE INDEX tasks_subtask_position_unique ON tasks (parent_id, position) WHERE parent_id IS NOT NULL`,
	} {
		if _, err := facades.Schema().Orm().Query().Exec(sql); err != nil {
			return err
		}
	}
	return nil
}

func (r *M20260927000008AddSubtasksToTasks) Down() error {
	for _, sql := range []string{
		`DELETE FROM tasks WHERE parent_id IS NOT NULL`,
		`DROP INDEX tasks_subtask_position_unique`,
		`DROP INDEX tasks_column_position_unique`,
		`ALTER TABLE tasks ADD CONSTRAINT tasks_board_id_column_key_position_unique UNIQUE (board_id, column_key, position)`,
	} {
		if _, err := facades.Schema().Orm().Query().Exec(sql); err != nil {
			return err
		}
	}
	return facades.Schema().Table("tasks", func(table schema.Blueprint) {
		table.DropForeign("parent_id")
		table.DropColumn("parent_id", "done")
	})
}
