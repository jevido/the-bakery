package main

import (
	"cmp"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

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

	runsMu sync.Mutex
	runs   map[string]*localRun
}

func NewWorkshopService(client *api.Client, s *session.Session, agentFolders *agents.Engine) *WorkshopService {
	return &WorkshopService{
		client: client, session: s, agents: agentFolders,
		boards: workshop.NewBoards(configBase(), apiHost(client)),
		runs:   map[string]*localRun{},
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

// Wails events for runs: every event of one run on run:<id>, and the list of
// this machine's runs whenever one starts or ends.
const (
	eventRunPrefix = "run:"
	eventRuns      = "workshop:runs"
	// keptRunEvents is how many events a run keeps for a panel opened late.
	keptRunEvents = 500
)

// RunInfo is one run on this machine, for the frontend.
type RunInfo struct {
	ID             string    `json:"id"`
	APIRunID       uint64    `json:"api_run_id"`
	BoardID        uint64    `json:"board_id"`
	TaskID         uint64    `json:"task_id"`
	TaskTitle      string    `json:"task_title"`
	AgentSlug      string    `json:"agent_slug"`
	AgentName      string    `json:"agent_name"`
	Branch         string    `json:"branch"`
	Worktree       string    `json:"worktree"`
	Model          string    `json:"model"`
	PermissionMode string    `json:"permission_mode"`
	StartedAt      time.Time `json:"started_at"`
	// Status is "running", "succeeded", "failed" or "stopped".
	Status     string  `json:"status"`
	CostUSD    float64 `json:"cost_usd"`
	Turns      int     `json:"turns"`
	DurationMS int64   `json:"duration_ms"`
}

// localRun is a run this app started, with what it has said so far.
type localRun struct {
	mu     sync.Mutex
	info   RunInfo
	skills []string
	events []workshop.RunEvent
	proc   *workshop.Process
}

func (r *localRun) snapshot() RunInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.info
}

func (s *WorkshopService) claudeBinary() string {
	return cmp.Or(os.Getenv("BAKERY_CLAUDE"), "claude")
}

// StartRun puts the agent to work on the task on this machine: it prepares
// the worktree and the run's files, records the run with the API, and
// starts Claude. It returns the run's id on this machine.
func (s *WorkshopService) StartRun(ctx context.Context, boardID, taskID uint64, agentSlug string) (RunInfo, error) {
	cfg, err := s.boards.Load(boardID)
	if err != nil {
		return RunInfo{}, err
	}
	s.runsMu.Lock()
	running := 0
	for _, r := range s.runs {
		info := r.snapshot()
		if info.Status != "running" {
			continue
		}
		if info.TaskID == taskID {
			s.runsMu.Unlock()
			return RunInfo{}, fmt.Errorf("%s is already working on this task", info.AgentName)
		}
		if info.BoardID == boardID {
			running++
		}
	}
	s.runsMu.Unlock()
	if running >= cfg.MaxConcurrentRuns {
		return RunInfo{}, fmt.Errorf("this board already has %d runs going on this machine, its limit; wait for one to end or raise it in Board settings", running)
	}
	if _, err := exec.LookPath(s.claudeBinary()); err != nil {
		return RunInfo{}, errors.New("claude is not installed or not on the PATH of this app")
	}
	folder, err := s.agents.Folder(agentSlug)
	if err != nil {
		return RunInfo{}, fmt.Errorf("agent %s: %w", agentSlug, err)
	}
	if folder.Sync == nil || folder.Sync.AgentID == 0 {
		return RunInfo{}, fmt.Errorf("%s has not synced yet; wait for the sync, then try again", folder.Manifest.Name)
	}

	prep, err := s.prepareRun(ctx, boardID, taskID, agentSlug)
	if err != nil {
		return RunInfo{}, err
	}
	host, _ := os.Hostname()
	apiRun, err := s.client.StartRun(ctx, s.session.Token(), taskID, folder.Sync.AgentID, prep.Agent.Name, host, prep.Worktree.Branch)
	if err != nil {
		return RunInfo{}, fmt.Errorf("recording the run: %w", err)
	}
	run := &localRun{
		info: RunInfo{
			ID: prep.ID, APIRunID: apiRun.ID, BoardID: boardID, TaskID: taskID, TaskTitle: prep.Task.Title,
			AgentSlug: agentSlug, AgentName: prep.Agent.Name, Branch: prep.Worktree.Branch, Worktree: prep.Worktree.Path,
			Model: prep.Agent.Model, PermissionMode: prep.Agent.PermissionMode, StartedAt: time.Now(), Status: "running",
		},
		skills: prep.Skills,
	}
	s.runsMu.Lock()
	s.runs[prep.ID] = run
	s.runsMu.Unlock()

	// The run outlives the request that started it.
	proc, err := workshop.StartProcess(context.WithoutCancel(ctx), s.claudeBinary(), prep.Spec, prep.Dir, func(ev workshop.RunEvent) { s.record(run, ev) })
	if err != nil {
		s.end(run, workshop.Outcome{ExitCode: -1, Stderr: err.Error()})
		return RunInfo{}, err
	}
	run.mu.Lock()
	run.proc = proc
	run.mu.Unlock()
	s.emitRuns()
	go func() { s.end(run, proc.Wait()) }()
	return run.snapshot(), nil
}

// record keeps one event of a run and passes it to the frontend.
func (s *WorkshopService) record(run *localRun, ev workshop.RunEvent) {
	run.mu.Lock()
	var notes []workshop.RunEvent
	switch ev.Kind {
	case "init":
		for _, want := range run.skills {
			if !slices.Contains(ev.Skills, want) {
				notes = append(notes, workshop.RunEvent{Kind: "note", Text: "The skill " + want + " did not load."})
			}
		}
	case "result":
		run.info.CostUSD, run.info.Turns, run.info.DurationMS = ev.CostUSD, ev.Turns, ev.DurationMS
	}
	for _, e := range append([]workshop.RunEvent{ev}, notes...) {
		run.events = append(run.events, e)
		if len(run.events) > keptRunEvents {
			run.events = run.events[len(run.events)-keptRunEvents:]
		}
	}
	id := run.info.ID
	run.mu.Unlock()
	if s.app != nil {
		s.app.Event.Emit(eventRunPrefix+id, ev)
		for _, n := range notes {
			s.app.Event.Emit(eventRunPrefix+id, n)
		}
	}
}

// end closes a run once its process is gone: its status here, and its
// record on the API.
func (s *WorkshopService) end(run *localRun, out workshop.Outcome) {
	status := "failed"
	switch {
	case out.Stopped:
		status = "stopped"
	case out.Result != nil && out.Result.Subtype == "success" && !out.Result.IsError:
		status = "succeeded"
	}
	summary := out.LastText
	if out.Result != nil && strings.TrimSpace(out.Result.Text) != "" {
		summary = out.Result.Text
	}
	if status == "failed" && strings.TrimSpace(summary) == "" {
		summary = lastLine(out.Stderr)
	}
	note := fmt.Sprintf("Run %s.", status)
	if out.ExitCode != 0 && !out.Stopped {
		note = fmt.Sprintf("Claude exited with code %d. %s", out.ExitCode, lastLine(out.Stderr))
	}
	s.record(run, workshop.RunEvent{Kind: "note", Text: strings.TrimSpace(note)})

	run.mu.Lock()
	run.info.Status = status
	info := run.info
	run.mu.Unlock()
	s.emitRuns()

	end := api.RunEnd{Status: status, CostUSD: info.CostUSD, Turns: info.Turns, Summary: trimSummary(summary)}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := s.client.FinishRun(ctx, s.session.Token(), info.APIRunID, end); err != nil {
		s.record(run, workshop.RunEvent{Kind: "note", Text: "Could not record the end of the run: " + err.Error()})
	}
}

// StopRun asks a run to stop; it ends as stopped within a few seconds.
func (s *WorkshopService) StopRun(id string) error {
	s.runsMu.Lock()
	run := s.runs[id]
	s.runsMu.Unlock()
	if run == nil {
		return errors.New("no such run on this machine")
	}
	run.mu.Lock()
	proc := run.proc
	run.mu.Unlock()
	if proc != nil {
		proc.Stop()
	}
	return nil
}

// RunEvents returns what a run has said so far (its last 500 events).
func (s *WorkshopService) RunEvents(id string) []workshop.RunEvent {
	s.runsMu.Lock()
	run := s.runs[id]
	s.runsMu.Unlock()
	if run == nil {
		return []workshop.RunEvent{}
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	return slices.Clone(run.events)
}

// Runs lists the runs this app started since it opened, newest first.
func (s *WorkshopService) Runs() []RunInfo {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	out := make([]RunInfo, 0, len(s.runs))
	for _, r := range s.runs {
		out = append(out, r.snapshot())
	}
	slices.SortFunc(out, func(a, b RunInfo) int { return b.StartedAt.Compare(a.StartedAt) })
	return out
}

func (s *WorkshopService) emitRuns() {
	if s.app != nil {
		s.app.Event.Emit(eventRuns, s.Runs())
	}
}

// ServiceShutdown stops the runs still going, and waits a little for them
// to be recorded as stopped.
func (s *WorkshopService) ServiceShutdown() error {
	s.runsMu.Lock()
	var procs []*workshop.Process
	for _, r := range s.runs {
		r.mu.Lock()
		if r.proc != nil && r.info.Status == "running" {
			procs = append(procs, r.proc)
		}
		r.mu.Unlock()
	}
	s.runsMu.Unlock()
	for _, p := range procs {
		p.Stop()
	}
	deadline := time.After(8 * time.Second)
	for _, p := range procs {
		select {
		case <-p.Done():
		case <-deadline:
			return nil
		}
	}
	// Give the last API calls a moment.
	time.Sleep(500 * time.Millisecond)
	return nil
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "\n"); i >= 0 {
		s = s[i+1:]
	}
	return s
}

// trimSummary keeps a summary to 2 000 characters.
func trimSummary(s string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 2000 {
		return string(r[:2000]) + "…"
	}
	return s
}
