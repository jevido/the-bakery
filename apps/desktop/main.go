package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The frontend build is embedded; `all:` keeps files Vite names with a
// leading underscore.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "The Bakery",
		Description: "A desktop workbench where guilds plan and run their work",
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
