package migrations

// M20260927000023CreateSignalsTable holds abuse signals: sign-ups, guilds
// founded and rate-limit refusals, with the client's IP, for the console's
// Signals screen.
type M20260927000023CreateSignalsTable struct{}

func (r *M20260927000023CreateSignalsTable) Signature() string {
	return "20260927000023_create_signals_table"
}

func (r *M20260927000023CreateSignalsTable) Up() error {
	return exec(
		`CREATE TABLE signals (
			id bigserial PRIMARY KEY,
			kind varchar(20) NOT NULL CHECK (kind IN ('sign_up', 'guild_founded', 'limit_hit')),
			limit_name varchar(30) NOT NULL DEFAULT '',
			member_id bigint,
			ip varchar(64) NOT NULL DEFAULT '',
			at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX signals_kind_at_index ON signals (kind, at)`,
	)
}

func (r *M20260927000023CreateSignalsTable) Down() error {
	return exec(`DROP TABLE signals`)
}
