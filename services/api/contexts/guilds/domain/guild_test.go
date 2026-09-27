package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestFound(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{"valid", "First Colony", "First Colony", nil},
		{"trimmed", "  First Colony ", "First Colony", nil},
		{"empty", "   ", "", ErrInvalidName},
		{"60 runes", strings.Repeat("ü", 60), strings.Repeat("ü", 60), nil},
		{"61 runes", strings.Repeat("ü", 61), "", ErrInvalidName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, ev, err := Found(tt.input, 7)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Found() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if g.Name != tt.want || ev.Name != tt.want {
				t.Errorf("name = %q / event %q, want %q", g.Name, ev.Name, tt.want)
			}
			if !g.HasMember(7) || ev.FounderID != 7 {
				t.Error("founder is not the first member")
			}
		})
	}
}

func TestAddMember(t *testing.T) {
	g := Rehydrate(1, "First Colony", []uint64{7})

	ev, err := g.AddMember(8)
	if err != nil {
		t.Fatalf("AddMember(8) error = %v", err)
	}
	if ev != (MemberJoined{GuildID: 1, MemberID: 8}) {
		t.Errorf("event = %+v", ev)
	}
	if !g.HasMember(8) {
		t.Error("new member missing")
	}
	if _, err := g.AddMember(8); !errors.Is(err, ErrAlreadyMember) {
		t.Errorf("second AddMember(8) error = %v, want ErrAlreadyMember", err)
	}
	if _, err := g.AddMember(7); !errors.Is(err, ErrAlreadyMember) {
		t.Errorf("AddMember(founder) error = %v, want ErrAlreadyMember", err)
	}
}
