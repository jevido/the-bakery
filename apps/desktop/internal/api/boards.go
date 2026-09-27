package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Guild struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type Board struct {
	ID      uint64 `json:"id"`
	GuildID uint64 `json:"guild_id"`
	Name    string `json:"name"`
}

// Task is a task or a subtask. A subtask has a ParentID and no column
// (ColumnID nil).
// SubtasksTotal and SubtasksDone are filled in on a board's tasks and in
// GetTask.
type Task struct {
	ID            uint64  `json:"id"`
	BoardID       uint64  `json:"board_id"`
	ParentID      *uint64 `json:"parent_id"`
	ColumnID      *uint64 `json:"column_id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Position      string  `json:"position"`
	Done          bool    `json:"done"`
	WorkType      *string `json:"work_type"`
	SubtasksTotal int     `json:"subtasks_total"`
	SubtasksDone  int     `json:"subtasks_done"`
}

// TaskDetail is one task with its subtasks in order.
type TaskDetail struct {
	Task     Task   `json:"task"`
	Subtasks []Task `json:"subtasks"`
}

// Column is one of a board's columns with its tasks in order.
type Column struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Tasks []Task `json:"tasks"`
}

// BoardView is a board with every column, in board order.
type BoardView struct {
	Board   Board    `json:"board"`
	Columns []Column `json:"columns"`
}

func (c *Client) ListGuilds(ctx context.Context, token string) ([]Guild, error) {
	var res struct {
		Guilds []Guild `json:"guilds"`
	}
	err := c.do(ctx, http.MethodGet, "/api/guilds", token, nil, &res)
	return res.Guilds, err
}

func (c *Client) ListBoards(ctx context.Context, token string, guildID uint64) ([]Board, error) {
	var res struct {
		Boards []Board `json:"boards"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/guilds/%d/boards", guildID), token, nil, &res)
	return res.Boards, err
}

func (c *Client) CreateBoard(ctx context.Context, token string, guildID uint64, name string) (Board, error) {
	var res struct {
		Board Board `json:"board"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/guilds/%d/boards", guildID), token, map[string]string{"name": name}, &res)
	return res.Board, err
}

func (c *Client) GetBoard(ctx context.Context, token string, boardID uint64) (BoardView, error) {
	var res BoardView
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/boards/%d", boardID), token, nil, &res)
	return res, err
}

func (c *Client) CreateTask(ctx context.Context, token string, boardID uint64, title string) (Task, error) {
	var res struct {
		Task Task `json:"task"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/boards/%d/tasks", boardID), token, map[string]string{"title": title}, &res)
	return res.Task, err
}

// UpdateTask changes the title and/or description; nil leaves one as it is.
func (c *Client) UpdateTask(ctx context.Context, token string, taskID uint64, title, description *string) (Task, error) {
	body := map[string]*string{}
	if title != nil {
		body["title"] = title
	}
	if description != nil {
		body["description"] = description
	}
	var res struct {
		Task Task `json:"task"`
	}
	err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/tasks/%d", taskID), token, body, &res)
	return res.Task, err
}

// MoveTask puts a task in the column right after afterID and/or right
// before beforeID (nil for either end).
func (c *Client) MoveTask(ctx context.Context, token string, taskID, columnID uint64, afterID, beforeID *uint64) (Task, error) {
	body := map[string]any{"column_id": columnID, "after_id": afterID, "before_id": beforeID}
	var res struct {
		Task Task `json:"task"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/tasks/%d/move", taskID), token, body, &res)
	return res.Task, err
}

func (c *Client) DeleteTask(ctx context.Context, token string, taskID uint64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/tasks/%d", taskID), token, nil, nil)
}

func (c *Client) GetTask(ctx context.Context, token string, taskID uint64) (TaskDetail, error) {
	var res TaskDetail
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/tasks/%d", taskID), token, nil, &res)
	return res, err
}

func (c *Client) AddSubtask(ctx context.Context, token string, taskID uint64, title string) (Task, error) {
	var res struct {
		Subtask Task `json:"subtask"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/tasks/%d/subtasks", taskID), token, map[string]string{"title": title}, &res)
	return res.Subtask, err
}

// SetSubtaskDone ticks a subtask off or opens it again.
func (c *Client) SetSubtaskDone(ctx context.Context, token string, taskID uint64, done bool) (Task, error) {
	var res struct {
		Task Task `json:"task"`
	}
	err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/tasks/%d", taskID), token, map[string]bool{"done": done}, &res)
	return res.Task, err
}

// MoveSubtask puts a subtask right after afterID and/or right before
// beforeID among its siblings (nil for either end).
func (c *Client) MoveSubtask(ctx context.Context, token string, taskID uint64, afterID, beforeID *uint64) (Task, error) {
	body := map[string]any{"after_id": afterID, "before_id": beforeID}
	var res struct {
		Task Task `json:"task"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/tasks/%d/move", taskID), token, body, &res)
	return res.Task, err
}

// Comment is a comment on a task, with its author's display name.
type Comment struct {
	ID         uint64     `json:"id"`
	TaskID     uint64     `json:"task_id"`
	AuthorID   uint64     `json:"author_id"`
	AuthorName string     `json:"author_name"`
	Body       string     `json:"body"`
	CreatedAt  time.Time  `json:"created_at"`
	EditedAt   *time.Time `json:"edited_at"`
}

// Activity is one entry of a task's history. Data depends on Kind.
type Activity struct {
	ID        uint64         `json:"id"`
	Kind      string         `json:"kind"`
	ActorID   uint64         `json:"actor_id"`
	ActorName string         `json:"actor_name"`
	At        time.Time      `json:"at"`
	Data      map[string]any `json:"data"`
}

func (c *Client) ListComments(ctx context.Context, token string, taskID uint64) ([]Comment, error) {
	var res struct {
		Comments []Comment `json:"comments"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/tasks/%d/comments", taskID), token, nil, &res)
	return res.Comments, err
}

func (c *Client) AddComment(ctx context.Context, token string, taskID uint64, body string) (Comment, error) {
	var res struct {
		Comment Comment `json:"comment"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/tasks/%d/comments", taskID), token, map[string]string{"body": body}, &res)
	return res.Comment, err
}

func (c *Client) EditComment(ctx context.Context, token string, commentID uint64, body string) (Comment, error) {
	var res struct {
		Comment Comment `json:"comment"`
	}
	err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/comments/%d", commentID), token, map[string]string{"body": body}, &res)
	return res.Comment, err
}

func (c *Client) DeleteComment(ctx context.Context, token string, commentID uint64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/comments/%d", commentID), token, nil, nil)
}

// ListActivity returns up to limit entries, newest first, older than the
// entry before (0: from the newest).
func (c *Client) ListActivity(ctx context.Context, token string, taskID, before uint64, limit int) ([]Activity, error) {
	var res struct {
		Activity []Activity `json:"activity"`
	}
	path := fmt.Sprintf("/api/tasks/%d/activity?limit=%d", taskID, limit)
	if before > 0 {
		path += fmt.Sprintf("&before=%d", before)
	}
	err := c.do(ctx, http.MethodGet, path, token, nil, &res)
	return res.Activity, err
}

// BoardColumn is a column without its tasks, as the column endpoints return it.
type BoardColumn struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

func (c *Client) CreateColumn(ctx context.Context, token string, boardID uint64, name string) (BoardColumn, error) {
	var res struct {
		Column BoardColumn `json:"column"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/boards/%d/columns", boardID), token, map[string]string{"name": name}, &res)
	return res.Column, err
}

func (c *Client) RenameColumn(ctx context.Context, token string, columnID uint64, name string) (BoardColumn, error) {
	var res struct {
		Column BoardColumn `json:"column"`
	}
	err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/columns/%d", columnID), token, map[string]string{"name": name}, &res)
	return res.Column, err
}

// MoveColumn puts a column right after afterID and/or right before
// beforeID (nil for either end).
func (c *Client) MoveColumn(ctx context.Context, token string, columnID uint64, afterID, beforeID *uint64) (BoardColumn, error) {
	var res struct {
		Column BoardColumn `json:"column"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/columns/%d/move", columnID), token, map[string]any{"after_id": afterID, "before_id": beforeID}, &res)
	return res.Column, err
}

func (c *Client) DeleteColumn(ctx context.Context, token string, columnID uint64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/columns/%d", columnID), token, nil, nil)
}

// WorkType is one of a guild's kinds of work.
type WorkType struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func (c *Client) ListWorkTypes(ctx context.Context, token string, guildID uint64) ([]WorkType, error) {
	var res struct {
		WorkTypes []WorkType `json:"work_types"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/guilds/%d/work-types", guildID), token, nil, &res)
	return res.WorkTypes, err
}

func (c *Client) AddWorkType(ctx context.Context, token string, guildID uint64, key, name string) (WorkType, error) {
	var res struct {
		WorkType WorkType `json:"work_type"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/guilds/%d/work-types", guildID), token, map[string]string{"key": key, "name": name}, &res)
	return res.WorkType, err
}

// UpdateWorkType renames a work type and/or moves it to position (nil
// leaves either as it is).
func (c *Client) UpdateWorkType(ctx context.Context, token string, guildID uint64, key string, name *string, position *int) (WorkType, error) {
	body := map[string]any{}
	if name != nil {
		body["name"] = *name
	}
	if position != nil {
		body["position"] = *position
	}
	var res struct {
		WorkType WorkType `json:"work_type"`
	}
	err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/guilds/%d/work-types/%s", guildID, url.PathEscape(key)), token, body, &res)
	return res.WorkType, err
}

func (c *Client) DeleteWorkType(ctx context.Context, token string, guildID uint64, key string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/guilds/%d/work-types/%s", guildID, url.PathEscape(key)), token, nil, nil)
}

// SetTaskWorkType gives a task one of the guild's work types ("" clears it).
func (c *Client) SetTaskWorkType(ctx context.Context, token string, taskID uint64, key string) (Task, error) {
	var res struct {
		Task Task `json:"task"`
	}
	err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/tasks/%d", taskID), token, map[string]string{"work_type": key}, &res)
	return res.Task, err
}
