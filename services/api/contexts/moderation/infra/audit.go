package infra

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

type auditRecord struct {
	ID         uint64 `gorm:"primaryKey"`
	ActorKind  string
	ActorID    uint64
	Action     string
	TargetKind string
	TargetID   uint64
	Reason     string
	Meta       string
	IP         string `gorm:"column:ip"`
	At         time.Time
}

func (auditRecord) TableName() string { return "audit_entries" }

type AuditEntries struct{}

func (AuditEntries) Add(ctx context.Context, e domain.AuditEntry) error {
	meta, err := json.Marshal(e.Meta)
	if err != nil {
		return err
	}
	rec := auditRecord{ActorKind: e.ActorKind, ActorID: e.ActorID, Action: e.Action, TargetKind: e.TargetKind,
		TargetID: e.TargetID, Reason: e.Reason, Meta: string(meta), IP: e.IP, At: e.At}
	return query(ctx).Create(&rec)
}

func (AuditEntries) List(ctx context.Context, f app.AuditFilter) ([]domain.AuditEntry, error) {
	q := query(ctx)
	if f.ActorKind != "" {
		q = q.Where("actor_kind", f.ActorKind)
	}
	if f.ActorID != 0 {
		q = q.Where("actor_id", f.ActorID)
	}
	if f.Action != "" {
		q = q.Where("action", f.Action)
	}
	if f.TargetKind != "" {
		q = q.Where("target_kind", f.TargetKind)
	}
	if f.TargetID != 0 {
		q = q.Where("target_id", f.TargetID)
	}
	if f.From != nil {
		q = q.Where("at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("at < ?", *f.To)
	}
	if f.Before != 0 {
		q = q.Where("id < ?", f.Before)
	}
	var recs []auditRecord
	if err := q.OrderByDesc("id").Limit(f.Limit).Find(&recs); err != nil {
		return nil, err
	}
	out := make([]domain.AuditEntry, len(recs))
	for i, r := range recs {
		var meta map[string]any
		_ = json.Unmarshal([]byte(r.Meta), &meta)
		out[i] = domain.AuditEntry{ID: r.ID, ActorKind: r.ActorKind, ActorID: r.ActorID, Action: r.Action, TargetKind: r.TargetKind,
			TargetID: r.TargetID, Reason: r.Reason, Meta: meta, IP: r.IP, At: r.At}
	}
	return out, nil
}
