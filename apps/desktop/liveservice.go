package main

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/session"
)

// Events LiveService sends the frontend.
const (
	// eventBoard carries one api.BoardEvent.
	eventBoard = "board:event"
	// eventStatus carries a liveStatus.
	eventStatus = "board:status"
	// eventResync says events may have been missed: refetch the board.
	eventResync = "board:resync"
)

// liveStatus is the state of the board's stream, for the indicator.
type liveStatus struct {
	BoardID uint64 `json:"board_id"`
	// State is "connecting", "live", "reconnecting" or "signed-out".
	State string `json:"state"`
}

// LiveService keeps one board's event stream open, from Go, and passes its
// events to the frontend as Wails events. It reconnects with backoff
// (1 s up to 30 s, with jitter) and asks the frontend to refetch after
// every reconnect, since missed events are not replayed.
type LiveService struct {
	client  *api.Client
	session *session.Session
	logger  *slog.Logger
	app     *application.App

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewLiveService(client *api.Client, s *session.Session, logger *slog.Logger) *LiveService {
	return &LiveService{client: client, session: s, logger: logger}
}

// Watch starts following the board, and stops following any other.
func (s *LiveService) Watch(boardID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.follow(ctx, boardID)
}

// Unwatch stops following the board.
func (s *LiveService) Unwatch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

func (s *LiveService) ServiceShutdown() error {
	s.Unwatch()
	return nil
}

func (s *LiveService) emit(ctx context.Context, name string, data any) {
	if ctx.Err() == nil && s.app != nil {
		s.app.Event.Emit(name, data)
	}
}

func (s *LiveService) follow(ctx context.Context, boardID uint64) {
	backoff := time.Second
	first := true
	s.emit(ctx, eventStatus, liveStatus{boardID, "connecting"})
	for ctx.Err() == nil {
		token := s.session.Token()
		if token == "" {
			s.emit(ctx, eventStatus, liveStatus{boardID, "signed-out"})
			return
		}
		opened := time.Now()
		err := s.client.WatchBoard(ctx, token, boardID,
			func() {
				s.emit(ctx, eventStatus, liveStatus{boardID, "live"})
				if !first {
					s.emit(ctx, eventResync, liveStatus{boardID, "live"})
				}
				first = false
			},
			func(ev api.BoardEvent) { s.emit(ctx, eventBoard, ev) },
		)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, api.ErrUnauthorized) {
			_ = s.session.Logout(context.WithoutCancel(ctx))
			s.emit(ctx, eventStatus, liveStatus{boardID, "signed-out"})
			return
		}
		// A stream that stayed up a while earns a fresh, short backoff.
		if time.Since(opened) > time.Minute {
			backoff = time.Second
		}
		s.logger.Info("board stream ended, reconnecting", "board", boardID, "in", backoff, "err", err)
		s.emit(ctx, eventStatus, liveStatus{boardID, "reconnecting"})
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff + rand.N(backoff/2)):
		}
		backoff = min(backoff*2, 30*time.Second)
	}
}
