package main

import (
	"context"
	"errors"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/session"
)

// ErrSignedOut tells the frontend to show the login screen again.
var ErrSignedOut = errors.New("signed out: sign in again")

// BoardsService gives the frontend the member's guilds, boards and tasks.
// Every call uses the signed-in member's token, which never leaves Go.
type BoardsService struct {
	client  *api.Client
	session *session.Session
}

func NewBoardsService(client *api.Client, s *session.Session) *BoardsService {
	return &BoardsService{client: client, session: s}
}

// call runs f with the member's token. A token the API refuses ends the
// session.
func call[T any](sess *session.Session, f func(token string) (T, error)) (T, error) {
	token := sess.Token()
	if token == "" {
		var zero T
		return zero, ErrSignedOut
	}
	v, err := f(token)
	if errors.Is(err, api.ErrUnauthorized) {
		_ = sess.Logout(context.Background())
		return v, ErrSignedOut
	}
	return v, err
}

func (s *BoardsService) ListGuilds(ctx context.Context) ([]api.Guild, error) {
	return call(s.session, func(t string) ([]api.Guild, error) { return s.client.ListGuilds(ctx, t) })
}

func (s *BoardsService) ListBoards(ctx context.Context, guildID uint64) ([]api.Board, error) {
	return call(s.session, func(t string) ([]api.Board, error) { return s.client.ListBoards(ctx, t, guildID) })
}

func (s *BoardsService) CreateBoard(ctx context.Context, guildID uint64, name string) (api.Board, error) {
	return call(s.session, func(t string) (api.Board, error) { return s.client.CreateBoard(ctx, t, guildID, name) })
}

func (s *BoardsService) GetBoard(ctx context.Context, boardID uint64) (api.BoardView, error) {
	return call(s.session, func(t string) (api.BoardView, error) { return s.client.GetBoard(ctx, t, boardID) })
}

func (s *BoardsService) CreateTask(ctx context.Context, boardID uint64, title string) (api.Task, error) {
	return call(s.session, func(t string) (api.Task, error) { return s.client.CreateTask(ctx, t, boardID, title) })
}

func (s *BoardsService) UpdateTask(ctx context.Context, taskID uint64, title, description *string) (api.Task, error) {
	return call(s.session, func(t string) (api.Task, error) { return s.client.UpdateTask(ctx, t, taskID, title, description) })
}

// MoveTask puts a task in column right after afterID and/or right before
// beforeID; null for either means that end of the column.
func (s *BoardsService) MoveTask(ctx context.Context, taskID, columnID uint64, afterID, beforeID *uint64) (api.Task, error) {
	return call(s.session, func(t string) (api.Task, error) {
		return s.client.MoveTask(ctx, t, taskID, columnID, afterID, beforeID)
	})
}

func (s *BoardsService) DeleteTask(ctx context.Context, taskID uint64) error {
	_, err := call(s.session, func(t string) (struct{}, error) { return struct{}{}, s.client.DeleteTask(ctx, t, taskID) })
	return err
}

func (s *BoardsService) CreateColumn(ctx context.Context, boardID uint64, name string) (api.BoardColumn, error) {
	return call(s.session, func(t string) (api.BoardColumn, error) { return s.client.CreateColumn(ctx, t, boardID, name) })
}

func (s *BoardsService) RenameColumn(ctx context.Context, columnID uint64, name string) (api.BoardColumn, error) {
	return call(s.session, func(t string) (api.BoardColumn, error) { return s.client.RenameColumn(ctx, t, columnID, name) })
}

func (s *BoardsService) MoveColumn(ctx context.Context, columnID uint64, afterID, beforeID *uint64) (api.BoardColumn, error) {
	return call(s.session, func(t string) (api.BoardColumn, error) {
		return s.client.MoveColumn(ctx, t, columnID, afterID, beforeID)
	})
}

// DeleteColumn deletes an empty column; the API refuses one with tasks, and
// the last one.
func (s *BoardsService) DeleteColumn(ctx context.Context, columnID uint64) error {
	_, err := call(s.session, func(t string) (struct{}, error) { return struct{}{}, s.client.DeleteColumn(ctx, t, columnID) })
	return err
}

func (s *BoardsService) WorkTypes(ctx context.Context, guildID uint64) ([]api.WorkType, error) {
	wts, err := call(s.session, func(t string) ([]api.WorkType, error) { return s.client.ListWorkTypes(ctx, t, guildID) })
	if wts == nil {
		wts = []api.WorkType{}
	}
	return wts, err
}

func (s *BoardsService) AddWorkType(ctx context.Context, guildID uint64, key, name string) (api.WorkType, error) {
	return call(s.session, func(t string) (api.WorkType, error) { return s.client.AddWorkType(ctx, t, guildID, key, name) })
}

func (s *BoardsService) RenameWorkType(ctx context.Context, guildID uint64, key, name string) (api.WorkType, error) {
	return call(s.session, func(t string) (api.WorkType, error) {
		return s.client.UpdateWorkType(ctx, t, guildID, key, &name, nil)
	})
}

// MoveWorkType puts a work type at position (from 0) in the guild's list.
func (s *BoardsService) MoveWorkType(ctx context.Context, guildID uint64, key string, position int) (api.WorkType, error) {
	return call(s.session, func(t string) (api.WorkType, error) {
		return s.client.UpdateWorkType(ctx, t, guildID, key, nil, &position)
	})
}

// DeleteWorkType deletes a work type no task has; the API refuses one in use.
func (s *BoardsService) DeleteWorkType(ctx context.Context, guildID uint64, key string) error {
	_, err := call(s.session, func(t string) (struct{}, error) {
		return struct{}{}, s.client.DeleteWorkType(ctx, t, guildID, key)
	})
	return err
}

// DraftTask prioritizes a task for one agent (0 clears it) and/or forbids
// it for agents (nil leaves either as it is).
func (s *BoardsService) DraftTask(ctx context.Context, taskID uint64, prioritizedAgentID *uint64, forbidden *bool) (api.Task, error) {
	return call(s.session, func(t string) (api.Task, error) {
		return s.client.DraftTask(ctx, t, taskID, prioritizedAgentID, forbidden)
	})
}
