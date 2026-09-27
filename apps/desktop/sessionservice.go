package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/session"
)

// SessionService is the frontend's way to sign in and out. The token stays
// in Go; the frontend only ever sees the member.
type SessionService struct {
	session *session.Session
	logger  *slog.Logger
}

func NewSessionService(s *session.Session, logger *slog.Logger) *SessionService {
	return &SessionService{session: s, logger: logger}
}

// ServiceStartup restores the last session so a relaunch skips the login.
func (s *SessionService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.session.Restore(ctx); err != nil {
		// Not fatal: the member can still sign in once the API is back.
		s.logger.Warn("could not restore the session", "err", err)
	}
	return nil
}

func (s *SessionService) Register(ctx context.Context, email, displayName, password string) (api.Member, error) {
	return s.session.Register(ctx, email, displayName, password)
}

func (s *SessionService) Login(ctx context.Context, email, password string) (api.Member, error) {
	return s.session.Login(ctx, email, password)
}

// Logout is "Clock out": the token is forgotten on this machine.
func (s *SessionService) Logout(ctx context.Context) error {
	return s.session.Logout(ctx)
}

// Me returns the signed-in member, or null when nobody is signed in.
func (s *SessionService) Me() *api.Member {
	return s.session.Member()
}
