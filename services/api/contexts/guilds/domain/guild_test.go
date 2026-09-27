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
	g := Rehydrate(1, "First Colony", false, []uint64{7})

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

func TestArchiveAndRestore(t *testing.T) {
	g := Rehydrate(1, "First Colony", false, []uint64{7, 8})
	if _, err := g.Archive(7); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Archive(7); !errors.Is(err, ErrArchived) {
		t.Errorf("second Archive error = %v", err)
	}
	for name, err := range map[string]error{
		"rename": func() error { _, err := g.Rename("New"); return err }(),
		"add":    func() error { _, err := g.AddMember(9); return err }(),
		"leave":  func() error { _, err := g.Leave(8); return err }(),
		"remove": func() error { _, err := g.RemoveMember(8, 7); return err }(),
	} {
		if !errors.Is(err, ErrArchived) {
			t.Errorf("%s on archived guild: error = %v, want ErrArchived", name, err)
		}
	}
	if _, err := g.Restore(8); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Restore(8); !errors.Is(err, ErrNotArchived) {
		t.Errorf("second Restore error = %v", err)
	}
	if ev, err := g.Rename("  Second Colony "); err != nil || g.Name != "Second Colony" || ev.Name != "Second Colony" {
		t.Errorf("Rename = %+v, %v; name %q", ev, err, g.Name)
	}
	if _, err := g.Rename(""); !errors.Is(err, ErrInvalidName) {
		t.Errorf("Rename(\"\") error = %v", err)
	}
}

func TestLeaveAndRemove(t *testing.T) {
	g := Rehydrate(1, "First Colony", false, []uint64{7, 8, 9})
	if _, err := g.RemoveMember(9, 7); err != nil || g.HasMember(9) {
		t.Fatalf("RemoveMember: %v", err)
	}
	if _, err := g.RemoveMember(9, 7); !errors.Is(err, ErrNotAMember) {
		t.Errorf("remove twice error = %v", err)
	}
	if _, err := g.Leave(8); err != nil || g.HasMember(8) {
		t.Fatalf("Leave: %v", err)
	}
	if _, err := g.Leave(7); !errors.Is(err, ErrLastMember) {
		t.Errorf("last member Leave error = %v, want ErrLastMember", err)
	}
	if !g.HasMember(7) {
		t.Error("last member was removed")
	}
}
