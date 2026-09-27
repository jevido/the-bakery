package migrations

import (
	"github.com/jevido/the-bakery/services/api/app/facades"
)

// M20260927000011CreateBoardColumnsTable gives every board its own columns.
// Boards had four fixed columns, stored on each task as column_key; now a
// board owns rows in board_columns and a task on the board refers to one by
// column_id. A subtask has no column (column_id NULL). Every existing board
// gets the four columns it had, in the same order, and its tasks keep their
// place. Old activity named columns by key; it is rewritten to the names.
type M20260927000011CreateBoardColumnsTable struct{}

func (r *M20260927000011CreateBoardColumnsTable) Signature() string {
	return "20260927000011_create_board_columns_table"
}

// defaultColumns are the fixed keys, their names and the positions the
// first four keys from domain.KeyBetween get.
const defaultColumns = `(VALUES ('backlog', 'Backlog', 'a0'), ('todo', 'To do', 'a1'), ('doing', 'Doing', 'a2'), ('done', 'Done', 'a3'))`

func (r *M20260927000011CreateBoardColumnsTable) Up() error {
	return exec(
		`CREATE TABLE board_columns (
			id bigserial PRIMARY KEY,
			board_id bigint NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
			name varchar(40) NOT NULL,
			position varchar(255) COLLATE "C" NOT NULL,
			created_at timestamptz,
			updated_at timestamptz,
			CONSTRAINT board_columns_position_unique UNIQUE (board_id, position)
		)`,
		`CREATE UNIQUE INDEX board_columns_name_unique ON board_columns (board_id, lower(name))`,
		`INSERT INTO board_columns (board_id, name, position, created_at, updated_at)
			SELECT b.id, d.name, d.position, now(), now() FROM boards b CROSS JOIN `+defaultColumns+` AS d (key, name, position)`,
		// A column that still holds tasks cannot be deleted. NO ACTION (not
		// RESTRICT) is checked at the end of the statement, so deleting a
		// board, which cascades to its tasks and columns, still works.
		`ALTER TABLE tasks ADD COLUMN column_id bigint REFERENCES board_columns (id)`,
		`UPDATE tasks t SET column_id = c.id
			FROM board_columns c, `+defaultColumns+` AS d (key, name, position)
			WHERE t.parent_id IS NULL AND c.board_id = t.board_id AND d.key = t.column_key AND c.name = d.name`,
		`DROP INDEX tasks_column_position_unique`,
		`ALTER TABLE tasks DROP COLUMN column_key`,
		`CREATE UNIQUE INDEX tasks_column_position_unique ON tasks (column_id, position) WHERE parent_id IS NULL`,
		`ALTER TABLE tasks ADD CONSTRAINT tasks_column_only_on_board CHECK ((parent_id IS NULL) = (column_id IS NOT NULL))`,
		`UPDATE task_activity a SET data = jsonb_set(a.data, '{column}', to_jsonb(d.name))
			FROM `+defaultColumns+` AS d (key, name, position)
			WHERE a.kind = 'created' AND a.data->>'column' = d.key`,
		`UPDATE task_activity a SET data = jsonb_set(a.data, '{from}', to_jsonb(d.name))
			FROM `+defaultColumns+` AS d (key, name, position)
			WHERE a.kind = 'moved' AND a.data->>'from' = d.key`,
		`UPDATE task_activity a SET data = jsonb_set(a.data, '{to}', to_jsonb(d.name))
			FROM `+defaultColumns+` AS d (key, name, position)
			WHERE a.kind = 'moved' AND a.data->>'to' = d.key`,
	)
}

// Down puts the four fixed columns back. It refuses while a task stands in
// a column that is not one of the four defaults (added or renamed by a
// member): those tasks would have nowhere to go. Subtasks get their
// parent's key again.
func (r *M20260927000011CreateBoardColumnsTable) Down() error {
	return exec(
		`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM tasks t JOIN board_columns c ON c.id = t.column_id
				WHERE c.name NOT IN ('Backlog', 'To do', 'Doing', 'Done')) THEN
				RAISE EXCEPTION 'tasks stand in columns other than Backlog, To do, Doing and Done; move them first';
			END IF;
		END $$`,
		`ALTER TABLE tasks ADD COLUMN column_key varchar(10)`,
		`UPDATE tasks t SET column_key = d.key
			FROM board_columns c JOIN `+defaultColumns+` AS d (key, name, position) ON d.name = c.name
			WHERE t.column_id = c.id`,
		`UPDATE tasks t SET column_key = p.column_key FROM tasks p WHERE t.parent_id = p.id`,
		`ALTER TABLE tasks ALTER COLUMN column_key SET NOT NULL`,
		`ALTER TABLE tasks DROP CONSTRAINT tasks_column_only_on_board`,
		`DROP INDEX tasks_column_position_unique`,
		`ALTER TABLE tasks DROP COLUMN column_id`,
		`CREATE UNIQUE INDEX tasks_column_position_unique ON tasks (board_id, column_key, position) WHERE parent_id IS NULL`,
		`DROP TABLE board_columns`,
		`UPDATE task_activity a SET data = jsonb_set(a.data, '{column}', to_jsonb(d.key))
			FROM `+defaultColumns+` AS d (key, name, position)
			WHERE a.kind = 'created' AND a.data->>'column' = d.name`,
		`UPDATE task_activity a SET data = jsonb_set(a.data, '{from}', to_jsonb(d.key))
			FROM `+defaultColumns+` AS d (key, name, position)
			WHERE a.kind = 'moved' AND a.data->>'from' = d.name`,
		`UPDATE task_activity a SET data = jsonb_set(a.data, '{to}', to_jsonb(d.key))
			FROM `+defaultColumns+` AS d (key, name, position)
			WHERE a.kind = 'moved' AND a.data->>'to' = d.name`,
	)
}

// exec runs the statements in order on the migration's own connection (and
// transaction).
func exec(statements ...string) error {
	for _, sql := range statements {
		if _, err := facades.Schema().Orm().Query().Exec(sql); err != nil {
			return err
		}
	}
	return nil
}
