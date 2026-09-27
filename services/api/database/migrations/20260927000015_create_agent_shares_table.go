package migrations

// M20260927000015CreateAgentSharesTable records which agent is shared with
// which guild. guild_id is guilds' id, without a foreign key.
type M20260927000015CreateAgentSharesTable struct{}

func (r *M20260927000015CreateAgentSharesTable) Signature() string {
	return "20260927000015_create_agent_shares_table"
}

func (r *M20260927000015CreateAgentSharesTable) Up() error {
	return exec(
		`CREATE TABLE agent_shares (
			agent_id bigint NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
			guild_id bigint NOT NULL,
			shared_at timestamptz NOT NULL DEFAULT now(),
			PRIMARY KEY (agent_id, guild_id)
		)`,
		`CREATE INDEX agent_shares_guild_index ON agent_shares (guild_id)`,
	)
}

func (r *M20260927000015CreateAgentSharesTable) Down() error {
	return exec(`DROP TABLE agent_shares`)
}
