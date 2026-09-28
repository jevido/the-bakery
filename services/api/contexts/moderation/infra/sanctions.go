package infra

import (
	"context"
	"time"

	contractsorm "github.com/goravel/framework/contracts/database/orm"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

type sanctionRecord struct {
	ID           uint64 `gorm:"primaryKey"`
	TargetKind   string
	TargetID     uint64
	Kind         string
	Reason       string
	Until        *time.Time
	ByOperatorID uint64
	LiftedAt     *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (sanctionRecord) TableName() string { return "sanctions" }

func (r sanctionRecord) toDomain() domain.Sanction {
	return domain.Sanction{ID: r.ID, TargetKind: r.TargetKind, TargetID: r.TargetID, Kind: r.Kind, Reason: r.Reason,
		Until: r.Until, ByOperatorID: r.ByOperatorID, LiftedAt: r.LiftedAt, CreatedAt: r.CreatedAt}
}

type Sanctions struct{}

// Add ends the target's finished suspension (it is lifted when it ran out)
// before storing the new sanction, so the one-unlifted-per-target index
// lets it in; one still holding makes it fail.
func (Sanctions) Add(ctx context.Context, s domain.Sanction) (domain.Sanction, error) {
	rec := sanctionRecord{TargetKind: s.TargetKind, TargetID: s.TargetID, Kind: s.Kind, Reason: s.Reason, Until: s.Until, ByOperatorID: s.ByOperatorID}
	err := facades.Orm().WithContext(ctx).Transaction(func(tx contractsorm.Query) error {
		if _, err := tx.Exec(`UPDATE sanctions SET lifted_at = until WHERE target_kind = ? AND target_id = ? AND lifted_at IS NULL AND until IS NOT NULL AND until <= ?`,
			s.TargetKind, s.TargetID, s.CreatedAt); err != nil {
			return err
		}
		return tx.Create(&rec)
	})
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Sanction{}, domain.ErrAlreadySanctioned
		}
		return domain.Sanction{}, err
	}
	return rec.toDomain(), nil
}

func (Sanctions) Save(ctx context.Context, s domain.Sanction) error {
	_, err := query(ctx).Model(&sanctionRecord{}).Where("id", s.ID).Update("lifted_at", s.LiftedAt)
	return err
}

func (Sanctions) ByID(ctx context.Context, id uint64) (domain.Sanction, bool, error) {
	var rec sanctionRecord
	if err := query(ctx).Where("id", id).FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return domain.Sanction{}, false, nil
		}
		return domain.Sanction{}, false, err
	}
	return rec.toDomain(), true, nil
}

func (Sanctions) Active(ctx context.Context, targetKind string, targetID uint64, now time.Time) (*domain.Sanction, error) {
	var recs []sanctionRecord
	if err := query(ctx).Where("target_kind", targetKind).Where("target_id", targetID).Where("lifted_at IS NULL").
		Where("(until IS NULL OR until > ?)", now).Limit(1).Find(&recs); err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, nil
	}
	s := recs[0].toDomain()
	return &s, nil
}

func (Sanctions) OfTarget(ctx context.Context, targetKind string, targetID uint64) ([]domain.Sanction, error) {
	var recs []sanctionRecord
	if err := query(ctx).Where("target_kind", targetKind).Where("target_id", targetID).OrderByDesc("id").Find(&recs); err != nil {
		return nil, err
	}
	out := make([]domain.Sanction, len(recs))
	for i, r := range recs {
		out[i] = r.toDomain()
	}
	return out, nil
}
