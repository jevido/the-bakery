package main

import (
	"context"
	"errors"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/workshop"
)

// WorkshopService is how the frontend reaches this machine's workshop: board
// configs now, runs later. Board configs never go to the API.
type WorkshopService struct {
	app    *application.App
	boards *workshop.Boards
}

func NewWorkshopService(client *api.Client) *WorkshopService {
	return &WorkshopService{boards: workshop.NewBoards(configBase(), apiHost(client))}
}

// BoardSettings is a board's config on this machine, with whether it can
// run agents and, if not, why.
type BoardSettings struct {
	Config workshop.BoardConfig `json:"config"`
	Dir    string               `json:"dir"`
	// Linked is true when the repo is set and valid.
	Linked bool `json:"linked"`
	// Problem says what is wrong with a saved config (the repo moved, the
	// branch is gone), "" when nothing is.
	Problem string `json:"problem"`
}

// GetBoardConfig reads a board's config, or the defaults.
func (s *WorkshopService) GetBoardConfig(ctx context.Context, boardID uint64) (BoardSettings, error) {
	cfg, err := s.boards.Load(boardID)
	if err != nil {
		return BoardSettings{}, err
	}
	out := BoardSettings{Config: cfg, Dir: s.boards.Dir(boardID)}
	if err := cfg.Validate(ctx); err != nil {
		out.Problem = err.Error()
	} else {
		out.Linked = cfg.Linked()
	}
	return out, nil
}

// ConfigProblem is a refused save: which field, and why.
type ConfigProblem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// SaveBoardConfig checks a board's config against this machine and saves
// it. A config that does not check out is not saved.
func (s *WorkshopService) SaveBoardConfig(ctx context.Context, boardID uint64, cfg workshop.BoardConfig) (*ConfigProblem, error) {
	if err := cfg.Validate(ctx); err != nil {
		var ce *workshop.ConfigError
		if errors.As(err, &ce) {
			return &ConfigProblem{Field: ce.Field, Message: ce.Message}, nil
		}
		return nil, err
	}
	return nil, s.boards.Save(boardID, cfg)
}

// OpenBoardConfigFolder shows the board's config folder in the file
// manager, making it first if needed.
func (s *WorkshopService) OpenBoardConfigFolder(boardID uint64) error {
	dir := s.boards.Dir(boardID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return s.app.Browser.OpenFile(dir)
}

// PickRepo asks for a repository folder. It returns "" when the member
// cancels.
func (s *WorkshopService) PickRepo() (string, error) {
	return s.app.Dialog.OpenFile().SetTitle("Choose the git repository for this board").
		CanChooseDirectories(true).CanChooseFiles(false).PromptForSingleSelection()
}
