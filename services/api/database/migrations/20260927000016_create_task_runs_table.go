package migrations

// M20260927000016CreateTaskRunsTable records every run of an agent on a
// task. agent_id is the agents context's id, without a foreign key; its name
// is kept as it was when the run started.
type M20260927000016CreateTaskRunsTable struct{}

func (r *M20260927000016CreateTaskRunsTable) Signature() string {
	return "20260927000016_create_task_runs_table"
}

func (r *M20260927000016CreateTaskRunsTable) Up() error {
	return exec(
		`CREATE TABLE task_runs (
			id bigserial PRIMARY KEY,
			task_id bigint NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
			member_id bigint NOT NULL,
			agent_id bigint NOT NULL,
			agent_name varchar(200) NOT NULL,
			machine varchar(200) NOT NULL DEFAULT '',
			branch varchar(200) NOT NULL DEFAULT '',
			status varchar(20) NOT NULL CHECK (status IN ('running', 'succeeded', 'failed', 'stopped')),
			started_at timestamptz NOT NULL,
			ended_at timestamptz,
			cost_usd numeric(10,4) NOT NULL DEFAULT 0 CHECK (cost_usd >= 0),
			turns integer NOT NULL DEFAULT 0 CHECK (turns >= 0),
			summary text NOT NULL DEFAULT '',
			files_changed integer NOT NULL DEFAULT 0 CHECK (files_changed >= 0),
			additions integer NOT NULL DEFAULT 0 CHECK (additions >= 0),
			deletions integer NOT NULL DEFAULT 0 CHECK (deletions >= 0)
		)`,
		`CREATE INDEX task_runs_task_started_index ON task_runs (task_id, started_at DESC)`,
	)
}

func (r *M20260927000016CreateTaskRunsTable) Down() error {
	return exec(`DROP TABLE task_runs`)
}
