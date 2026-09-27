// Package boards wires the boards context: its routes, the functions it
// publishes, and the dev seeder's board. It takes membership answers from
// guilds.Memberships and display names from identity, and nothing else from
// either.
package boards

import (
	"context"
	"time"

	"github.com/goravel/framework/contracts/route"

	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
	boardshttp "github.com/jevido/the-bakery/services/api/contexts/boards/http"
	"github.com/jevido/the-bakery/services/api/contexts/boards/infra"
	"github.com/jevido/the-bakery/services/api/contexts/guilds"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

// memberNames adapts identity's display names to boards' MemberNames.
type memberNames struct{}

func (memberNames) DisplayNames(ctx context.Context, ids []uint64) (map[uint64]string, error) {
	return identity.DisplayNames(ctx, ids)
}

var service = app.NewService(
	guilds.NewMemberships(), infra.Boards{}, infra.Tasks{}, infra.Comments{}, infra.ActivityLog{}, infra.PresenceLog{}, infra.WorkTypes{}, infra.Runs{}, memberNames{},
	app.NewDispatcher(infra.LogEvents{}.Handle, infra.ActivityProjector{}.Handle, infra.BoardEventPublisher{}.Handle),
)

// hub hands board events from Postgres to this process's open streams.
var hub = infra.NewBoardHub()

// Routes registers the board and task routes, all behind
// identity.RequireMember.
func Routes(r route.Router) {
	c := boardshttp.NewController(service, identity.MemberID)
	r.Middleware(identity.RequireMember).Group(func(r route.Router) {
		r.Get("/api/guilds/{guild}/boards", c.ListBoards)
		r.Post("/api/guilds/{guild}/boards", c.CreateBoard)
		r.Get("/api/boards/{board}", c.GetBoard)
		r.Post("/api/boards/{board}/columns", c.AddColumn)
		r.Get("/api/guilds/{guild}/work-types", c.ListWorkTypes)
		r.Post("/api/guilds/{guild}/work-types", c.AddWorkType)
		r.Patch("/api/guilds/{guild}/work-types/{key}", c.UpdateWorkType)
		r.Delete("/api/guilds/{guild}/work-types/{key}", c.DeleteWorkType)
		r.Patch("/api/columns/{column}", c.RenameColumn)
		r.Post("/api/columns/{column}/move", c.MoveColumn)
		r.Delete("/api/columns/{column}", c.DeleteColumn)
		r.Post("/api/boards/{board}/tasks", c.CreateTask)
		r.Get("/api/tasks/{task}", c.GetTask)
		r.Patch("/api/tasks/{task}", c.UpdateTask)
		r.Post("/api/tasks/{task}/subtasks", c.AddSubtask)
		r.Post("/api/tasks/{task}/expand", c.ExpandTask)
		r.Post("/api/tasks/{task}/move", c.MoveTask)
		r.Delete("/api/tasks/{task}", c.DeleteTask)
		r.Get("/api/tasks/{task}/comments", c.ListComments)
		r.Post("/api/tasks/{task}/comments", c.CommentOnTask)
		r.Patch("/api/comments/{comment}", c.EditComment)
		r.Delete("/api/comments/{comment}", c.DeleteComment)
		r.Get("/api/tasks/{task}/activity", c.ListActivity)
		r.Get("/api/tasks/{task}/runs", c.ListRuns)
		r.Post("/api/tasks/{task}/runs", c.StartRun)
		r.Patch("/api/runs/{run}", c.FinishRun)
	})
}

// StreamRoutes registers the board event stream. It must not sit behind
// the request timeout, which buffers the whole response.
func StreamRoutes(r route.Router) {
	c := boardshttp.NewEventsController(service, hub, identity.MemberID)
	r.Middleware(identity.RequireMember).Get("/api/boards/{board}/events", c.Stream)
}

// WatchBoard returns the board's events as JSON board events, for a member
// of its guild, and a function that stops watching. The channel closes if
// the events may have been interrupted; watch again and refetch the board.
func WatchBoard(ctx context.Context, boardID, memberID uint64) (<-chan []byte, func(), error) {
	if err := service.WatchBoard(ctx, boardID, memberID); err != nil {
		return nil, nil, err
	}
	events, stop := hub.Subscribe(boardID)
	return events, stop, nil
}

// CloseStreams ends every open board event stream in this process and
// refuses new ones, for when the process is about to stop.
func CloseStreams() { hub.Close() }

// SeedTask is a task for SeedBoard.
type SeedTask struct {
	Title    string
	Column   string
	WorkType string
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
		t, err := service.CreateTask(ctx, b.ID, memberID, st.Title, "", st.WorkType)
		if err != nil {
			return err
		}
		if st.Column != "" && st.Column != b.First().Name {
			if _, err := service.MoveTask(ctx, t.ID, memberID, app.ColumnRef{Name: st.Column}, nil, nil); err != nil {
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

// Task is a task or a subtask. A subtask has a ParentID and no column
// (ColumnID 0). The subtask counts are filled in where the caller asked for
// them (GetBoard, GetTask).
type Task struct {
	ID            uint64
	BoardID       uint64
	ParentID      *uint64
	ColumnID      uint64
	Title         string
	Description   string
	WorkType      string
	Done          bool
	SubtasksTotal int
	SubtasksDone  int
}

// Column is one of a board's columns with its tasks in order.
type Column struct {
	ID    uint64
	Name  string
	Tasks []Task
}

// ColumnRef names a board's column by id or, when ID is 0, by name
// (ignoring case).
type ColumnRef struct {
	ID   uint64
	Name string
}

func (r ColumnRef) app() app.ColumnRef { return app.ColumnRef{ID: r.ID, Name: r.Name} }

// Given reports whether the ref names a column at all.
func (r ColumnRef) Given() bool { return r.ID != 0 || r.Name != "" }

type BoardView struct {
	Board   Board
	Columns []Column
}

func boardOf(b domain.Board) Board { return Board{ID: b.ID, GuildID: b.GuildID, Name: b.Name} }

func taskOf(t domain.Task) Task {
	return Task{ID: t.ID, BoardID: t.BoardID, ParentID: t.ParentID, ColumnID: t.ColumnID, Title: t.Title, Description: t.Description, WorkType: t.WorkType, Done: t.Done}
}

func tasksOf(ts []domain.Task) []Task {
	out := make([]Task, len(ts))
	for i, t := range ts {
		out[i] = taskOf(t)
	}
	return out
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

// GetBoard returns a board with its columns in order, each with its tasks
// in order.
func GetBoard(ctx context.Context, boardID, memberID uint64) (BoardView, error) {
	b, tasks, counts, err := service.GetBoard(ctx, boardID, memberID)
	if err != nil {
		return BoardView{}, err
	}
	byColumn := map[uint64][]Task{}
	for _, t := range tasks {
		out := taskOf(t)
		out.SubtasksTotal, out.SubtasksDone = counts[t.ID].Total, counts[t.ID].Done
		byColumn[t.ColumnID] = append(byColumn[t.ColumnID], out)
	}
	view := BoardView{Board: boardOf(b)}
	for _, c := range b.Columns {
		view.Columns = append(view.Columns, Column{ID: c.ID, Name: c.Name, Tasks: append([]Task{}, byColumn[c.ID]...)})
	}
	return view, nil
}

// CreateBoard makes a board in a guild the member is in.
func CreateBoard(ctx context.Context, guildID, memberID uint64, name string) (Board, error) {
	b, err := service.CreateBoard(ctx, guildID, memberID, name)
	return boardOf(b), err
}

// CreateTask adds a task at the bottom of a column (the board's first when
// column is not given).
func CreateTask(ctx context.Context, boardID, memberID uint64, title, description, workType string, column ColumnRef) (Task, error) {
	if column.Given() {
		// Refuse a bad column before anything is created.
		if _, err := service.ResolveColumn(ctx, boardID, memberID, column.app()); err != nil {
			return Task{}, err
		}
	}
	t, err := service.CreateTask(ctx, boardID, memberID, title, description, workType)
	if err != nil {
		return Task{}, err
	}
	if column.Given() {
		if t, err = service.MoveTask(ctx, t.ID, memberID, column.app(), nil, nil); err != nil {
			return Task{}, err
		}
	}
	return taskOf(t), nil
}

// GetTask returns a task with its subtasks in order.
func GetTask(ctx context.Context, taskID, memberID uint64) (Task, []Task, error) {
	t, subtasks, err := service.GetTask(ctx, taskID, memberID)
	if err != nil {
		return Task{}, nil, err
	}
	out := taskOf(t)
	for _, st := range subtasks {
		out.SubtasksTotal++
		if st.Done {
			out.SubtasksDone++
		}
	}
	return out, tasksOf(subtasks), nil
}

// ExpandTask adds 1 to 50 subtasks at the end of a task's subtasks.
func ExpandTask(ctx context.Context, taskID, memberID uint64, titles []string) ([]Task, error) {
	added, err := service.ExpandTask(ctx, taskID, memberID, titles)
	return tasksOf(added), err
}

// TaskChanges are the fields UpdateTask changes; nil leaves one as it is,
// and an empty WorkType clears it.
type TaskChanges struct {
	Title       *string
	Description *string
	WorkType    *string
	Done        *bool
}

// UpdateTask changes a task's title, description or work type, and ticks a
// subtask off or on.
func UpdateTask(ctx context.Context, taskID, memberID uint64, c TaskChanges) (Task, error) {
	t, err := service.UpdateTask(ctx, taskID, memberID, app.TaskChanges(c))
	return taskOf(t), err
}

// WorkType is one of a guild's work types.
type WorkType struct {
	Key  string
	Name string
}

// ListWorkTypes returns the guild's work types in order.
func ListWorkTypes(ctx context.Context, guildID, memberID uint64) ([]WorkType, error) {
	wts, err := service.ListWorkTypes(ctx, guildID, memberID)
	if err != nil {
		return nil, err
	}
	out := make([]WorkType, len(wts))
	for i, wt := range wts {
		out[i] = WorkType{Key: wt.Key, Name: wt.Name}
	}
	return out, nil
}

// MoveTask puts a task in column right after afterID and/or right before
// beforeID; with neither, at the bottom.
func MoveTask(ctx context.Context, taskID, memberID uint64, column ColumnRef, afterID, beforeID *uint64) (Task, error) {
	t, err := service.MoveTask(ctx, taskID, memberID, column.app(), afterID, beforeID)
	return taskOf(t), err
}

// Comment is a comment on a task with its author's display name.
type Comment struct {
	ID         uint64
	TaskID     uint64
	AuthorID   uint64
	AuthorName string
	Body       string
	CreatedAt  time.Time
	EditedAt   *time.Time
}

func commentOf(c app.AuthoredComment) Comment {
	return Comment{ID: c.ID, TaskID: c.TaskID, AuthorID: c.AuthorID, AuthorName: c.AuthorName, Body: c.Body, CreatedAt: c.CreatedAt, EditedAt: c.EditedAt}
}

// ListComments returns a task's comments, oldest first.
func ListComments(ctx context.Context, taskID, memberID uint64) ([]Comment, error) {
	cs, err := service.ListComments(ctx, taskID, memberID)
	if err != nil {
		return nil, err
	}
	out := make([]Comment, len(cs))
	for i, c := range cs {
		out[i] = commentOf(c)
	}
	return out, nil
}

// CommentOnTask writes a comment on a task as the member.
func CommentOnTask(ctx context.Context, taskID, memberID uint64, body string) (Comment, error) {
	c, err := service.CommentOnTask(ctx, taskID, memberID, body)
	return commentOf(c), err
}

// DeleteTask removes a task.
func DeleteTask(ctx context.Context, taskID, memberID uint64) error {
	return service.DeleteTask(ctx, taskID, memberID)
}

// Run is one run of an agent on a task, as the MCP server reads it.
type Run struct {
	ID         uint64
	AgentName  string
	MemberName string
	Branch     string
	Status     string
	StartedAt  time.Time
	EndedAt    *time.Time
	CostUSD    float64
	Summary    string
}

// ListRuns returns a task's latest runs, newest first; limit 0 means all.
func ListRuns(ctx context.Context, taskID, memberID uint64, limit int) ([]Run, error) {
	rs, err := service.ListRuns(ctx, taskID, memberID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]Run, len(rs))
	for i, r := range rs {
		out[i] = Run{
			ID: r.ID, AgentName: r.AgentName, MemberName: r.MemberName, Branch: r.Branch, Status: string(r.Status),
			StartedAt: r.StartedAt, EndedAt: r.EndedAt, CostUSD: r.CostUSD, Summary: r.Summary,
		}
	}
	return out, nil
}
