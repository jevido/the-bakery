package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewBoard(t *testing.T) {
	if _, err := NewBoard(1, "Getting settled"); err != nil {
		t.Fatalf("NewBoard() error = %v", err)
	}
	for _, name := range []string{"", "  ", strings.Repeat("x", 61)} {
		if _, err := NewBoard(1, name); !errors.Is(err, ErrInvalidBoardName) {
			t.Errorf("NewBoard(%q) error = %v, want ErrInvalidBoardName", name, err)
		}
	}
}

func TestNewTaskLandsAtBottomOfBacklog(t *testing.T) {
	first, _, err := NewTask(1, "Build a research bench", "", "")
	if err != nil {
		t.Fatal(err)
	}
	second, ev, err := NewTask(1, "Plant rice", "", first.Position)
	if err != nil {
		t.Fatal(err)
	}
	if second.Column != Backlog || ev.Column != Backlog {
		t.Errorf("column = %q, want backlog", second.Column)
	}
	if second.Position <= first.Position {
		t.Errorf("second %q not after first %q", second.Position, first.Position)
	}
	for _, title := range []string{" ", strings.Repeat("x", 201)} {
		if _, _, err := NewTask(1, title, "", ""); !errors.Is(err, ErrInvalidTitle) {
			t.Errorf("NewTask(%q) error = %v, want ErrInvalidTitle", title, err)
		}
	}
}

func TestMove(t *testing.T) {
	a, _, _ := NewTask(1, "a", "", "")
	b, _, _ := NewTask(1, "b", "", a.Position)
	c, _, _ := NewTask(1, "c", "", b.Position)

	// c goes to the top of todo, then a goes above it.
	ev, err := c.Move(Todo, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if ev.From != Backlog || ev.To != Todo || c.Column != Todo {
		t.Errorf("event = %+v, column = %q", ev, c.Column)
	}
	if _, err := a.Move(Todo, "", c.Position); err != nil {
		t.Fatal(err)
	}
	if a.Position >= c.Position {
		t.Errorf("a %q not above c %q", a.Position, c.Position)
	}
	// b slots between them.
	if _, err := b.Move(Todo, a.Position, c.Position); err != nil {
		t.Fatal(err)
	}
	if a.Position >= b.Position || b.Position >= c.Position {
		t.Errorf("order a=%q b=%q c=%q", a.Position, b.Position, c.Position)
	}
	if _, err := b.Move("archive", "", ""); !errors.Is(err, ErrInvalidColumn) {
		t.Errorf("Move(archive) error = %v, want ErrInvalidColumn", err)
	}
	if _, err := b.Move(Todo, c.Position, a.Position); !errors.Is(err, ErrInvalidPosition) {
		t.Errorf("Move with swapped neighbours error = %v, want ErrInvalidPosition", err)
	}
}
