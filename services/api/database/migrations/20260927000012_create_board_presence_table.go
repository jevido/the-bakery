package migrations

// M20260927000012CreateBoardPresenceTable keeps who has a board open: one row
// per open event stream, refreshed on every ping and pruned a minute after it
// goes quiet (an API process that died leaves rows behind). member_id is
// identity's id, without a foreign key, as elsewhere in boards.
type M20260927000012CreateBoardPresenceTable struct{}

func (r *M20260927000012CreateBoardPresenceTable) Signature() string {
	return "20260927000012_create_board_presence_table"
}

func (r *M20260927000012CreateBoardPresenceTable) Up() error {
	return exec(
		`CREATE TABLE board_presence (
			conn_id varchar(32) PRIMARY KEY,
			board_id bigint NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
			member_id bigint NOT NULL,
			seen_at timestamptz NOT NULL
		)`,
		`CREATE INDEX board_presence_board_id_index ON board_presence (board_id)`,
		`CREATE INDEX board_presence_seen_at_index ON board_presence (seen_at)`,
	)
}

func (r *M20260927000012CreateBoardPresenceTable) Down() error {
	return exec(`DROP TABLE board_presence`)
}
