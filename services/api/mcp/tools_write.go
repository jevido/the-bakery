package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jevido/the-bakery/services/api/contexts/boards"
)

type createBoardIn struct {
	GuildID uint64 `json:"guild_id" jsonschema:"a guild id from list_guilds"`
	Name    string `json:"name" jsonschema:"1 to 60 characters"`
}

type boardResult struct {
	Board boardOut `json:"board"`
}

type createTaskIn struct {
	BoardID     uint64 `json:"board_id" jsonschema:"a board id from list_boards"`
	Title       string `json:"title" jsonschema:"1 to 200 characters"`
	Description string `json:"description,omitempty"`
	Column      string `json:"column,omitempty" jsonschema:"backlog (the default), todo, doing or done; the task goes to the bottom"`
}

type taskResult struct {
	Task taskOut `json:"task"`
}

type updateTaskIn struct {
	TaskID      uint64  `json:"task_id" jsonschema:"a task id from get_board"`
	Title       *string `json:"title,omitempty" jsonschema:"new title; leave out to keep it"`
	Description *string `json:"description,omitempty" jsonschema:"new description; leave out to keep it"`
}

type moveTaskIn struct {
	TaskID       uint64  `json:"task_id" jsonschema:"a task id from get_board"`
	Column       string  `json:"column" jsonschema:"backlog, todo, doing or done"`
	AfterTaskID  *uint64 `json:"after_task_id,omitempty" jsonschema:"put it right below this task in that column"`
	BeforeTaskID *uint64 `json:"before_task_id,omitempty" jsonschema:"put it right above this task in that column"`
}

type deleteTaskIn struct {
	TaskID uint64 `json:"task_id" jsonschema:"a task id from get_board"`
}

type deletedOut struct {
	Deleted uint64 `json:"deleted" jsonschema:"the id of the deleted task"`
}

func ptr[T any](v T) *T { return &v }

func addWriteTools(s *sdk.Server) {
	sdk.AddTool(s, &sdk.Tool{
		Name:        "create_board",
		Title:       "Create a board",
		Description: "Creates a board in a guild and returns it with its id.",
		Annotations: &sdk.ToolAnnotations{DestructiveHint: ptr(false), OpenWorldHint: ptr(false)},
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in createBoardIn) (*sdk.CallToolResult, boardResult, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, boardResult{}, err
		}
		b, err := boards.CreateBoard(ctx, in.GuildID, me, in.Name)
		if err != nil {
			r, _ := failed(err)
			return r, boardResult{}, nil
		}
		return nil, boardResult{Board: boardOut{ID: b.ID, GuildID: b.GuildID, Name: b.Name}}, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name:        "create_task",
		Title:       "Create a task",
		Description: "Adds a task to a board, at the bottom of a column (backlog unless you say otherwise), and returns it with its id.",
		Annotations: &sdk.ToolAnnotations{DestructiveHint: ptr(false), OpenWorldHint: ptr(false)},
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in createTaskIn) (*sdk.CallToolResult, taskResult, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, taskResult{}, err
		}
		t, err := boards.CreateTask(ctx, in.BoardID, me, in.Title, in.Description, in.Column)
		if err != nil {
			r, _ := failed(err)
			return r, taskResult{}, nil
		}
		return nil, taskResult{Task: taskOutOf(t)}, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name:        "update_task",
		Title:       "Update a task",
		Description: "Changes a task's title and/or description. Leave a field out to keep it. Returns the task.",
		Annotations: &sdk.ToolAnnotations{DestructiveHint: ptr(false), IdempotentHint: true, OpenWorldHint: ptr(false)},
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in updateTaskIn) (*sdk.CallToolResult, taskResult, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, taskResult{}, err
		}
		t, err := boards.UpdateTask(ctx, in.TaskID, me, in.Title, in.Description, nil)
		if err != nil {
			r, _ := failed(err)
			return r, taskResult{}, nil
		}
		return nil, taskResult{Task: taskOutOf(t)}, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name:  "move_task",
		Title: "Move a task",
		Description: "Moves a task to a column: right below after_task_id and/or right above before_task_id, or to the " +
			"bottom of the column when you give neither. Neighbours must be other tasks in that column. Returns the task.",
		Annotations: &sdk.ToolAnnotations{DestructiveHint: ptr(false), OpenWorldHint: ptr(false)},
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in moveTaskIn) (*sdk.CallToolResult, taskResult, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, taskResult{}, err
		}
		t, err := boards.MoveTask(ctx, in.TaskID, me, in.Column, in.AfterTaskID, in.BeforeTaskID)
		if err != nil {
			r, _ := failed(err)
			return r, taskResult{}, nil
		}
		return nil, taskResult{Task: taskOutOf(t)}, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name:        "delete_task",
		Title:       "Delete a task",
		Description: "Deletes a task for good. There is no undo; ask the person first.",
		Annotations: &sdk.ToolAnnotations{DestructiveHint: ptr(true), OpenWorldHint: ptr(false)},
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in deleteTaskIn) (*sdk.CallToolResult, deletedOut, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, deletedOut{}, err
		}
		if err := boards.DeleteTask(ctx, in.TaskID, me); err != nil {
			r, _ := failed(err)
			return r, deletedOut{}, nil
		}
		return nil, deletedOut{Deleted: in.TaskID}, nil
	})
}
