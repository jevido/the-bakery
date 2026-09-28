package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	taskTitleMax = 200
	expandMax    = 50
)

var (
	ErrInvalidTitle    = errors.New("task title must be 1 to 200 characters")
	ErrNestedSubtask   = errors.New("a subtask cannot have subtasks")
	ErrTooManySubtasks = errors.New("expanding takes 1 to 50 subtasks")
	ErrNotSubtask      = errors.New("only a subtask can be ticked off")
	ErrSubtaskNoColumn = errors.New("a subtask is not in a column; reorder it under its parent")
)

// Task is its own aggregate: moving one task never touches the board or
// the other tasks. A task on the board stands in one of the board's columns
// (ColumnID). A task with a ParentID is a subtask: it has no column
// (ColumnID 0), is ordered under its parent, and can be ticked off.
type Task struct {
	ID          uint64
	BoardID     uint64
	ParentID    *uint64
	ColumnID    uint64
	Title       string
	Description string
	Position    string
	Done        bool
	// WorkType is the key of one of the guild's work types, or "" for none.
	WorkType string
	// PrioritizedAgentID is the agent (the agents context's id) only who
	// may take the task, ahead of its work priorities; nil for none.
	PrioritizedAgentID *uint64
	// Forbidden keeps every agent off the task; people still work it.
	Forbidden bool
}

func (t Task) IsSubtask() bool { return t.ParentID != nil }

// Every event carries ActorID, the member whose command caused it. The
// aggregate does not know who that is; the use case fills it in.

// TaskCreated is announced when a task is added to a board.
type TaskCreated struct {
	TaskID   uint64
	BoardID  uint64
	ActorID  uint64
	ColumnID uint64
	Position string
}

// TaskEdited is announced when a task's title and/or description changes.
// ParentID is set when the task is a subtask.
type TaskEdited struct {
	TaskID      uint64
	BoardID     uint64
	ActorID     uint64
	ParentID    *uint64
	Title       bool
	Description bool
	WorkType    bool
	// Drafting is true when who may take the task changed (prioritized or
	// forbidden).
	Drafting bool
}

// TaskDeleted is announced when a task (with its subtasks) or a subtask is
// deleted. ParentID is set for a subtask.
type TaskDeleted struct {
	TaskID   uint64
	BoardID  uint64
	ActorID  uint64
	ParentID *uint64
}

// TaskMoved is announced when a task changes column or position.
type TaskMoved struct {
	TaskID   uint64
	BoardID  uint64
	ActorID  uint64
	From     uint64
	To       uint64
	Position string
}

// SubtaskAdded is announced when a task gets a subtask.
type SubtaskAdded struct {
	SubtaskID uint64
	ParentID  uint64
	BoardID   uint64
	ActorID   uint64
	Title     string
}

// SubtaskMoved is announced when a subtask moves among its siblings.
type SubtaskMoved struct {
	SubtaskID uint64
	ParentID  uint64
	BoardID   uint64
	ActorID   uint64
}

// SubtaskReopened is announced when a ticked-off subtask is opened again.
type SubtaskReopened struct {
	SubtaskID uint64
	ParentID  uint64
	BoardID   uint64
	ActorID   uint64
}

// SubtaskCompleted is announced when a subtask is ticked off.
type SubtaskCompleted struct {
	SubtaskID uint64
	ParentID  uint64
	BoardID   uint64
	ActorID   uint64
	Title     string
}

// NewTask creates a task at the bottom of a column (the board's first, when
// created from the board). last is the position of the column's current
// last task ("" when it is empty).
func NewTask(boardID, columnID uint64, title, description, last string) (Task, TaskCreated, error) {
	title, err := cleanTitle(title)
	if err != nil {
		return Task{}, TaskCreated{}, err
	}
	pos, err := KeyBetween(last, "")
	if err != nil {
		return Task{}, TaskCreated{}, err
	}
	t := Task{BoardID: boardID, ColumnID: columnID, Title: title, Description: description, Position: pos}
	return t, TaskCreated{BoardID: boardID, ColumnID: columnID, Position: pos}, nil
}

// SubtaskDraft is one subtask to add: a title, and optionally a
// description and a work type (a key of the guild's; the use case checks).
type SubtaskDraft struct {
	Title       string
	Description string
	WorkType    string
}

// Drafts makes plain drafts from titles.
func Drafts(titles []string) []SubtaskDraft {
	out := make([]SubtaskDraft, len(titles))
	for i, t := range titles {
		out[i] = SubtaskDraft{Title: t}
	}
	return out
}

// Expand makes subtasks of parent from drafts, in order, below the parent's
// current last subtask (at position last, "" when it has none). Subtasks
// are one level deep, so a subtask cannot be expanded.
func Expand(parent Task, drafts []SubtaskDraft, last string) ([]Task, error) {
	if parent.IsSubtask() {
		return nil, ErrNestedSubtask
	}
	if len(drafts) < 1 || len(drafts) > expandMax {
		return nil, ErrTooManySubtasks
	}
	subtasks := make([]Task, len(drafts))
	for i, d := range drafts {
		title, err := cleanTitle(d.Title)
		if err != nil {
			return nil, err
		}
		pos, err := KeyBetween(last, "")
		if err != nil {
			return nil, err
		}
		parentID := parent.ID
		subtasks[i] = Task{BoardID: parent.BoardID, ParentID: &parentID, Title: title, Position: pos,
			Description: strings.TrimSpace(d.Description), WorkType: d.WorkType}
		last = pos
	}
	return subtasks, nil
}

// Added is the event for a subtask that has just been stored.
func (t Task) Added() SubtaskAdded {
	return SubtaskAdded{SubtaskID: t.ID, ParentID: *t.ParentID, BoardID: t.BoardID, Title: t.Title}
}

// Edit changes the title, description and/or work type (nil leaves one as
// it is; an empty work type clears it). The use case makes sure the work type
// is one of the guild's. The event is announced only when something actually
// changed.
func (t *Task) Edit(title, description, workType *string) (*TaskEdited, error) {
	ev := TaskEdited{TaskID: t.ID, BoardID: t.BoardID, ParentID: t.ParentID}
	if title != nil {
		before := t.Title
		if err := t.Rename(*title); err != nil {
			return nil, err
		}
		ev.Title = t.Title != before
	}
	if description != nil {
		ev.Description = t.Description != *description
		t.Describe(*description)
	}
	if workType != nil {
		ev.WorkType = t.WorkType != *workType
		t.WorkType = *workType
	}
	if !ev.Title && !ev.Description && !ev.WorkType {
		return nil, nil
	}
	return &ev, nil
}

func (t *Task) Rename(title string) error {
	title, err := cleanTitle(title)
	if err != nil {
		return err
	}
	t.Title = title
	return nil
}

func (t *Task) Describe(description string) {
	t.Description = description
}

// Complete ticks a subtask off. The event is announced only when it was
// still open.
func (t *Task) Complete() (*SubtaskCompleted, error) {
	if !t.IsSubtask() {
		return nil, ErrNotSubtask
	}
	if t.Done {
		return nil, nil
	}
	t.Done = true
	return &SubtaskCompleted{SubtaskID: t.ID, ParentID: *t.ParentID, BoardID: t.BoardID, Title: t.Title}, nil
}

// Reopen unticks a subtask. The event is announced only when it was done.
func (t *Task) Reopen() (*SubtaskReopened, error) {
	if !t.IsSubtask() {
		return nil, ErrNotSubtask
	}
	if !t.Done {
		return nil, nil
	}
	t.Done = false
	return &SubtaskReopened{SubtaskID: t.ID, ParentID: *t.ParentID, BoardID: t.BoardID}, nil
}

// Deleted is the event for deleting this task.
func (t Task) Deleted() TaskDeleted {
	return TaskDeleted{TaskID: t.ID, BoardID: t.BoardID, ParentID: t.ParentID}
}

// Move puts the task in the column between the tasks at positions above
// and below ("" for the column's top or bottom end). The use case makes
// sure the column is one of the task's board.
func (t *Task) Move(columnID uint64, above, below string) (TaskMoved, error) {
	if t.IsSubtask() {
		return TaskMoved{}, ErrSubtaskNoColumn
	}
	pos, err := KeyBetween(above, below)
	if err != nil {
		return TaskMoved{}, err
	}
	ev := TaskMoved{TaskID: t.ID, BoardID: t.BoardID, From: t.ColumnID, To: columnID, Position: pos}
	t.ColumnID, t.Position = columnID, pos
	return ev, nil
}

// Reposition puts a subtask between its siblings at positions above and
// below ("" for the first or last place).
func (t *Task) Reposition(above, below string) (SubtaskMoved, error) {
	if !t.IsSubtask() {
		return SubtaskMoved{}, ErrNotSubtask
	}
	pos, err := KeyBetween(above, below)
	if err != nil {
		return SubtaskMoved{}, err
	}
	t.Position = pos
	return SubtaskMoved{SubtaskID: t.ID, ParentID: *t.ParentID, BoardID: t.BoardID}, nil
}

func cleanTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if n := utf8.RuneCountInString(title); n < 1 || n > taskTitleMax {
		return "", ErrInvalidTitle
	}
	return title, nil
}
