// Package domain holds the boards model: boards, tasks, columns and
// positions. It imports nothing outside the standard library.
package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const boardNameMax = 60

var (
	ErrInvalidBoardName = errors.New("board name must be 1 to 60 characters")
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
