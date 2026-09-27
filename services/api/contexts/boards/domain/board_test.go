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

func TestNewTaskLandsAtBottomOfItsColumn(t *testing.T) {
	first, _, err := NewTask(1, 10, "Build a research bench", "", "")
	if err != nil {
		t.Fatal(err)
	}
	second, ev, err := NewTask(1, 10, "Plant rice", "", first.Position)
	if err != nil {
		t.Fatal(err)
	}
	if second.ColumnID != 10 || ev.ColumnID != 10 {
		t.Errorf("column = %d, want 10", second.ColumnID)
	}
	if second.Position <= first.Position {
		t.Errorf("second %q not after first %q", second.Position, first.Position)
	}
	for _, title := range []string{" ", strings.Repeat("x", 201)} {
		if _, _, err := NewTask(1, 10, title, "", ""); !errors.Is(err, ErrInvalidTitle) {
			t.Errorf("NewTask(%q) error = %v, want ErrInvalidTitle", title, err)
		}
	}
}

func TestMove(t *testing.T) {
	const backlog, todo = 10, 11
	a, _, _ := NewTask(1, backlog, "a", "", "")
	b, _, _ := NewTask(1, backlog, "b", "", a.Position)
	c, _, _ := NewTask(1, backlog, "c", "", b.Position)

	// c goes to the top of todo, then a goes above it.
	ev, err := c.Move(todo, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if ev.From != backlog || ev.To != todo || c.ColumnID != todo {
		t.Errorf("event = %+v, column = %d", ev, c.ColumnID)
	}
	if _, err := a.Move(todo, "", c.Position); err != nil {
		t.Fatal(err)
	}
	if a.Position >= c.Position {
		t.Errorf("a %q not above c %q", a.Position, c.Position)
	}
	// b slots between them.
	if _, err := b.Move(todo, a.Position, c.Position); err != nil {
		t.Fatal(err)
	}
	if a.Position >= b.Position || b.Position >= c.Position {
		t.Errorf("order a=%q b=%q c=%q", a.Position, b.Position, c.Position)
	}
	if _, err := b.Move(todo, c.Position, a.Position); !errors.Is(err, ErrInvalidPosition) {
		t.Errorf("Move with swapped neighbours error = %v, want ErrInvalidPosition", err)
	}
}

// boardWithIDs is a new board whose columns have ids 1 to 4, as if stored.
func boardWithIDs(t *testing.T) Board {
	t.Helper()
	b, err := NewBoard(1, "Getting settled")
	if err != nil {
		t.Fatal(err)
	}
	for i := range b.Columns {
		b.Columns[i].ID = uint64(i + 1)
	}
	return b
}

func names(b Board) []string {
	var out []string
	for _, c := range b.Columns {
		out = append(out, c.Name)
	}
	return out
}

func TestNewBoardHasTheDefaultColumns(t *testing.T) {
	b := boardWithIDs(t)
	if got := names(b); strings.Join(got, ",") != "Backlog,To do,Doing,Done" {
		t.Errorf("columns = %v", got)
	}
	for i := 1; i < len(b.Columns); i++ {
		if b.Columns[i-1].Position >= b.Columns[i].Position {
			t.Errorf("columns out of order at %d", i)
		}
	}
	if b.First().Name != "Backlog" {
		t.Errorf("first = %q", b.First().Name)
	}
}

func TestColumnNames(t *testing.T) {
	b := boardWithIDs(t)
	tests := []struct {
		name    string
		column  string
		wantErr error
	}{
		{"new", "Review", nil},
		{"trimmed", "  Blocked  ", nil},
		{"longest", strings.Repeat("x", 40), nil},
		{"empty", " ", ErrInvalidColumnName},
		{"too long", strings.Repeat("x", 41), ErrInvalidColumnName},
		{"duplicate ignoring case", "to DO", ErrDuplicateColumn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := b
			b.Columns = append([]Column{}, b.Columns...)
			if _, err := b.AddColumn(tt.column); !errors.Is(err, tt.wantErr) {
				t.Errorf("AddColumn(%q) error = %v, want %v", tt.column, err, tt.wantErr)
			}
		})
	}
	if _, err := b.RenameColumn(2, "to do"); err != nil {
		t.Errorf("renaming a column to itself in another case: %v", err)
	}
	if _, err := b.RenameColumn(2, "Doing"); !errors.Is(err, ErrDuplicateColumn) {
		t.Errorf("RenameColumn to another's name error = %v", err)
	}
	if _, err := b.RenameColumn(99, "Review"); !errors.Is(err, ErrColumnNotFound) {
		t.Errorf("RenameColumn of a stranger error = %v", err)
	}
	if c, ok := b.ColumnNamed(" doing "); !ok || c.ID != 3 {
		t.Errorf("ColumnNamed(doing) = %+v, %v", c, ok)
	}
}

func TestMoveColumn(t *testing.T) {
	b := boardWithIDs(t) // 1 Backlog, 2 To do, 3 Doing, 4 Done
	id := func(v uint64) *uint64 { return &v }
	if _, err := b.MoveColumn(4, nil, id(1)); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(names(b), ","); got != "Done,Backlog,To do,Doing" {
		t.Errorf("after moving Done first: %s", got)
	}
	if _, err := b.MoveColumn(4, id(3), nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(names(b), ","); got != "Backlog,To do,Doing,Done" {
		t.Errorf("after moving Done after Doing: %s", got)
	}
	if _, err := b.MoveColumn(1, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(names(b), ","); got != "To do,Doing,Done,Backlog" {
		t.Errorf("after moving Backlog to the end: %s", got)
	}
	if _, err := b.MoveColumn(2, id(2), nil); !errors.Is(err, ErrInvalidColumnOrder) {
		t.Errorf("MoveColumn after itself error = %v", err)
	}
	if _, err := b.MoveColumn(2, id(4), id(3)); !errors.Is(err, ErrInvalidColumnOrder) {
		t.Errorf("MoveColumn between swapped neighbours error = %v", err)
	}
}

func TestRemoveColumn(t *testing.T) {
	b := boardWithIDs(t)
	if err := b.RemoveColumn(2, true); !errors.Is(err, ErrColumnNotEmpty) {
		t.Errorf("removing a column with tasks error = %v", err)
	}
	for _, id := range []uint64{2, 3, 4} {
		if err := b.RemoveColumn(id, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.RemoveColumn(1, false); !errors.Is(err, ErrLastColumn) {
		t.Errorf("removing the last column error = %v", err)
	}
	if err := b.RemoveColumn(9, false); !errors.Is(err, ErrColumnNotFound) {
		t.Errorf("removing a stranger error = %v", err)
	}
}
