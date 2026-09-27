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
func call[T any](s *BoardsService, f func(token string) (T, error)) (T, error) {
	token := s.session.Token()
	if token == "" {
		var zero T
		return zero, ErrSignedOut
	}
	v, err := f(token)
	if errors.Is(err, api.ErrUnauthorized) {
		_ = s.session.Logout()
		return v, ErrSignedOut
	}
	return v, err
}

func (s *BoardsService) ListGuilds(ctx context.Context) ([]api.Guild, error) {
	return call(s, func(t string) ([]api.Guild, error) { return s.client.ListGuilds(ctx, t) })
}

func (s *BoardsService) ListBoards(ctx context.Context, guildID uint64) ([]api.Board, error) {
	return call(s, func(t string) ([]api.Board, error) { return s.client.ListBoards(ctx, t, guildID) })
}

func (s *BoardsService) CreateBoard(ctx context.Context, guildID uint64, name string) (api.Board, error) {
	return call(s, func(t string) (api.Board, error) { return s.client.CreateBoard(ctx, t, guildID, name) })
}

func (s *BoardsService) GetBoard(ctx context.Context, boardID uint64) (api.BoardView, error) {
	return call(s, func(t string) (api.BoardView, error) { return s.client.GetBoard(ctx, t, boardID) })
}

func (s *BoardsService) CreateTask(ctx context.Context, boardID uint64, title string) (api.Task, error) {
	return call(s, func(t string) (api.Task, error) { return s.client.CreateTask(ctx, t, boardID, title) })
}

func (s *BoardsService) UpdateTask(ctx context.Context, taskID uint64, title, description *string) (api.Task, error) {
	return call(s, func(t string) (api.Task, error) { return s.client.UpdateTask(ctx, t, taskID, title, description) })
}

// MoveTask puts a task in column right after afterID and/or right before
// beforeID; null for either means that end of the column.
func (s *BoardsService) MoveTask(ctx context.Context, taskID uint64, column string, afterID, beforeID *uint64) (api.Task, error) {
	return call(s, func(t string) (api.Task, error) {
		return s.client.MoveTask(ctx, t, taskID, column, afterID, beforeID)
	})
}

func (s *BoardsService) DeleteTask(ctx context.Context, taskID uint64) error {
	_, err := call(s, func(t string) (struct{}, error) { return struct{}{}, s.client.DeleteTask(ctx, t, taskID) })
	return err
}
