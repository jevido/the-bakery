package mcp

import (
	"context"

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

type taskOut struct {
	ID          uint64 `json:"id" jsonschema:"the task id, for update_task, move_task and delete_task"`
	BoardID     uint64 `json:"board_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Column      string `json:"column" jsonschema:"backlog, todo, doing or done"`
}

type columnOut struct {
	Column string    `json:"column"`
	Tasks  []taskOut `json:"tasks" jsonschema:"top to bottom"`
}

func taskOutOf(t boards.Task) taskOut {
	return taskOut{ID: t.ID, BoardID: t.BoardID, Title: t.Title, Description: t.Description, Column: t.Column}
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
	Columns []columnOut `json:"columns" jsonschema:"backlog, todo, doing, done, in that order"`
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
		Description: "Returns one board with its four columns in order (backlog, todo, doing, done), each with its tasks " +
			"from top to bottom: id, title, description. Use the task ids with update_task, move_task and delete_task.",
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
			col := columnOut{Column: c.Column, Tasks: []taskOut{}}
			for _, t := range c.Tasks {
				col.Tasks = append(col.Tasks, taskOutOf(t))
			}
			out.Columns = append(out.Columns, col)
		}
		return nil, out, nil
	})
}
