package mcp

import (
	"context"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jevido/the-bakery/services/api/contexts/boards"
	"github.com/jevido/the-bakery/services/api/contexts/guilds"
)

type guildOut struct {
	ID       uint64 `json:"id" jsonschema:"the guild id, for list_boards and create_board"`
	Name     string `json:"name"`
	Archived bool   `json:"archived" jsonschema:"archived guilds are read-only"`
}

type boardOut struct {
	ID      uint64 `json:"id" jsonschema:"the board id, for get_board and create_task"`
	GuildID uint64 `json:"guild_id"`
	Name    string `json:"name"`
}

// taskOut is a task or a subtask. A subtask has parent_id and done, and no
// column; a task on the board has its subtask counts.
type taskOut struct {
	ID            uint64  `json:"id" jsonschema:"the task id, for get_task, update_task, move_task, expand_task and delete_task"`
	BoardID       uint64  `json:"board_id"`
	ParentID      *uint64 `json:"parent_id,omitempty" jsonschema:"set on a subtask: the task it belongs to"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	ColumnID      uint64  `json:"column_id,omitempty" jsonschema:"the column it stands in (see get_board); a subtask has none"`
	WorkType      string  `json:"work_type,omitempty" jsonschema:"the kind of work it needs, a key from list_work_types"`
	Done          *bool   `json:"done,omitempty" jsonschema:"set on a subtask: whether it is ticked off"`
	SubtasksTotal *int    `json:"subtasks_total,omitempty" jsonschema:"how many subtasks this task has"`
	SubtasksDone  *int    `json:"subtasks_done,omitempty" jsonschema:"how many of them are done"`
}

type columnOut struct {
	ID    uint64    `json:"id" jsonschema:"the column id, for move_task and create_task"`
	Name  string    `json:"name"`
	Tasks []taskOut `json:"tasks" jsonschema:"top to bottom"`
}

func taskOutOf(t boards.Task) taskOut {
	out := taskOut{ID: t.ID, BoardID: t.BoardID, ParentID: t.ParentID, Title: t.Title, Description: t.Description, ColumnID: t.ColumnID, WorkType: t.WorkType}
	if t.ParentID != nil {
		out.Done = &t.Done
	} else {
		out.SubtasksTotal, out.SubtasksDone = &t.SubtasksTotal, &t.SubtasksDone
	}
	return out
}

func tasksOutOf(ts []boards.Task) []taskOut {
	out := make([]taskOut, len(ts))
	for i, t := range ts {
		out[i] = taskOutOf(t)
	}
	return out
}

type commentOut struct {
	ID         uint64     `json:"id"`
	AuthorName string     `json:"author_name"`
	Body       string     `json:"body" jsonschema:"markdown"`
	CreatedAt  time.Time  `json:"created_at"`
	EditedAt   *time.Time `json:"edited_at,omitempty"`
}

func commentOutOf(c boards.Comment) commentOut {
	return commentOut{ID: c.ID, AuthorName: c.AuthorName, Body: c.Body, CreatedAt: c.CreatedAt, EditedAt: c.EditedAt}
}

type listWorkTypesIn struct {
	GuildID uint64 `json:"guild_id" jsonschema:"a guild id from list_guilds"`
}

type workTypeOut struct {
	Key  string `json:"key" jsonschema:"use this in create_task and update_task"`
	Name string `json:"name"`
}

type listWorkTypesOut struct {
	WorkTypes []workTypeOut `json:"work_types" jsonschema:"in the guild's order"`
}

type getTaskIn struct {
	TaskID uint64 `json:"task_id" jsonschema:"a task id from get_board"`
}

// recentComments is how many comments get_task returns: the last ones.
const recentComments = 20

type getTaskOut struct {
	Task          taskOut      `json:"task"`
	Subtasks      []taskOut    `json:"subtasks" jsonschema:"in order"`
	Comments      []commentOut `json:"comments" jsonschema:"the last 20 comments, oldest first"`
	CommentsTotal int          `json:"comments_total"`
}

type listGuildsIn struct {
	IncludeArchived bool `json:"include_archived,omitempty" jsonschema:"also list archived guilds"`
}

type listGuildsOut struct {
	Guilds []guildOut `json:"guilds"`
}

type listBoardsIn struct {
	GuildID uint64 `json:"guild_id" jsonschema:"a guild id from list_guilds"`
}

type listBoardsOut struct {
	Boards []boardOut `json:"boards"`
}

type getBoardIn struct {
	BoardID uint64 `json:"board_id" jsonschema:"a board id from list_boards"`
}

type getBoardOut struct {
	Board   boardOut    `json:"board"`
	Columns []columnOut `json:"columns" jsonschema:"the board's columns, left to right"`
}

var readOnly = &sdk.ToolAnnotations{ReadOnlyHint: true}

func addReadTools(s *sdk.Server) {
	sdk.AddTool(s, &sdk.Tool{
		Name:        "list_guilds",
		Title:       "List guilds",
		Description: "Lists the guilds you are a member of: id, name, and whether it is archived (read-only). Pass a guild id to list_boards.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in listGuildsIn) (*sdk.CallToolResult, listGuildsOut, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, listGuildsOut{}, err
		}
		gs, err := guilds.ListGuildsOf(ctx, me, in.IncludeArchived)
		if err != nil {
			r, _ := failed(err)
			return r, listGuildsOut{}, nil
		}
		out := listGuildsOut{Guilds: []guildOut{}}
		for _, g := range gs {
			out.Guilds = append(out.Guilds, guildOut{ID: g.ID, Name: g.Name, Archived: g.Archived})
		}
		return nil, out, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name:        "list_boards",
		Title:       "List boards",
		Description: "Lists the boards of one guild: id and name. Pass a board id to get_board to see its tasks.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in listBoardsIn) (*sdk.CallToolResult, listBoardsOut, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, listBoardsOut{}, err
		}
		bs, err := boards.ListBoards(ctx, in.GuildID, me)
		if err != nil {
			r, _ := failed(err)
			return r, listBoardsOut{}, nil
		}
		out := listBoardsOut{Boards: []boardOut{}}
		for _, b := range bs {
			out.Boards = append(out.Boards, boardOut{ID: b.ID, GuildID: b.GuildID, Name: b.Name})
		}
		return nil, out, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name:  "get_board",
		Title: "Get a board",
		Description: "Returns one board with its columns left to right (id and name; boards start with Backlog, To do, " +
			"Doing and Done, and members add their own), each with its tasks " +
			"from top to bottom: id, title, description, and how many subtasks each has and how many are done. Subtasks " +
			"are not listed here; get_task shows them. Use the task ids with get_task, update_task, move_task, " +
			"expand_task and delete_task.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in getBoardIn) (*sdk.CallToolResult, getBoardOut, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, getBoardOut{}, err
		}
		v, err := boards.GetBoard(ctx, in.BoardID, me)
		if err != nil {
			r, _ := failed(err)
			return r, getBoardOut{}, nil
		}
		out := getBoardOut{Board: boardOut{ID: v.Board.ID, GuildID: v.Board.GuildID, Name: v.Board.Name}}
		for _, c := range v.Columns {
			col := columnOut{ID: c.ID, Name: c.Name, Tasks: []taskOut{}}
			for _, t := range c.Tasks {
				col.Tasks = append(col.Tasks, taskOutOf(t))
			}
			out.Columns = append(out.Columns, col)
		}
		return nil, out, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name:        "list_work_types",
		Title:       "List work types",
		Description: "Lists the kinds of work a guild's tasks can need (Coding, Research, ...): key and name. A task's work_type is one of these keys.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in listWorkTypesIn) (*sdk.CallToolResult, listWorkTypesOut, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, listWorkTypesOut{}, err
		}
		wts, err := boards.ListWorkTypes(ctx, in.GuildID, me)
		if err != nil {
			r, _ := failed(err)
			return r, listWorkTypesOut{}, nil
		}
		out := listWorkTypesOut{WorkTypes: []workTypeOut{}}
		for _, wt := range wts {
			out.WorkTypes = append(out.WorkTypes, workTypeOut{Key: wt.Key, Name: wt.Name})
		}
		return nil, out, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name:  "get_task",
		Title: "Get a task",
		Description: "Returns one task in full: its description, its subtasks in order (with whether each is done), " +
			"and its last 20 comments, oldest first. Read this before working on a task.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in getTaskIn) (*sdk.CallToolResult, getTaskOut, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, getTaskOut{}, err
		}
		t, subtasks, err := boards.GetTask(ctx, in.TaskID, me)
		if err != nil {
			r, _ := failed(err)
			return r, getTaskOut{}, nil
		}
		cs, err := boards.ListComments(ctx, in.TaskID, me)
		if err != nil {
			r, _ := failed(err)
			return r, getTaskOut{}, nil
		}
		out := getTaskOut{Task: taskOutOf(t), Subtasks: tasksOutOf(subtasks), Comments: []commentOut{}, CommentsTotal: len(cs)}
		for _, c := range cs[max(0, len(cs)-recentComments):] {
			out.Comments = append(out.Comments, commentOutOf(c))
		}
		return nil, out, nil
	})
}
