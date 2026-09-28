package infra

import (
	"context"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

type reportRecord struct {
	ID         uint64 `gorm:"primaryKey"`
	ByMemberID uint64
	TargetKind string
	TargetID   uint64
	Reason     string
	Status     string
	HandledBy  *uint64
	HandledAt  *time.Time
	SanctionID *uint64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (reportRecord) TableName() string { return "reports" }

func (r reportRecord) toDomain() domain.Report {
	out := domain.Report{ID: r.ID, ByMemberID: r.ByMemberID, TargetKind: r.TargetKind, TargetID: r.TargetID, Reason: r.Reason,
		Status: r.Status, HandledAt: r.HandledAt, CreatedAt: r.CreatedAt}
	if r.HandledBy != nil {
		out.HandledBy = *r.HandledBy
	}
	if r.SanctionID != nil {
		out.SanctionID = *r.SanctionID
	}
	return out
}

func nonZero(n uint64) *uint64 {
	if n == 0 {
		return nil
	}
	return &n
}

type Reports struct{}

func (Reports) Add(ctx context.Context, r domain.Report) (domain.Report, error) {
	rec := reportRecord{ByMemberID: r.ByMemberID, TargetKind: r.TargetKind, TargetID: r.TargetID, Reason: r.Reason, Status: r.Status}
	if err := query(ctx).Create(&rec); err != nil {
		return domain.Report{}, err
	}
	return rec.toDomain(), nil
}

func (Reports) Save(ctx context.Context, r domain.Report) error {
	_, err := query(ctx).Model(&reportRecord{}).Where("id", r.ID).Update(map[string]any{
		"status": r.Status, "handled_by": nonZero(r.HandledBy), "handled_at": r.HandledAt, "sanction_id": nonZero(r.SanctionID),
	})
	return err
}

func (Reports) ByID(ctx context.Context, id uint64) (domain.Report, bool, error) {
	var rec reportRecord
	if err := query(ctx).Where("id", id).FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return domain.Report{}, false, nil
		}
		return domain.Report{}, false, err
	}
	return rec.toDomain(), true, nil
}

func (Reports) FiledSince(ctx context.Context, memberID uint64, since time.Time) (int, error) {
	n, err := query(ctx).Model(&reportRecord{}).Where("by_member_id", memberID).Where("created_at > ?", since).Count()
	return int(n), err
}

func (Reports) WithStatus(ctx context.Context, status string) ([]domain.Report, error) {
	q := query(ctx)
	if status != "" {
		q = q.Where("status", status)
	}
	var recs []reportRecord
	if err := q.OrderBy("id").Limit(500).Find(&recs); err != nil {
		return nil, err
	}
	out := make([]domain.Report, len(recs))
	for i, r := range recs {
		out[i] = r.toDomain()
	}
	return out, nil
}

// About lists the reports about one target, newest first.
func (Reports) About(ctx context.Context, targetKind string, targetID uint64) ([]domain.Report, error) {
	var recs []reportRecord
	if err := query(ctx).Where("target_kind", targetKind).Where("target_id", targetID).OrderByDesc("id").Limit(200).Find(&recs); err != nil {
		return nil, err
	}
	out := make([]domain.Report, len(recs))
	for i, r := range recs {
		out[i] = r.toDomain()
	}
	return out, nil
}
