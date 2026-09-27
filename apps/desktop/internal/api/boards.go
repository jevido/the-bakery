package api

import (
	"context"
	"fmt"
	"net/http"
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

type Task struct {
	ID          uint64 `json:"id"`
	BoardID     uint64 `json:"board_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Column      string `json:"column"`
	Position    string `json:"position"`
}

// Column is one board column with its tasks in order.
type Column struct {
	Column string `json:"column"`
	Tasks  []Task `json:"tasks"`
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

// MoveTask puts a task in column right after afterID and/or right before
// beforeID (nil for either end).
func (c *Client) MoveTask(ctx context.Context, token string, taskID uint64, column string, afterID, beforeID *uint64) (Task, error) {
	body := map[string]any{"column": column, "after_id": afterID, "before_id": beforeID}
	var res struct {
		Task Task `json:"task"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/tasks/%d/move", taskID), token, body, &res)
	return res.Task, err
}

func (c *Client) DeleteTask(ctx context.Context, token string, taskID uint64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/tasks/%d", taskID), token, nil, nil)
}
