// Package boards wires the boards context: its routes, and the dev seeder's
// board. It takes membership answers from guilds.Memberships and nothing
// else from guilds.
package boards

import (
	"context"

	"github.com/goravel/framework/contracts/route"

	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
	boardshttp "github.com/jevido/the-bakery/services/api/contexts/boards/http"
	"github.com/jevido/the-bakery/services/api/contexts/boards/infra"
	"github.com/jevido/the-bakery/services/api/contexts/guilds"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

var service = app.NewService(guilds.NewMemberships(), infra.Boards{}, infra.Tasks{}, infra.LogEvents{})

// Routes registers the board and task routes, all behind
// identity.RequireMember.
func Routes(r route.Router) {
	c := boardshttp.NewController(service, identity.MemberID)
	r.Middleware(identity.RequireMember).Group(func(r route.Router) {
		r.Get("/api/guilds/{guild}/boards", c.ListBoards)
		r.Post("/api/guilds/{guild}/boards", c.CreateBoard)
		r.Get("/api/boards/{board}", c.GetBoard)
		r.Post("/api/boards/{board}/tasks", c.CreateTask)
		r.Patch("/api/tasks/{task}", c.UpdateTask)
		r.Post("/api/tasks/{task}/move", c.MoveTask)
		r.Delete("/api/tasks/{task}", c.DeleteTask)
	})
}

// SeedTask is a task for SeedBoard.
type SeedTask struct {
	Title  string
	Column string
}

// SeedBoard makes sure the guild has a board with this name; a new board
// gets the given tasks, an existing one is left as it is. For the dev
// seeder only, which acts as memberID.
func SeedBoard(ctx context.Context, guildID, memberID uint64, name string, tasks []SeedTask) error {
	if _, found, err := (infra.Boards{}).ByName(ctx, guildID, name); err != nil || found {
		return err
	}
	b, err := service.CreateBoard(ctx, guildID, memberID, name)
	if err != nil {
		return err
	}
	for _, st := range tasks {
		t, err := service.CreateTask(ctx, b.ID, memberID, st.Title, "")
		if err != nil {
			return err
		}
		if st.Column != string(domain.Backlog) {
			if _, err := service.MoveTask(ctx, t.ID, memberID, st.Column, nil, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// Board, Task and BoardView are what other modules (the MCP server) see.
type Board struct {
	ID      uint64
	GuildID uint64
	Name    string
}

type Task struct {
	ID          uint64
	BoardID     uint64
	Title       string
	Description string
	Column      string
}

type Column struct {
	Column string
	Tasks  []Task
}

type BoardView struct {
	Board   Board
	Columns []Column
}

func boardOf(b domain.Board) Board { return Board{ID: b.ID, GuildID: b.GuildID, Name: b.Name} }

func taskOf(t domain.Task) Task {
	return Task{ID: t.ID, BoardID: t.BoardID, Title: t.Title, Description: t.Description, Column: string(t.Column)}
}

// ListBoards returns a guild's boards; the member must be in the guild.
func ListBoards(ctx context.Context, guildID, memberID uint64) ([]Board, error) {
	bs, err := service.ListBoards(ctx, guildID, memberID)
	if err != nil {
		return nil, err
	}
	out := make([]Board, len(bs))
	for i, b := range bs {
		out[i] = boardOf(b)
	}
	return out, nil
}

// GetBoard returns a board with all four columns in order, each with its
// tasks in order.
func GetBoard(ctx context.Context, boardID, memberID uint64) (BoardView, error) {
	b, tasks, err := service.GetBoard(ctx, boardID, memberID)
	if err != nil {
		return BoardView{}, err
	}
	byColumn := map[domain.Column][]Task{}
	for _, t := range tasks {
		byColumn[t.Column] = append(byColumn[t.Column], taskOf(t))
	}
	view := BoardView{Board: boardOf(b)}
	for _, c := range domain.Columns {
		view.Columns = append(view.Columns, Column{Column: string(c), Tasks: append([]Task{}, byColumn[c]...)})
	}
	return view, nil
}

// CreateBoard makes a board in a guild the member is in.
func CreateBoard(ctx context.Context, guildID, memberID uint64, name string) (Board, error) {
	b, err := service.CreateBoard(ctx, guildID, memberID, name)
	return boardOf(b), err
}

// CreateTask adds a task at the bottom of column ("" means backlog).
func CreateTask(ctx context.Context, boardID, memberID uint64, title, description, column string) (Task, error) {
	if column != "" && column != string(domain.Backlog) {
		// Refuse a bad column before anything is created.
		if _, err := domain.ParseColumn(column); err != nil {
			return Task{}, err
		}
	}
	t, err := service.CreateTask(ctx, boardID, memberID, title, description)
	if err != nil {
		return Task{}, err
	}
	if column != "" && column != string(domain.Backlog) {
		if t, err = service.MoveTask(ctx, t.ID, memberID, column, nil, nil); err != nil {
			return Task{}, err
		}
	}
	return taskOf(t), nil
}

// UpdateTask changes a task's title and/or description (nil keeps it).
func UpdateTask(ctx context.Context, taskID, memberID uint64, title, description *string) (Task, error) {
	t, err := service.UpdateTask(ctx, taskID, memberID, title, description)
	return taskOf(t), err
}

// MoveTask puts a task in column right after afterID and/or right before
// beforeID; with neither, at the bottom.
func MoveTask(ctx context.Context, taskID, memberID uint64, column string, afterID, beforeID *uint64) (Task, error) {
	t, err := service.MoveTask(ctx, taskID, memberID, column, afterID, beforeID)
	return taskOf(t), err
}

// DeleteTask removes a task.
func DeleteTask(ctx context.Context, taskID, memberID uint64) error {
	return service.DeleteTask(ctx, taskID, memberID)
}
