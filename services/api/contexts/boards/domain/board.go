// Package domain holds the boards model: boards, tasks, columns and
// positions. It imports nothing outside the standard library.
package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	boardNameMax = 60
	taskTitleMax = 200
)

var (
	ErrInvalidBoardName = errors.New("board name must be 1 to 60 characters")
	ErrInvalidTitle     = errors.New("task title must be 1 to 200 characters")
	ErrInvalidColumn    = errors.New("column must be one of backlog, todo, doing, done")
)

// Column is where a task stands on its board.
type Column string

const (
	Backlog Column = "backlog"
	Todo    Column = "todo"
	Doing   Column = "doing"
	Done    Column = "done"
)

// Columns lists every column in board order.
var Columns = []Column{Backlog, Todo, Doing, Done}

func ParseColumn(s string) (Column, error) {
	for _, c := range Columns {
		if string(c) == s {
			return c, nil
		}
	}
	return "", ErrInvalidColumn
}

// Board belongs to one guild, known here only by its id.
type Board struct {
	ID      uint64
	GuildID uint64
	Name    string
}

func NewBoard(guildID uint64, name string) (Board, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > boardNameMax {
		return Board{}, ErrInvalidBoardName
	}
	return Board{GuildID: guildID, Name: name}, nil
}

// Task is its own aggregate: moving one task never touches the board or
// the other tasks.
type Task struct {
	ID          uint64
	BoardID     uint64
	Title       string
	Description string
	Column      Column
	Position    string
}

// TaskCreated is announced when a task is added to a board.
type TaskCreated struct {
	TaskID   uint64
	BoardID  uint64
	Column   Column
	Position string
}

// TaskMoved is announced when a task changes column or position.
type TaskMoved struct {
	TaskID   uint64
	BoardID  uint64
	From     Column
	To       Column
	Position string
}

// NewTask creates a task at the bottom of the backlog. last is the position
// of the backlog's current last task ("" when it is empty).
func NewTask(boardID uint64, title, description, last string) (Task, TaskCreated, error) {
	title, err := cleanTitle(title)
	if err != nil {
		return Task{}, TaskCreated{}, err
	}
	pos, err := KeyBetween(last, "")
	if err != nil {
		return Task{}, TaskCreated{}, err
	}
	t := Task{BoardID: boardID, Title: title, Description: description, Column: Backlog, Position: pos}
	return t, TaskCreated{BoardID: boardID, Column: Backlog, Position: pos}, nil
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

// Move puts the task in column between the tasks at positions above and
// below ("" for the column's top or bottom end).
func (t *Task) Move(column Column, above, below string) (TaskMoved, error) {
	if _, err := ParseColumn(string(column)); err != nil {
		return TaskMoved{}, err
	}
	pos, err := KeyBetween(above, below)
	if err != nil {
		return TaskMoved{}, err
	}
	ev := TaskMoved{TaskID: t.ID, BoardID: t.BoardID, From: t.Column, To: column, Position: pos}
	t.Column, t.Position = column, pos
	return ev, nil
}

func cleanTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if n := utf8.RuneCountInString(title); n < 1 || n > taskTitleMax {
		return "", ErrInvalidTitle
	}
	return title, nil
}
