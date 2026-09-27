package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

// M20260927000010CreateTaskActivityTable stores task activity, the
// projection of boards' domain events. actor_id is identity's member id
// (no foreign key, as for comments).
type M20260927000010CreateTaskActivityTable struct{}

func (r *M20260927000010CreateTaskActivityTable) Signature() string {
	return "20260927000010_create_task_activity_table"
}

func (r *M20260927000010CreateTaskActivityTable) Up() error {
	if err := facades.Schema().Create("task_activity", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("task_id")
		table.UnsignedBigInteger("board_id")
		table.String("kind", 20)
		table.UnsignedBigInteger("actor_id")
		table.Jsonb("data")
		table.TimestampTz("at")
		table.Foreign("task_id").References("id").On("tasks").CascadeOnDelete()
	}); err != nil {
		return err
	}
	_, err := facades.Schema().Orm().Query().Exec(`CREATE INDEX task_activity_task_id_id_index ON task_activity (task_id, id DESC)`)
	return err
}

func (r *M20260927000010CreateTaskActivityTable) Down() error {
	return facades.Schema().DropIfExists("task_activity")
}
