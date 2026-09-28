package app

import (
	"context"
	"errors"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

var ErrReportNotFound = errors.New("report not found")

type Reports interface {
	Add(ctx context.Context, r domain.Report) (domain.Report, error)
	Save(ctx context.Context, r domain.Report) error
	ByID(ctx context.Context, id uint64) (domain.Report, bool, error)
	// FiledSince counts a member's reports since a time.
	FiledSince(ctx context.Context, memberID uint64, since time.Time) (int, error)
	// WithStatus lists reports, oldest first ("" for all).
	WithStatus(ctx context.Context, status string) ([]domain.Report, error)
}

// FileReport lets a member tell the operators about a member or a guild.
func (s *Service) FileReport(ctx context.Context, memberID uint64, targetKind string, targetID uint64, reason string) (domain.Report, error) {
	now := s.now()
	today, err := s.reports.FiledSince(ctx, memberID, now.Add(-24*time.Hour))
	if err != nil {
		return domain.Report{}, err
	}
	r, err := domain.FileReport(memberID, targetKind, targetID, reason, today, now)
	if err != nil {
		return domain.Report{}, err
	}
	if err := s.requireTarget(ctx, targetKind, targetID); err != nil {
		return domain.Report{}, err
	}
	if r, err = s.reports.Add(ctx, r); err != nil {
		return domain.Report{}, err
	}
	_ = s.Record(ctx, domain.AuditEntry{ActorKind: domain.ActorMember, ActorID: memberID, Action: "report.filed",
		TargetKind: targetKind, TargetID: targetID, Reason: r.Reason, Meta: map[string]any{"report_id": r.ID}})
	return r, nil
}

// ReportsWithStatus lists reports for the console, oldest first.
func (s *Service) ReportsWithStatus(ctx context.Context, status string) ([]domain.Report, error) {
	return s.reports.WithStatus(ctx, status)
}

// DismissReport closes a report without action.
func (s *Service) DismissReport(ctx context.Context, operatorID, reportID uint64) (domain.Report, error) {
	return s.handleReport(ctx, operatorID, reportID, "report.dismissed", func(r *domain.Report) error { return r.Dismiss(operatorID, s.now()) })
}

// ActionReport closes a report with the sanction it led to.
func (s *Service) ActionReport(ctx context.Context, operatorID, reportID, sanctionID uint64) (domain.Report, error) {
	if _, found, err := s.sanctions.ByID(ctx, sanctionID); err != nil || !found {
		if err == nil {
			err = ErrSanctionNotFound
		}
		return domain.Report{}, err
	}
	return s.handleReport(ctx, operatorID, reportID, "report.actioned", func(r *domain.Report) error { return r.Action(operatorID, sanctionID, s.now()) })
}

func (s *Service) handleReport(ctx context.Context, operatorID, reportID uint64, action string, f func(*domain.Report) error) (domain.Report, error) {
	r, found, err := s.reports.ByID(ctx, reportID)
	if err != nil {
		return domain.Report{}, err
	}
	if !found {
		return domain.Report{}, ErrReportNotFound
	}
	if err := f(&r); err != nil {
		return domain.Report{}, err
	}
	if err := s.reports.Save(ctx, r); err != nil {
		return domain.Report{}, err
	}
	meta := map[string]any{"report_id": r.ID}
	if r.SanctionID != 0 {
		meta["sanction_id"] = r.SanctionID
	}
	_ = s.Record(ctx, domain.AuditEntry{ActorKind: domain.ActorOperator, ActorID: operatorID, Action: action,
		TargetKind: domain.TargetReport, TargetID: r.ID, Meta: meta})
	return r, nil
}

func (s *Service) requireTarget(ctx context.Context, targetKind string, targetID uint64) error {
	var exists bool
	var err error
	if targetKind == domain.TargetMember {
		exists, err = s.targets.MemberExists(ctx, targetID)
	} else {
		exists, err = s.targets.GuildExists(ctx, targetID)
	}
	if err != nil {
		return err
	}
	if !exists {
		return ErrTargetNotFound
	}
	return nil
}
