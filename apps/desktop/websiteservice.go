package main

import (
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// WebsiteService opens pages of The Bakery's website (where guilds are
// founded and run) in the system browser.
type WebsiteService struct {
	app     *application.App
	baseURL string
}

// OpenAdmin opens the guild admin, /admin.
func (s *WebsiteService) OpenAdmin() error {
	return s.app.Browser.OpenURL(strings.TrimRight(s.baseURL, "/") + "/admin")
}
