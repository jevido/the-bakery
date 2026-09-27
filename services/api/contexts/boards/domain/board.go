// Package domain holds the boards model: boards and their columns, tasks
// and subtasks, comments and positions. It imports nothing outside the
// standard library.
package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	boardNameMax  = 60
	columnNameMax = 40
)

var (
	ErrInvalidBoardName   = errors.New("board name must be 1 to 60 characters")
	ErrInvalidColumnName  = errors.New("column name must be 1 to 40 characters")
	ErrDuplicateColumn    = errors.New("this board already has a column with that name")
	ErrColumnNotFound     = errors.New("this board has no such column")
	ErrColumnNotEmpty     = errors.New("only an empty column can be deleted; move its tasks first")
	ErrLastColumn         = errors.New("a board keeps at least one column")
	ErrInvalidColumnOrder = errors.New("neighbour must be another column of this board")
)

// DefaultColumns are the columns every new board starts with.
var DefaultColumns = []string{"Backlog", "To do", "Doing", "Done"}

// Column is a named lane of one board. Position orders the board's columns.
type Column struct {
	ID       uint64
	BoardID  uint64
	Name     string
	Position string
}

// Board belongs to one guild, known here only by its id, and owns its
// columns, in order.
type Board struct {
	ID      uint64
	GuildID uint64
	Name    string
	Columns []Column
}

// ColumnCreated, ColumnRenamed, ColumnMoved and ColumnDeleted are announced
// when a board's columns change. The use case fills in ActorID.
type ColumnCreated struct {
	ColumnID uint64
	BoardID  uint64
	ActorID  uint64
	Name     string
	Position string
}

type ColumnRenamed struct {
	ColumnID uint64
	BoardID  uint64
	ActorID  uint64
	Name     string
}

type ColumnMoved struct {
	ColumnID uint64
	BoardID  uint64
	ActorID  uint64
	Position string
}

type ColumnDeleted struct {
	ColumnID uint64
	BoardID  uint64
	ActorID  uint64
}

// NewBoard makes a board with the default columns.
func NewBoard(guildID uint64, name string) (Board, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > boardNameMax {
		return Board{}, ErrInvalidBoardName
	}
	b := Board{GuildID: guildID, Name: name}
	for _, c := range DefaultColumns {
		if _, err := b.AddColumn(c); err != nil {
			return Board{}, err
		}
	}
	return b, nil
}

// Column finds one of the board's columns by id.
func (b Board) Column(id uint64) (Column, bool) {
	for _, c := range b.Columns {
		if c.ID == id {
			return c, true
		}
	}
	return Column{}, false
}

// ColumnNamed finds a column by name, ignoring case and surrounding spaces.
func (b Board) ColumnNamed(name string) (Column, bool) {
	name = strings.TrimSpace(name)
	for _, c := range b.Columns {
		if strings.EqualFold(c.Name, name) {
			return c, true
		}
	}
	return Column{}, false
}

// First is the column new tasks land in.
func (b Board) First() Column {
	return b.Columns[0]
}

// AddColumn puts a new column at the end of the board.
func (b *Board) AddColumn(name string) (Column, error) {
	name, err := b.checkName(name, 0)
	if err != nil {
		return Column{}, err
	}
	last := ""
	if len(b.Columns) > 0 {
		last = b.Columns[len(b.Columns)-1].Position
	}
	pos, err := KeyBetween(last, "")
	if err != nil {
		return Column{}, err
	}
	c := Column{BoardID: b.ID, Name: name, Position: pos}
	b.Columns = append(b.Columns, c)
	return c, nil
}

// RenameColumn gives a column a new name, unique on the board.
func (b *Board) RenameColumn(id uint64, name string) (Column, error) {
	i := b.index(id)
	if i < 0 {
		return Column{}, ErrColumnNotFound
	}
	name, err := b.checkName(name, id)
	if err != nil {
		return Column{}, err
	}
	b.Columns[i].Name = name
	return b.Columns[i], nil
}

// MoveColumn puts a column right after afterID and/or right before
// beforeID; with neither, at the end.
func (b *Board) MoveColumn(id uint64, afterID, beforeID *uint64) (Column, error) {
	i := b.index(id)
	if i < 0 {
		return Column{}, ErrColumnNotFound
	}
	moved := b.Columns[i]
	others := append(append([]Column{}, b.Columns[:i]...), b.Columns[i+1:]...)
	at := func(ref uint64) int {
		for j, c := range others {
			if c.ID == ref {
				return j
			}
		}
		return -1
	}
	above, below := "", ""
	switch {
	case afterID != nil:
		j := at(*afterID)
		if j < 0 {
			return Column{}, ErrInvalidColumnOrder
		}
		above = others[j].Position
		if beforeID != nil {
			k := at(*beforeID)
			if k < 0 {
				return Column{}, ErrInvalidColumnOrder
			}
			below = others[k].Position
		} else if j+1 < len(others) {
			below = others[j+1].Position
		}
	case beforeID != nil:
		k := at(*beforeID)
		if k < 0 {
			return Column{}, ErrInvalidColumnOrder
		}
		below = others[k].Position
		if k > 0 {
			above = others[k-1].Position
		}
	default:
		if len(others) > 0 {
			above = others[len(others)-1].Position
		}
	}
	pos, err := KeyBetween(above, below)
	if err != nil {
		return Column{}, ErrInvalidColumnOrder
	}
	moved.Position = pos
	b.Columns = others
	j := 0
	for j < len(b.Columns) && b.Columns[j].Position < pos {
		j++
	}
	b.Columns = append(b.Columns[:j], append([]Column{moved}, b.Columns[j:]...)...)
	return moved, nil
}

// RemoveColumn takes an empty column off the board. hasTasks says whether
// any task stands in it; the tasks are their own aggregates, so the caller
// asks them.
func (b *Board) RemoveColumn(id uint64, hasTasks bool) error {
	i := b.index(id)
	if i < 0 {
		return ErrColumnNotFound
	}
	if hasTasks {
		return ErrColumnNotEmpty
	}
	if len(b.Columns) == 1 {
		return ErrLastColumn
	}
	b.Columns = append(b.Columns[:i], b.Columns[i+1:]...)
	return nil
}

func (b Board) index(id uint64) int {
	for i, c := range b.Columns {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// checkName cleans a column name and refuses a duplicate of another column
// (not the one with id self).
func (b Board) checkName(name string, self uint64) (string, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > columnNameMax {
		return "", ErrInvalidColumnName
	}
	for _, c := range b.Columns {
		if strings.EqualFold(c.Name, name) && (self == 0 || c.ID != self) {
			return "", ErrDuplicateColumn
		}
	}
	return name, nil
}
