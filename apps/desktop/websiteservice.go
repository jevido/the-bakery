package main

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/session"
)

// WebsiteService opens pages of The Bakery's website (where guilds are
// founded and run) in the system browser, signed in as the app's member.
type WebsiteService struct {
	app     *application.App
	client  *api.Client
	session *session.Session
	logger  *slog.Logger
	baseURL string
}

// OpenAdmin opens the guild admin, /admin.
func (s *WebsiteService) OpenAdmin(ctx context.Context) error {
	return s.open(ctx, "/admin")
}

// open opens path on the website. When a member is signed in, it goes
// through /handoff with a one-time code so the website signs them in too;
// the token itself never goes into the URL. If that fails, the page opens
// signed out and asks to sign in.
func (s *WebsiteService) open(ctx context.Context, path string) error {
	base := strings.TrimRight(s.baseURL, "/")
	target := base + path
	if token := s.session.Token(); token != "" {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		code, err := s.client.Handoff(ctx, token)
		if err == nil {
			target = base + "/handoff?" + url.Values{"code": {code}, "next": {path}}.Encode()
		} else {
			s.logger.Warn("could not sign in on the website; opening it signed out", "err", err)
		}
	}
	return s.app.Browser.OpenURL(target)
}
