package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

// M20260927000003CreateBoardsTables stores the boards context's boards and
// tasks. guild_id is guilds' id; there is deliberately no foreign key
// across contexts.
type M20260927000003CreateBoardsTables struct{}

func (r *M20260927000003CreateBoardsTables) Signature() string {
	return "20260927000003_create_boards_tables"
}

func (r *M20260927000003CreateBoardsTables) Up() error {
	if err := facades.Schema().Create("boards", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("guild_id")
		table.String("name", 60)
		table.TimestampsTz()
		table.Index("guild_id")
	}); err != nil {
		return err
	}
	if err := facades.Schema().Create("tasks", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("board_id")
		table.String("title", 200)
		table.Text("description").Default("")
		table.String("column_key", 10)
		table.String("position")
		table.TimestampsTz()
		table.Foreign("board_id").References("id").On("boards").CascadeOnDelete()
		table.Unique("board_id", "column_key", "position")
	}); err != nil {
		return err
	}
	// Positions are fractional keys compared byte by byte; a locale-aware
	// collation would order them wrongly.
	// Run through the schema's own ORM so it shares the migration's
	// connection (and transaction).
	_, err := facades.Schema().Orm().Query().Exec(`ALTER TABLE tasks ALTER COLUMN position TYPE varchar(255) COLLATE "C"`)
	return err
}

func (r *M20260927000003CreateBoardsTables) Down() error {
	if err := facades.Schema().DropIfExists("tasks"); err != nil {
		return err
	}
	return facades.Schema().DropIfExists("boards")
}
