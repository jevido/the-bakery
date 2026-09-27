package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

// updaterPublicKey verifies every update. It is the public half of the key
// the release workflow signs with.
//
//go:embed build/updater.pub
var updaterPublicKey []byte

// manifestURL is the signed update manifest of the latest desktop release.
// BAKERY_UPDATE_MANIFEST_URL overrides it, for testing against a local server.
const manifestURL = "https://github.com/jevido/the-bakery/releases/latest/download/manifest.json"

// UpdateService keeps the app up to date from GitHub releases. The Wails
// updater checks, downloads and verifies (a bad signature is refused); the
// frontend listens to its wails:updater:* events and asks for the install.
type UpdateService struct {
	app     *application.App
	logger  *slog.Logger
	version string
	enabled bool
}

func (s *UpdateService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if s.version == "dev" {
		return nil // dev builds never update themselves
	}
	provider, err := endpoint.New(endpoint.Config{URL: envOr("BAKERY_UPDATE_MANIFEST_URL", manifestURL)})
	if err != nil {
		return fmt.Errorf("update provider: %w", err)
	}
	err = s.app.Updater.Init(updater.Config{
		CurrentVersion: s.version,
		Providers:      []updater.Provider{provider},
		PublicKey:      updaterPublicKey,
		CheckInterval:  6 * time.Hour,
		Window:         updater.WindowNone,
	})
	if err != nil {
		// Not fatal: the app works without updates.
		s.logger.Warn("updates disabled", "err", err)
		return nil
	}
	s.enabled = true
	// The periodic check waits a full interval; look once shortly after start.
	go func() {
		select {
		case <-time.After(5 * time.Second):
			s.Check(ctx)
		case <-ctx.Done():
		}
	}()
	return nil
}

// Version is the running version ("dev" for development builds).
func (s *UpdateService) Version() string { return s.version }

// Check looks for a newer release now. The answer arrives as updater events.
func (s *UpdateService) Check(ctx context.Context) {
	if !s.enabled {
		return
	}
	if _, err := s.app.Updater.Check(ctx); err != nil {
		s.logger.Warn("update check failed", "err", err)
	}
}

// Install downloads and verifies the update, puts it in place and restarts
// the app. Progress and errors arrive as updater events too.
func (s *UpdateService) Install(ctx context.Context) error {
	if !s.enabled {
		return errors.New("updates are not available in this build")
	}
	if err := s.app.Updater.DownloadAndInstall(ctx); err != nil {
		return err
	}
	// Inside an AppImage the running binary sits on a read-only mount, so the
	// updater's own swap cannot work; replace the AppImage file instead.
	if appImage := os.Getenv("APPIMAGE"); appImage != "" {
		return s.replaceAppImage(appImage, s.app.Updater.DownloadedPath())
	}
	return s.app.Updater.Restart(ctx)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
