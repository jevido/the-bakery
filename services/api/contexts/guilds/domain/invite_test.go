package domain

import (
	"errors"
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }

func TestNewInviteRules(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		expiresIn *time.Duration
		maxUses   *int
		wantErr   error
	}{
		{"no end, no limit", nil, nil, nil},
		{"one day, ten uses", ptr(24 * time.Hour), ptr(10), nil},
		{"too short", ptr(time.Minute), nil, ErrInvalidInviteRule},
		{"too long", ptr(31 * 24 * time.Hour), nil, ErrInvalidInviteRule},
		{"zero uses", nil, ptr(0), ErrInvalidInviteRule},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv, _, err := NewInvite(1, 7, "code", now, tt.expiresIn, tt.maxUses)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if err == nil && tt.expiresIn != nil && !inv.ExpiresAt.Equal(now.Add(*tt.expiresIn)) {
				t.Errorf("ExpiresAt = %v", inv.ExpiresAt)
			}
		})
	}
}

func TestInviteUsable(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	inv, _, _ := NewInvite(1, 7, "code", now, ptr(time.Hour), ptr(2))
	if err := inv.Use(now); err != nil {
		t.Fatal(err)
	}
	if err := inv.Use(now); err != nil {
		t.Fatal(err)
	}
	if err := inv.Use(now); !errors.Is(err, ErrInviteUsedUp) {
		t.Errorf("third use error = %v, want ErrInviteUsedUp", err)
	}

	open, _, _ := NewInvite(1, 7, "code", now, ptr(time.Hour), nil)
	if err := open.Usable(now.Add(time.Hour)); !errors.Is(err, ErrInviteExpired) {
		t.Errorf("at expiry error = %v, want ErrInviteExpired", err)
	}
	open.Revoke(now)
	if err := open.Usable(now); !errors.Is(err, ErrInviteRevoked) {
		t.Errorf("revoked error = %v, want ErrInviteRevoked", err)
	}
}
