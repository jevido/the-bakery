package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestExpand(t *testing.T) {
	parent := Task{ID: 7, BoardID: 1, Column: Todo, Position: "a0"}
	parentID := uint64(7)
	subtask := Task{ID: 8, BoardID: 1, ParentID: &parentID, Position: "a0"}
	many := make([]string, 51)
	for i := range many {
		many[i] = "Haul steel"
	}

	tests := []struct {
		name    string
		parent  Task
		titles  []string
		wantErr error
	}{
		{"three", parent, []string{"Dig", "Wall", "Roof"}, nil},
		{"fifty", parent, many[:50], nil},
		{"none", parent, nil, ErrTooManySubtasks},
		{"fifty-one", parent, many, ErrTooManySubtasks},
		{"subtask of a subtask", subtask, []string{"Dig"}, ErrNestedSubtask},
		{"empty title", parent, []string{"Dig", " "}, ErrInvalidTitle},
		{"long title", parent, []string{strings.Repeat("x", 201)}, ErrInvalidTitle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Expand(tt.parent, tt.titles, "")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Expand() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(got) != len(tt.titles) {
				t.Fatalf("got %d subtasks, want %d", len(got), len(tt.titles))
			}
			for i, st := range got {
				if st.ParentID == nil || *st.ParentID != tt.parent.ID || st.BoardID != tt.parent.BoardID {
					t.Errorf("subtask %d parent/board = %v/%d", i, st.ParentID, st.BoardID)
				}
				if i > 0 && got[i-1].Position >= st.Position {
					t.Errorf("positions out of order: %q then %q", got[i-1].Position, st.Position)
				}
			}
		})
	}
}

func TestExpandGoesBelowExistingSubtasks(t *testing.T) {
	got, err := Expand(Task{ID: 1, BoardID: 1}, []string{"Dig"}, "a5")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Position <= "a5" {
		t.Errorf("position %q not below a5", got[0].Position)
	}
}

func TestCompleteAndReopen(t *testing.T) {
	parentID := uint64(1)
	st := Task{ID: 2, BoardID: 1, ParentID: &parentID}
	ev, err := st.Complete()
	if err != nil || ev == nil || !st.Done {
		t.Fatalf("Complete() = %v, %v; done = %v", ev, err, st.Done)
	}
	if ev, _ := st.Complete(); ev != nil {
		t.Error("completing twice announces twice")
	}
	if err := st.Reopen(); err != nil || st.Done {
		t.Fatalf("Reopen() = %v; done = %v", err, st.Done)
	}

	top := Task{ID: 1, BoardID: 1, Column: Backlog}
	if _, err := top.Complete(); !errors.Is(err, ErrNotSubtask) {
		t.Errorf("Complete() on a task error = %v, want ErrNotSubtask", err)
	}
	if err := top.Reopen(); !errors.Is(err, ErrNotSubtask) {
		t.Errorf("Reopen() on a task error = %v, want ErrNotSubtask", err)
	}
}

func TestSubtaskHasNoColumn(t *testing.T) {
	parentID := uint64(1)
	st := Task{ID: 2, BoardID: 1, ParentID: &parentID, Position: "a0"}
	if _, err := st.Move(Doing, "", ""); !errors.Is(err, ErrSubtaskNoColumn) {
		t.Errorf("Move() on a subtask error = %v, want ErrSubtaskNoColumn", err)
	}
	if err := st.Reposition("", "a0"); err != nil || st.Position >= "a0" {
		t.Errorf("Reposition() = %v, position %q", err, st.Position)
	}
	top := Task{ID: 1, Column: Backlog, Position: "a0"}
	if err := top.Reposition("", ""); !errors.Is(err, ErrNotSubtask) {
		t.Errorf("Reposition() on a task error = %v, want ErrNotSubtask", err)
	}
}

func TestEditAnnouncesOnlyChanges(t *testing.T) {
	str := func(s string) *string { return &s }
	task := Task{ID: 1, BoardID: 2, Title: "Dig", Description: "Deep"}
	if ev, err := task.Edit(str("Dig"), str("Deep")); err != nil || ev != nil {
		t.Errorf("unchanged Edit() = %+v, %v; want no event", ev, err)
	}
	ev, err := task.Edit(str("Dig deeper"), nil)
	if err != nil || ev == nil || !ev.Title || ev.Description {
		t.Errorf("title Edit() = %+v, %v", ev, err)
	}
	ev, err = task.Edit(nil, str("Very deep"))
	if err != nil || ev == nil || ev.Title || !ev.Description {
		t.Errorf("description Edit() = %+v, %v", ev, err)
	}
	if _, err := task.Edit(str(" "), nil); !errors.Is(err, ErrInvalidTitle) {
		t.Errorf("empty title Edit() error = %v", err)
	}
}
