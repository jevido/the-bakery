package main

import (
	"cmp"
	"embed"
	"log"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/session"
)

// The frontend build is embedded; `all:` keeps files Vite names with a
// leading underscore.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := api.New(cmp.Or(os.Getenv("BAKERY_API_URL"), api.DefaultURL))
	sess := session.New(client, session.NewStore(client.BaseURL(), logger))

	app := application.New(application.Options{
		Name:        "The Bakery",
		Description: "A desktop workbench where guilds plan and run their work",
		Logger:      logger,
		Services: []application.Service{
			application.NewService(NewSessionService(sess, logger)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "The Bakery",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		// Matches --bg in frontend/src/theme.css so there is no flash on start.
		BackgroundColour: application.NewRGB(0x1c, 0x1d, 0x1a),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
