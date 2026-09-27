package infra

import (
	"context"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
)

type presenceRecord struct {
	ConnID   string `gorm:"primaryKey"`
	BoardID  uint64
	MemberID uint64
	SeenAt   time.Time
}

func (presenceRecord) TableName() string { return "board_presence" }

func (r presenceRecord) toApp() app.Presence {
	return app.Presence{ConnID: r.ConnID, BoardID: r.BoardID, MemberID: r.MemberID, SeenAt: r.SeenAt}
}

type PresenceLog struct{}

func (PresenceLog) Enter(ctx context.Context, p app.Presence) error {
	return query(ctx).Create(&presenceRecord{ConnID: p.ConnID, BoardID: p.BoardID, MemberID: p.MemberID, SeenAt: p.SeenAt})
}

func (PresenceLog) Seen(ctx context.Context, connID string, at time.Time) error {
	_, err := query(ctx).Model(&presenceRecord{}).Where("conn_id", connID).Update("seen_at", at)
	return err
}

func (PresenceLog) Leave(ctx context.Context, connID string) (bool, error) {
	res, err := query(ctx).Where("conn_id", connID).Delete(&presenceRecord{})
	if err != nil {
		return false, err
	}
	return res.RowsAffected > 0, nil
}

func (PresenceLog) OnBoard(ctx context.Context, boardID uint64) ([]app.Presence, error) {
	var recs []presenceRecord
	if err := query(ctx).Where("board_id", boardID).OrderBy("seen_at").Find(&recs); err != nil {
		return nil, err
	}
	out := make([]app.Presence, len(recs))
	for i, r := range recs {
		out[i] = r.toApp()
	}
	return out, nil
}

// Prune deletes quiet streams in one statement and returns what it deleted,
// so two processes pruning at once never both announce the same departure.
func (PresenceLog) Prune(ctx context.Context, before time.Time) ([]app.Presence, error) {
	var recs []presenceRecord
	if err := query(ctx).Raw(
		`DELETE FROM board_presence WHERE seen_at < ? RETURNING conn_id, board_id, member_id, seen_at`, before,
	).Scan(&recs); err != nil {
		return nil, err
	}
	out := make([]app.Presence, len(recs))
	for i, r := range recs {
		out[i] = r.toApp()
	}
	return out, nil
}
