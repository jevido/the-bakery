package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

// M20260927000009CreateTaskCommentsTable stores comments on tasks.
// author_id is identity's member id; there is deliberately no foreign key
// to members, because boards does not reach into identity's tables.
type M20260927000009CreateTaskCommentsTable struct{}

func (r *M20260927000009CreateTaskCommentsTable) Signature() string {
	return "20260927000009_create_task_comments_table"
}

func (r *M20260927000009CreateTaskCommentsTable) Up() error {
	return facades.Schema().Create("task_comments", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("task_id")
		table.UnsignedBigInteger("author_id")
		table.Text("body")
		table.TimestampTz("created_at")
		table.TimestampTz("edited_at").Nullable()
		table.Foreign("task_id").References("id").On("tasks").CascadeOnDelete()
		table.Index("task_id", "created_at")
	})
}

func (r *M20260927000009CreateTaskCommentsTable) Down() error {
	return facades.Schema().DropIfExists("task_comments")
}
