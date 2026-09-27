package migrations

// M20260927000013CreateWorkTypesTable gives each guild a list of work types
// and tasks a work type key. guild_id is guilds' id without a foreign key:
// boards never reaches into guilds' tables. A guild's defaults are created
// by the boards context the first time they are asked for, not here.
type M20260927000013CreateWorkTypesTable struct{}

func (r *M20260927000013CreateWorkTypesTable) Signature() string {
	return "20260927000013_create_work_types_table"
}

func (r *M20260927000013CreateWorkTypesTable) Up() error {
	return exec(
		`CREATE TABLE work_types (
			id bigserial PRIMARY KEY,
			guild_id bigint NOT NULL,
			key varchar(32) NOT NULL,
			name varchar(40) NOT NULL,
			position varchar(255) COLLATE "C" NOT NULL,
			created_at timestamptz,
			updated_at timestamptz,
			CONSTRAINT work_types_key_unique UNIQUE (guild_id, key)
		)`,
		`ALTER TABLE tasks ADD COLUMN work_type varchar(32)`,
		`CREATE INDEX tasks_work_type_index ON tasks (work_type)`,
	)
}

func (r *M20260927000013CreateWorkTypesTable) Down() error {
	return exec(`ALTER TABLE tasks DROP COLUMN work_type`, `DROP TABLE work_types`)
}
