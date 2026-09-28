package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewSanction(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	tomorrow, yesterday := now.Add(24*time.Hour), now.Add(-time.Hour)
	tests := []struct {
		name   string
		target string
		kind   string
		reason string
		until  *time.Time
		want   error
	}{
		{"suspension", TargetMember, Suspension, "spam", &tomorrow, nil},
		{"ban", TargetGuild, Ban, "spam", nil, nil},
		{"suspension without end", TargetMember, Suspension, "spam", nil, ErrSuspensionNeedsUntil},
		{"suspension in the past", TargetMember, Suspension, "spam", &yesterday, ErrSuspensionNeedsUntil},
		{"ban with end", TargetMember, Ban, "spam", &tomorrow, ErrBanHasNoUntil},
		{"no reason", TargetMember, Ban, "  ", nil, ErrInvalidReason},
		{"long reason", TargetMember, Ban, strings.Repeat("x", 1001), nil, ErrInvalidReason},
		{"made-up kind", TargetMember, "exile", "spam", nil, ErrInvalidSanctionKind},
		{"made-up target", "board", Ban, "spam", nil, ErrInvalidSanctionTarget},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewSanction(tt.target, 5, tt.kind, tt.reason, tt.until, 1, now); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSanctionActiveAndLift(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	until := now.Add(time.Hour)
	s, _ := NewSanction(TargetMember, 5, Suspension, "spam", &until, 1, now)
	if !s.ActiveAt(now) || s.ActiveAt(until.Add(time.Second)) {
		t.Fatal("a suspension holds until its end")
	}
	b, _ := NewSanction(TargetGuild, 5, Ban, "spam", nil, 1, now)
	if !b.ActiveAt(now.Add(1000 * time.Hour)) {
		t.Fatal("a ban does not end by itself")
	}
	if err := b.Lift(now); err != nil || b.ActiveAt(now) {
		t.Fatalf("lift: %v", err)
	}
	if err := b.Lift(now); !errors.Is(err, ErrSanctionLifted) {
		t.Fatalf("lifted twice: %v", err)
	}
}
