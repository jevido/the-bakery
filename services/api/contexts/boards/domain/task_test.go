package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestExpand(t *testing.T) {
	parent := Task{ID: 7, BoardID: 1, ColumnID: 2, Position: "a0"}
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
			got, err := Expand(tt.parent, Drafts(tt.titles), "")
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
	got, err := Expand(Task{ID: 1, BoardID: 1}, Drafts([]string{"Dig"}), "a5")
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
	if ev, err := st.Reopen(); err != nil || ev == nil || st.Done {
		t.Fatalf("Reopen() = %v, %v; done = %v", ev, err, st.Done)
	}
	if ev, _ := st.Reopen(); ev != nil {
		t.Error("reopening an open subtask announces")
	}

	top := Task{ID: 1, BoardID: 1, ColumnID: 1}
	if _, err := top.Complete(); !errors.Is(err, ErrNotSubtask) {
		t.Errorf("Complete() on a task error = %v, want ErrNotSubtask", err)
	}
	if _, err := top.Reopen(); !errors.Is(err, ErrNotSubtask) {
		t.Errorf("Reopen() on a task error = %v, want ErrNotSubtask", err)
	}
}

func TestSubtaskHasNoColumn(t *testing.T) {
	parentID := uint64(1)
	st := Task{ID: 2, BoardID: 1, ParentID: &parentID, Position: "a0"}
	if _, err := st.Move(3, "", ""); !errors.Is(err, ErrSubtaskNoColumn) {
		t.Errorf("Move() on a subtask error = %v, want ErrSubtaskNoColumn", err)
	}
	if _, err := st.Reposition("", "a0"); err != nil || st.Position >= "a0" {
		t.Errorf("Reposition() = %v, position %q", err, st.Position)
	}
	top := Task{ID: 1, ColumnID: 1, Position: "a0"}
	if _, err := top.Reposition("", ""); !errors.Is(err, ErrNotSubtask) {
		t.Errorf("Reposition() on a task error = %v, want ErrNotSubtask", err)
	}
}

func TestEditAnnouncesOnlyChanges(t *testing.T) {
	str := func(s string) *string { return &s }
	task := Task{ID: 1, BoardID: 2, Title: "Dig", Description: "Deep"}
	if ev, err := task.Edit(str("Dig"), str("Deep"), nil); err != nil || ev != nil {
		t.Errorf("unchanged Edit() = %+v, %v; want no event", ev, err)
	}
	ev, err := task.Edit(str("Dig deeper"), nil, nil)
	if err != nil || ev == nil || !ev.Title || ev.Description {
		t.Errorf("title Edit() = %+v, %v", ev, err)
	}
	ev, err = task.Edit(nil, str("Very deep"), nil)
	if err != nil || ev == nil || ev.Title || !ev.Description {
		t.Errorf("description Edit() = %+v, %v", ev, err)
	}
	if _, err := task.Edit(str(" "), nil, nil); !errors.Is(err, ErrInvalidTitle) {
		t.Errorf("empty title Edit() error = %v", err)
	}
}

func TestEditWorkType(t *testing.T) {
	str := func(s string) *string { return &s }
	task := Task{ID: 1, BoardID: 2}
	ev, err := task.Edit(nil, nil, str("coding"))
	if err != nil || ev == nil || !ev.WorkType || task.WorkType != "coding" {
		t.Fatalf("setting a work type: %+v, %v, %q", ev, err, task.WorkType)
	}
	if ev, _ := task.Edit(nil, nil, str("coding")); ev != nil {
		t.Error("the same work type again announces a change")
	}
	if ev, _ := task.Edit(nil, nil, str("")); ev == nil || task.WorkType != "" {
		t.Error("clearing the work type")
	}
}

func TestWorkTypes(t *testing.T) {
	defaults := DefaultsFor(1)
	if len(defaults) != 7 || defaults[0].Key != "coding" || defaults[6].Name != "Ops" {
		t.Fatalf("defaults = %+v", defaults)
	}
	for i := 1; i < len(defaults); i++ {
		if defaults[i-1].Position >= defaults[i].Position {
			t.Errorf("defaults out of order at %d", i)
		}
	}
	for _, key := range []string{"lore", "a", "ops-2", "x" + strings.Repeat("y", 31)} {
		if _, err := NewWorkType(1, key, "Lore", ""); err != nil {
			t.Errorf("NewWorkType(%q) error = %v", key, err)
		}
	}
	for _, key := range []string{"", "Lore", "2d", "-x", "a b", "x" + strings.Repeat("y", 32)} {
		if _, err := NewWorkType(1, key, "Lore", ""); !errors.Is(err, ErrInvalidWorkTypeKey) {
			t.Errorf("NewWorkType(%q) error = %v, want ErrInvalidWorkTypeKey", key, err)
		}
	}
	if _, err := NewWorkType(1, "lore", " ", ""); !errors.Is(err, ErrInvalidWorkTypeName) {
		t.Errorf("empty name error = %v", err)
	}
}
