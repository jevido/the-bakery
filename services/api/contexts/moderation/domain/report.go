package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Report statuses.
const (
	ReportOpen      = "open"
	ReportDismissed = "dismissed"
	ReportActioned  = "actioned"
)

// ReportsPerDay is how many reports one member may file in a day.
const ReportsPerDay = 10

var (
	ErrInvalidReportTarget = errors.New("a report is about a member or a guild")
	ErrReportSelf          = errors.New("you cannot report yourself")
	ErrReportHandled       = errors.New("this report has already been handled")
	ErrTooManyReports      = errors.New("you have filed 10 reports today; try again tomorrow")
)

// Report is a member telling the operators about a member or a guild.
type Report struct {
	ID         uint64
	ByMemberID uint64
	TargetKind string
	TargetID   uint64
	Reason     string
	Status     string
	HandledBy  uint64
	HandledAt  *time.Time
	SanctionID uint64
	CreatedAt  time.Time
}

// FileReport makes an open report; filedToday is how many the member filed
// in the last day.
func FileReport(by uint64, targetKind string, targetID uint64, reason string, filedToday int, now time.Time) (Report, error) {
	if (targetKind != TargetMember && targetKind != TargetGuild) || targetID == 0 {
		return Report{}, ErrInvalidReportTarget
	}
	if targetKind == TargetMember && targetID == by {
		return Report{}, ErrReportSelf
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > reasonMax {
		return Report{}, ErrInvalidReason
	}
	if filedToday >= ReportsPerDay {
		return Report{}, ErrTooManyReports
	}
	return Report{ByMemberID: by, TargetKind: targetKind, TargetID: targetID, Reason: reason, Status: ReportOpen, CreatedAt: now}, nil
}

// Dismiss closes an open report without action.
func (r *Report) Dismiss(operatorID uint64, now time.Time) error {
	return r.handle(ReportDismissed, operatorID, 0, now)
}

// Action closes an open report with the sanction it led to.
func (r *Report) Action(operatorID, sanctionID uint64, now time.Time) error {
	return r.handle(ReportActioned, operatorID, sanctionID, now)
}

func (r *Report) handle(status string, operatorID, sanctionID uint64, now time.Time) error {
	if r.Status != ReportOpen {
		return ErrReportHandled
	}
	r.Status, r.HandledBy, r.HandledAt, r.SanctionID = status, operatorID, &now, sanctionID
	return nil
}
