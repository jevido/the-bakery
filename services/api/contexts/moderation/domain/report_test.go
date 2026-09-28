package domain

import (
	"errors"
	"testing"
	"time"
)

func TestFileReport(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name   string
		kind   string
		target uint64
		reason string
		today  int
		want   error
	}{
		{"a guild", TargetGuild, 3, "Spam boards", 0, nil},
		{"a member", TargetMember, 9, "Rude", 9, nil},
		{"yourself", TargetMember, 1, "me", 0, ErrReportSelf},
		{"no reason", TargetGuild, 3, " ", 0, ErrInvalidReason},
		{"eleventh today", TargetGuild, 3, "Spam", 10, ErrTooManyReports},
		{"a board", "board", 3, "Spam", 0, ErrInvalidReportTarget},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := FileReport(1, tt.kind, tt.target, tt.reason, tt.today, now)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if err == nil && r.Status != ReportOpen {
				t.Fatalf("report = %+v", r)
			}
		})
	}
}

func TestHandleReport(t *testing.T) {
	r, _ := FileReport(1, TargetGuild, 3, "Spam", 0, time.Now())
	if err := r.Action(7, 12, time.Now()); err != nil || r.Status != ReportActioned || r.SanctionID != 12 {
		t.Fatalf("action: %v %+v", err, r)
	}
	if err := r.Dismiss(7, time.Now()); !errors.Is(err, ErrReportHandled) {
		t.Fatalf("handled twice: %v", err)
	}
}
