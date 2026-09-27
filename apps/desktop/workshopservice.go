package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jevido/the-bakery/apps/desktop/internal/agents"
	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/session"
	"github.com/jevido/the-bakery/apps/desktop/internal/workshop"
)

// WorkshopService is how the frontend reaches this machine's workshop:
// board configs and runs. Board configs never go to the API.
type WorkshopService struct {
	app     *application.App
	client  *api.Client
	session *session.Session
	agents  *agents.Engine
	boards  *workshop.Boards
}

func NewWorkshopService(client *api.Client, s *session.Session, agentFolders *agents.Engine) *WorkshopService {
	return &WorkshopService{
		client: client, session: s, agents: agentFolders,
		boards: workshop.NewBoards(configBase(), apiHost(client)),
	}
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

// preparedRun is a run ready to start: its worktree has the skills, its
// files are written, and the spec says how to start Claude.
type preparedRun struct {
	ID       string
	Config   workshop.BoardConfig
	Worktree workshop.Worktree
	Skills   []string
	Spec     workshop.RunSpec
	Dir      string
	Agent    agents.Manifest
	Task     api.Task
}

// prepareRun gathers everything a run of the agent on the task needs,
// makes the worktree, places the skills and writes the run's files.
func (s *WorkshopService) prepareRun(ctx context.Context, boardID, taskID uint64, agentSlug string) (preparedRun, error) {
	token := s.session.Token()
	if token == "" {
		return preparedRun{}, ErrSignedOut
	}
	cfg, err := s.boards.Load(boardID)
	if err != nil {
		return preparedRun{}, err
	}
	if !cfg.Linked() {
		return preparedRun{}, errors.New("this board is not linked to a repository on this machine; open Board settings")
	}
	if err := cfg.Validate(ctx); err != nil {
		return preparedRun{}, fmt.Errorf("board settings: %w", err)
	}
	folder, err := s.agents.Folder(agentSlug)
	if err != nil {
		return preparedRun{}, fmt.Errorf("agent %s: %w", agentSlug, err)
	}
	if folder.ManifestErr != nil {
		return preparedRun{}, fmt.Errorf("agent %s: %w", agentSlug, folder.ManifestErr)
	}
	view, err := s.client.GetBoard(ctx, token, boardID)
	if err != nil {
		return preparedRun{}, err
	}
	detail, err := s.client.GetTask(ctx, token, taskID)
	if err != nil {
		return preparedRun{}, err
	}
	if detail.Task.BoardID != boardID {
		return preparedRun{}, errors.New("that task is on another board")
	}
	if detail.Task.ParentID != nil {
		return preparedRun{}, errors.New("agents work tasks on the board, not subtasks")
	}
	comments, err := s.client.ListComments(ctx, token, taskID)
	if err != nil {
		return preparedRun{}, err
	}
	traits, _, err := s.client.Traits(ctx, token)
	if err != nil {
		return preparedRun{}, err
	}
	runnerToken, err := s.session.RunnerToken(ctx)
	if err != nil {
		return preparedRun{}, err
	}

	wt, err := workshop.PrepareWorktree(ctx, cfg, taskID, detail.Task.Title)
	if err != nil {
		return preparedRun{}, err
	}
	boardDir := s.boards.Dir(boardID)
	skills, err := workshop.AssembleSkills(wt, agentSlug, filepath.Join(folder.Dir, "skills"), filepath.Join(boardDir, "skills"))
	if err != nil {
		return preparedRun{}, err
	}

	m := folder.Manifest
	agent := workshop.RunAgent{
		Slug: agentSlug, Name: m.Name, Title: m.Title, Backstory: m.Backstory,
		Model: m.Model, PermissionMode: m.PermissionMode, AllowedTools: m.AllowedTools,
	}
	for _, key := range m.Traits {
		for _, t := range traits {
			if t.Key == key && t.Instruction != "" {
				agent.TraitInstructions = append(agent.TraitInstructions, t.Instruction)
			}
		}
	}
	task := workshop.RunTask{ID: taskID, Title: detail.Task.Title, Description: detail.Task.Description, BoardName: view.Board.Name}
	for _, c := range view.Columns {
		if detail.Task.ColumnID != nil && c.ID == *detail.Task.ColumnID {
			task.ColumnName = c.Name
		}
	}
	for _, st := range detail.Subtasks {
		task.Subtasks = append(task.Subtasks, workshop.RunSubtask{Title: st.Title, Done: st.Done})
	}
	for _, c := range comments {
		task.Comments = append(task.Comments, workshop.RunComment{Author: c.AuthorName, Body: c.Body})
	}

	id := newUUID()
	runDir := filepath.Join(boardDir, "runs", id)
	instructions, _ := os.ReadFile(filepath.Join(boardDir, "CLAUDE.md"))
	boardMCP, _ := os.ReadFile(filepath.Join(boardDir, "mcp.json"))
	spec, err := workshop.BuildRunSpec(workshop.RunInput{
		RunID: id, Task: task, Agent: agent, Board: cfg, Worktree: wt, RunDir: runDir,
		BoardInstructions: string(instructions), BoardMCP: boardMCP,
		MCPURL: strings.TrimRight(s.client.BaseURL(), "/") + "/mcp", Token: runnerToken,
	})
	if err != nil {
		return preparedRun{}, err
	}
	// The run folder holds the runner token (in mcp.json): only for this user.
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return preparedRun{}, err
	}
	for path, content := range spec.Files {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			return preparedRun{}, err
		}
	}
	if err := os.WriteFile(filepath.Join(runDir, "prompt.md"), []byte(spec.Stdin), 0o600); err != nil {
		return preparedRun{}, err
	}
	return preparedRun{ID: id, Config: cfg, Worktree: wt, Skills: skills, Spec: spec, Dir: runDir, Agent: m, Task: detail.Task}, nil
}

// RunCommand is a prepared run as a command to paste into a terminal.
type RunCommand struct {
	Command string   `json:"command"`
	Skills  []string `json:"skills"`
}

// PrintRunCommand prepares a run of the agent on the task (worktree,
// skills, files) without starting it, and says how to start it by hand. For
// checking a board's setup; the run is not reported to the API.
func (s *WorkshopService) PrintRunCommand(ctx context.Context, boardID, taskID uint64, agentSlug string) (RunCommand, error) {
	run, err := s.prepareRun(ctx, boardID, taskID, agentSlug)
	if err != nil {
		return RunCommand{}, err
	}
	quoted := make([]string, len(run.Spec.Args))
	for i, a := range run.Spec.Args {
		quoted[i] = shellQuote(a)
	}
	cmd := "cd " + shellQuote(run.Spec.Dir) + " && claude " + strings.Join(quoted, " ") + " < " + shellQuote(filepath.Join(run.Dir, "prompt.md"))
	return RunCommand{Command: cmd, Skills: run.Skills}, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// newUUID makes a random (version 4) UUID.
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
