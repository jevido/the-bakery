package migrations

// M20260927000014CreateAgentsTables stores members' agents and their
// skillsets. owner_member_id is identity's id, without a foreign key: the
// agents context never reaches into identity's tables. A deleted agent keeps
// its row (deleted_at) so devices syncing later learn about it; its slug is
// free again for a new agent.
type M20260927000014CreateAgentsTables struct{}

func (r *M20260927000014CreateAgentsTables) Signature() string {
	return "20260927000014_create_agents_tables"
}

func (r *M20260927000014CreateAgentsTables) Up() error {
	return exec(
		`CREATE TABLE agents (
			id bigserial PRIMARY KEY,
			owner_member_id bigint NOT NULL,
			slug varchar(40) NOT NULL,
			name varchar(40) NOT NULL,
			title varchar(60) NOT NULL DEFAULT '',
			backstory text NOT NULL DEFAULT '',
			traits jsonb NOT NULL DEFAULT '[]',
			model varchar(100) NOT NULL,
			permission_mode varchar(20) NOT NULL,
			allowed_tools jsonb NOT NULL DEFAULT '[]',
			portrait_seed varchar(40) NOT NULL DEFAULT '',
			work_priorities jsonb NOT NULL DEFAULT '{}',
			revision integer NOT NULL,
			origin_agent_id bigint,
			origin_revision integer,
			deleted_at timestamptz,
			created_at timestamptz,
			updated_at timestamptz
		)`,
		`CREATE UNIQUE INDEX agents_owner_slug_unique ON agents (owner_member_id, slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX agents_owner_updated_index ON agents (owner_member_id, updated_at)`,
		`CREATE TABLE agent_files (
			agent_id bigint NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
			path varchar(500) NOT NULL,
			content text NOT NULL,
			CONSTRAINT agent_files_path_unique UNIQUE (agent_id, path)
		)`,
	)
}

func (r *M20260927000014CreateAgentsTables) Down() error {
	return exec(`DROP TABLE agent_files`, `DROP TABLE agents`)
}
