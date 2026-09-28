package main

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	log     *slog.Logger

	runsMu sync.Mutex
	runs   map[string]*localRun

	startMu   sync.Mutex
	ends      sync.WaitGroup
	machineID string
	// wake nudges the scheduler: a run ended, a board changed.
	wake chan struct{}
	// desk turns runs' permission prompts and questions into letters.
	desk *workshop.Desk
	// notify, when set, tells the desktop about a new letter (main sends it
	// as a notification while the window is not focused).
	notify func(title, body string)
}

func NewWorkshopService(client *api.Client, s *session.Session, agentFolders *agents.Engine, logger *slog.Logger) *WorkshopService {
	return &WorkshopService{
		client: client, session: s, agents: agentFolders, log: logger,
		boards:    workshop.NewBoards(configBase(), apiHost(client)),
		runs:      map[string]*localRun{},
		machineID: machineID(),
		wake:      make(chan struct{}, 1),
		desk:      workshop.NewDesk(),
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
	if err := s.boards.Save(boardID, cfg); err != nil {
		return nil, err
	}
	s.Wake()
	return nil, nil
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
		Desk: s.deskFor(id),
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
	Status string `json:"status"`
	// Waiting counts the run's letters waiting for an answer.
	Waiting int `json:"waiting"`
	// Kind is "work", or "plan" for Plan it.
	Kind string `json:"kind"`
	// State is what the agent is doing, for the colony view: thinking,
	// editing, running (a command), waiting (for an answer), done, failed.
	State      string  `json:"state"`
	CostUSD    float64 `json:"cost_usd"`
	Turns      int     `json:"turns"`
	DurationMS int64   `json:"duration_ms"`
	// Once the run has ended: what it left in its worktree, the column the
	// task moved to ("" when it did not move), and whether the worktree was
	// removed since.
	Diff            workshop.WorktreeDiff `json:"diff"`
	MovedTo         string                `json:"moved_to"`
	WorktreeRemoved bool                  `json:"worktree_removed"`
}

// localRun is a run this app started, with what it has said so far.
type localRun struct {
	mu     sync.Mutex
	info   RunInfo
	skills []string
	// What the finish needs: the repo, the base branch, the finish column
	// and the run's folder.
	repo, base, finishColumn, dir string
	seq                           int
	// The claim holding the task while the run goes, kept by heartbeat
	// until stopHeartbeat closes.
	claimID       uint64
	agentID       uint64
	stopHeartbeat chan struct{}
	events        []workshop.RunEvent
	proc          *workshop.Process
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
	// One start at a time, so two starts never both fit under the limit.
	s.startMu.Lock()
	defer s.startMu.Unlock()
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
		if info.AgentSlug == agentSlug {
			s.runsMu.Unlock()
			return RunInfo{}, fmt.Errorf("%s is busy with another task", info.AgentName)
		}
		if info.BoardID == boardID {
			running++
		}
	}
	s.runsMu.Unlock()
	// A person may assign up to the normal limit even while the board is
	// paused; the scheduler keeps to the limit of the board's speed.
	if running >= max(cfg.RunLimit(), cfg.MaxConcurrentRuns) {
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

	// The claim first: if another machine has the task, nothing is made.
	claim, err := s.client.ClaimTask(ctx, s.session.Token(), taskID, folder.Sync.AgentID, s.machineID)
	if err != nil {
		return RunInfo{}, err
	}
	release := func() {
		_ = s.client.ReleaseClaim(context.WithoutCancel(ctx), s.session.Token(), claim.ID)
	}
	prep, err := s.prepareRun(ctx, boardID, taskID, agentSlug)
	if err != nil {
		release()
		return RunInfo{}, err
	}
	host, _ := os.Hostname()
	apiRun, err := s.client.StartRun(ctx, s.session.Token(), taskID, folder.Sync.AgentID, prep.Agent.Name, "work", host, prep.Worktree.Branch)
	if err != nil {
		release()
		return RunInfo{}, fmt.Errorf("recording the run: %w", err)
	}
	run := &localRun{
		info: RunInfo{
			ID: prep.ID, APIRunID: apiRun.ID, BoardID: boardID, TaskID: taskID, TaskTitle: prep.Task.Title,
			AgentSlug: agentSlug, AgentName: prep.Agent.Name, Branch: prep.Worktree.Branch, Worktree: prep.Worktree.Path,
			Model: prep.Agent.Model, PermissionMode: prep.Agent.PermissionMode, StartedAt: time.Now(), Status: "running", State: "thinking", Kind: "work",
		},
		skills: prep.Skills,
		repo:   prep.Config.Repo, base: prep.Worktree.Base, finishColumn: prep.Config.FinishColumn, dir: prep.Dir,
		claimID: claim.ID, agentID: folder.Sync.AgentID, stopHeartbeat: make(chan struct{}),
	}
	s.runsMu.Lock()
	s.runs[prep.ID] = run
	s.runsMu.Unlock()
	s.saveRunFile(run)

	// The run outlives the request that started it.
	proc, err := workshop.StartProcess(context.WithoutCancel(ctx), s.claudeBinary(), prep.Spec, prep.Dir, func(ev workshop.RunEvent) { s.record(run, ev) })
	if err != nil {
		s.end(run, workshop.Outcome{ExitCode: -1, Stderr: err.Error()})
		return RunInfo{}, err
	}
	run.mu.Lock()
	run.proc = proc
	run.mu.Unlock()
	s.saveRunFile(run)
	s.emitRuns()
	go s.heartbeat(run)
	s.ends.Go(func() { s.end(run, proc.Wait()) })
	return run.snapshot(), nil
}

// record keeps one event of a run and passes it to the frontend.
func (s *WorkshopService) record(run *localRun, ev workshop.RunEvent) {
	run.mu.Lock()
	var notes []workshop.RunEvent
	before := run.info.State
	if st := workshop.StateAfter(run.info.State, ev); st != "" {
		run.info.State = st
	}
	if run.info.Waiting > 0 && run.info.Status == "running" {
		run.info.State = "waiting"
	}
	stateChanged := run.info.State != before
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
	all := append([]workshop.RunEvent{ev}, notes...)
	for i := range all {
		run.seq++
		all[i].Seq = run.seq
		run.events = append(run.events, all[i])
		if len(run.events) > keptRunEvents {
			run.events = run.events[len(run.events)-keptRunEvents:]
		}
	}
	id := run.info.ID
	state := agentState{Agent: run.info.AgentSlug, Run: id, Task: run.info.TaskID, State: run.info.State}
	run.mu.Unlock()
	if s.app != nil {
		for _, e := range all {
			s.app.Event.Emit(eventRunPrefix+id, e)
		}
		if stateChanged {
			s.app.Event.Emit(eventAgentState, state)
		}
	}
}

// end finishes a run once its process is gone: the diff it left, its
// record on the API, a comment on the task, and the move to the board's
// finish column when it succeeded.
func (s *WorkshopService) end(run *localRun, out workshop.Outcome) {
	status := workshop.EndStatus(out)
	summary := workshop.Summary(out)
	problem := ""
	switch {
	case out.Stopped:
		problem = "Stopped from the run panel."
	case status == "failed" && out.Result != nil:
		problem = "Claude ended with " + out.Result.Subtype + "."
	case status == "failed":
		problem = strings.TrimSpace(fmt.Sprintf("Claude exited with code %d. %s", out.ExitCode, lastLine(out.Stderr)))
	}
	if problem != "" {
		s.record(run, workshop.RunEvent{Kind: "note", Text: problem})
	}

	run.mu.Lock()
	info, wt := run.info, workshop.Worktree{Path: run.info.Worktree, Branch: run.info.Branch, Base: run.base}
	finishColumn := run.finishColumn
	run.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	diff, err := workshop.Diff(ctx, wt)
	if err != nil {
		s.record(run, workshop.RunEvent{Kind: "note", Text: "Could not count the changes: " + err.Error()})
	}
	token := s.session.Token()
	end := api.RunEnd{
		Status: status, CostUSD: info.CostUSD, Turns: info.Turns, Summary: summary,
		FilesChanged: diff.Committed.Files, Additions: diff.Committed.Additions, Deletions: diff.Committed.Deletions,
	}
	if _, err := s.client.FinishRun(ctx, token, info.APIRunID, end); err != nil {
		s.record(run, workshop.RunEvent{Kind: "note", Text: "Could not record the end of the run: " + err.Error()})
	}
	comment := workshop.FinishComment(workshop.RunReport{
		AgentName: info.AgentName, Branch: info.Branch, Status: status, Diff: diff,
		CostUSD: info.CostUSD, Summary: summary, Problem: problem,
	})
	if _, err := s.client.AddComment(ctx, token, info.TaskID, comment); err != nil {
		s.record(run, workshop.RunEvent{Kind: "note", Text: "Could not comment on the task: " + err.Error()})
	}
	moved := ""
	if status == "succeeded" && finishColumn != "" {
		moved = s.moveToFinish(ctx, run, token, info, finishColumn)
	}
	s.desk.CancelRun(info.ID)
	// Let the task go, now that the run is over and the task has moved on.
	run.mu.Lock()
	claimID := run.claimID
	if run.stopHeartbeat != nil {
		close(run.stopHeartbeat)
		run.stopHeartbeat = nil
	}
	run.mu.Unlock()
	if claimID != 0 {
		if err := s.client.ReleaseClaim(ctx, token, claimID); err != nil {
			s.record(run, workshop.RunEvent{Kind: "note", Text: "Could not let go of the task: " + err.Error()})
		}
	}

	run.mu.Lock()
	run.info.Status = status
	run.info.State = map[string]string{"succeeded": "done", "stopped": "done"}[status]
	if run.info.State == "" {
		run.info.State = "failed"
	}
	run.info.Diff = diff
	run.info.MovedTo = moved
	run.mu.Unlock()
	s.saveRunFile(run)
	s.record(run, workshop.RunEvent{Kind: "note", Text: "Run " + status + "."})
	s.emitRuns()
	s.Wake()
}

// heartbeatEvery is how often a run renews its claim; a claim lasts two
// minutes.
const heartbeatEvery = 30 * time.Second

// heartbeat keeps the run's claim until the run ends. A claim that lapsed
// anyway (the machine slept) is taken again; if another machine has the
// task by then, the run stops.
func (s *WorkshopService) heartbeat(run *localRun) {
	run.mu.Lock()
	stop := run.stopHeartbeat
	run.mu.Unlock()
	tick := time.NewTicker(heartbeatEvery)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
		}
		run.mu.Lock()
		claimID, taskID, agentID := run.claimID, run.info.TaskID, run.agentID
		run.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		_, err := s.client.HeartbeatClaim(ctx, s.session.Token(), claimID)
		var apiErr *api.Error
		if errors.As(err, &apiErr) && apiErr.Status == 410 {
			c, cerr := s.client.ClaimTask(ctx, s.session.Token(), taskID, agentID, s.machineID)
			if cerr == nil {
				run.mu.Lock()
				run.claimID = c.ID
				run.mu.Unlock()
				s.saveRunFile(run)
			} else {
				s.record(run, workshop.RunEvent{Kind: "note", Text: "Lost the task to another machine; stopping."})
				run.mu.Lock()
				proc := run.proc
				run.mu.Unlock()
				if proc != nil {
					proc.Stop()
				}
			}
		}
		cancel()
	}
}

// machineID names this machine to the API's claims: a random id made once
// and kept in the config folder.
func machineID() string {
	path := filepath.Join(configBase(), "machine")
	if raw, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(raw))) > 0 {
		return strings.TrimSpace(string(raw))
	}
	id := newUUID()
	_ = os.MkdirAll(configBase(), 0o755)
	_ = os.WriteFile(path, []byte(id+"\n"), 0o644)
	return id
}

// moveToFinish moves a finished task to the board's finish column, found by
// name, and returns its name; "" when the board has no such column.
func (s *WorkshopService) moveToFinish(ctx context.Context, run *localRun, token string, info RunInfo, column string) string {
	view, err := s.client.GetBoard(ctx, token, info.BoardID)
	if err != nil {
		s.record(run, workshop.RunEvent{Kind: "note", Text: "Could not move the task: " + err.Error()})
		return ""
	}
	for _, c := range view.Columns {
		if strings.EqualFold(c.Name, column) {
			if _, err := s.client.MoveTask(ctx, token, info.TaskID, c.ID, nil, nil); err != nil {
				s.record(run, workshop.RunEvent{Kind: "note", Text: "Could not move the task: " + err.Error()})
				return ""
			}
			s.record(run, workshop.RunEvent{Kind: "note", Text: "Moved the task to " + c.Name + "."})
			return c.Name
		}
	}
	s.record(run, workshop.RunEvent{Kind: "note", Text: "The board has no column " + column + "; the task stays where it is."})
	return ""
}

// runFile is run.json in a run's folder: enough to finish a run the app did
// not see end (it was closed or crashed during the run).
type runFile struct {
	APIRunID uint64 `json:"api_run_id"`
	BoardID  uint64 `json:"board_id"`
	TaskID   uint64 `json:"task_id"`
	Status   string `json:"status"`
	Repo     string `json:"repo"`
	Worktree string `json:"worktree"`
	Branch   string `json:"branch"`
	// PID is claude's process (and process group) while it runs.
	PID int `json:"pid,omitempty"`
	// ClaimID is the claim holding the task while the run goes.
	ClaimID uint64 `json:"claim_id,omitempty"`
}

func (s *WorkshopService) saveRunFile(run *localRun) {
	run.mu.Lock()
	f := runFile{APIRunID: run.info.APIRunID, BoardID: run.info.BoardID, TaskID: run.info.TaskID, Status: run.info.Status,
		Repo: run.repo, Worktree: run.info.Worktree, Branch: run.info.Branch, ClaimID: run.claimID}
	if run.proc != nil && run.info.Status == "running" {
		f.PID = run.proc.PID()
	}
	dir := run.dir
	run.mu.Unlock()
	raw, _ := json.MarshalIndent(f, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "run.json"), raw, 0o600)
}

// finishOrphans closes runs that were still running when this app last
// went away: they are recorded as failed.
func (s *WorkshopService) finishOrphans(ctx context.Context) {
	files, _ := filepath.Glob(filepath.Join(s.boards.Root(), "*", "runs", "*", "run.json"))
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var f runFile
		if json.Unmarshal(raw, &f) != nil || f.Status != "running" {
			continue
		}
		id := filepath.Base(filepath.Dir(path))
		s.runsMu.Lock()
		_, mine := s.runs[id]
		s.runsMu.Unlock()
		if mine {
			continue
		}
		// Claude may still be working without anyone watching: end it.
		// (Its process group id is its pid; a reused pid would have to
		// belong to a group of that id too.)
		workshop.KillGroup(f.PID)
		if f.ClaimID != 0 {
			_ = s.client.ReleaseClaim(ctx, s.session.Token(), f.ClaimID)
		}
		end := api.RunEnd{Status: "failed", Summary: "Desktop closed during the run."}
		if _, err := s.client.FinishRun(ctx, s.session.Token(), f.APIRunID, end); err != nil {
			// Out of reach or signed out: try again next launch. Refused
			// (already ended, not ours): nothing left to do.
			var apiErr *api.Error
			if errors.Is(err, api.ErrUnauthorized) || !errors.As(err, &apiErr) {
				continue
			}
		}
		f.Status = "failed"
		raw, _ = json.MarshalIndent(f, "", "  ")
		_ = os.WriteFile(path, raw, 0o600)
	}
}

func (s *WorkshopService) logger() *slog.Logger {
	if s.log == nil {
		return slog.Default()
	}
	return s.log
}

// ServiceStartup starts the colony's scheduler and finishes, once the
// member is signed in, the runs the app left running when it last went away.
func (s *WorkshopService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.desk.OnOpen = func(l workshop.Letter) { s.letterChanged(l, +1, eventLetterNew) }
	s.desk.OnClose = func(l workshop.Letter) { s.letterChanged(l, -1, eventLetterClosed) }
	if err := s.desk.Start(); err != nil {
		// Runs still work; they cannot ask, so what needs asking is denied.
		s.logger().Warn("the desk did not start; runs cannot ask for permission", "err", err)
	}
	go s.schedule(ctx)
	go func() {
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()
		for {
			if s.session.Token() != "" {
				s.finishOrphans(ctx)
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
	return nil
}

// RemoveWorktree deletes a finished run's worktree; its branch stays.
func (s *WorkshopService) RemoveWorktree(ctx context.Context, id string) error {
	s.runsMu.Lock()
	run := s.runs[id]
	s.runsMu.Unlock()
	if run == nil {
		return errors.New("no such run on this machine")
	}
	info := run.snapshot()
	if info.Kind == "plan" {
		// A plan works in the repository itself, not a worktree of its own.
		return errors.New("a plan has no worktree to remove")
	}
	s.runsMu.Lock()
	for _, r := range s.runs {
		if o := r.snapshot(); o.Status == "running" && o.Worktree == info.Worktree {
			s.runsMu.Unlock()
			return errors.New("a run is still working in this worktree; stop it first")
		}
	}
	s.runsMu.Unlock()
	run.mu.Lock()
	repo := run.repo
	run.mu.Unlock()
	if err := workshop.RemoveWorktree(ctx, repo, info.Worktree); err != nil {
		return err
	}
	run.mu.Lock()
	run.info.WorktreeRemoved = true
	run.mu.Unlock()
	s.emitRuns()
	return nil
}

// OpenWorktree shows a run's worktree in the file manager.
func (s *WorkshopService) OpenWorktree(id string) error {
	s.runsMu.Lock()
	run := s.runs[id]
	s.runsMu.Unlock()
	if run == nil {
		return errors.New("no such run on this machine")
	}
	return s.app.Browser.OpenFile(run.snapshot().Worktree)
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
	s.desk.CancelRun(id)
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
	// Wait for the runs to be finished: recorded, commented, and their
	// claims released.
	done := make(chan struct{})
	go func() {
		s.ends.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
	}
	s.desk.Stop()
	return nil
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "\n"); i >= 0 {
		s = s[i+1:]
	}
	return s
}

// SetSpeed sets a board's time controls on this machine: paused (no new
// runs start; running ones carry on), normal or fast.
func (s *WorkshopService) SetSpeed(boardID uint64, speed string) (BoardSettings, error) {
	switch speed {
	case workshop.SpeedPaused, workshop.SpeedNormal, workshop.SpeedFast:
	default:
		return BoardSettings{}, errors.New("speed is paused, normal or fast")
	}
	cfg, err := s.boards.Load(boardID)
	if err != nil {
		return BoardSettings{}, err
	}
	cfg.Speed = speed
	if err := s.boards.Save(boardID, cfg); err != nil {
		return BoardSettings{}, err
	}
	s.Wake()
	return s.GetBoardConfig(context.Background(), boardID)
}

// Wails events for letters.
const (
	eventLetterNew    = "letter:new"
	eventLetterClosed = "letter:closed"
)

// deskFor is the desk's MCP entry for a run, or nil when the desk is not
// running.
func (s *WorkshopService) deskFor(runID string) map[string]any {
	cfg := s.desk.ServerConfig(runID)
	if u, _ := cfg["url"].(string); strings.HasPrefix(u, "http://127.0.0.1:") {
		return cfg
	}
	return nil
}

// letterChanged keeps a run's count of open letters and tells the frontend.
func (s *WorkshopService) letterChanged(l workshop.Letter, delta int, event string) {
	s.runsMu.Lock()
	run := s.runs[l.RunID]
	s.runsMu.Unlock()
	if run != nil {
		run.mu.Lock()
		run.info.Waiting = max(0, run.info.Waiting+delta)
		if run.info.Status == "running" {
			if run.info.Waiting > 0 {
				run.info.State = "waiting"
			} else {
				run.info.State = "thinking"
			}
		}
		run.mu.Unlock()
		note := "Waiting for an answer: " + l.ToolName + "."
		if delta < 0 {
			note = "Answered."
		}
		s.record(run, workshop.RunEvent{Kind: "note", Text: note})
		if delta > 0 && s.notify != nil {
			info := run.snapshot()
			title := info.AgentName + " wants to use " + l.ToolName
			if l.Kind == "question" {
				title = info.AgentName + " has a question"
			}
			s.notify(title, info.TaskTitle)
		}
	}
	if s.app != nil {
		s.app.Event.Emit(event, l)
	}
	s.emitRuns()
}

// LetterAnswer is a person's answer to a letter: "allow", "allow_always"
// (allow, and allow the same kind of call on this board from now on),
// "deny" with an optional message, or answers to a question.
type LetterAnswer struct {
	Choice  string            `json:"choice"`
	Message string            `json:"message"`
	Answers map[string]string `json:"answers"`
}

// OpenLetters lists the letters waiting for an answer, oldest first.
func (s *WorkshopService) OpenLetters() []workshop.Letter {
	return s.desk.Open()
}

// AnswerLetter answers a letter; the run carries on with the answer.
func (s *WorkshopService) AnswerLetter(letterID string, a LetterAnswer) error {
	l, ok := s.desk.Get(letterID)
	if !ok {
		return errors.New("this letter was already answered or has expired")
	}
	switch a.Choice {
	case "allow", "deny":
	case "allow_always":
		if err := s.allowAlways(l); err != nil {
			return err
		}
		a.Choice = "allow"
	default:
		return fmt.Errorf("an answer is allow, allow_always or deny, not %q", a.Choice)
	}
	return s.desk.Answer(letterID, workshop.Answer{Behavior: a.Choice, Message: a.Message, Answers: a.Answers})
}

// allowAlways adds the letter's rule to its board's extra allowed tools, so
// later runs there do not ask.
func (s *WorkshopService) allowAlways(l workshop.Letter) error {
	s.runsMu.Lock()
	run := s.runs[l.RunID]
	s.runsMu.Unlock()
	if run == nil {
		return errors.New("the run of this letter is gone")
	}
	boardID := run.snapshot().BoardID
	cfg, err := s.boards.Load(boardID)
	if err != nil {
		return err
	}
	rule := workshop.RuleFor(l)
	if !slices.Contains(cfg.ExtraAllowedTools, rule) {
		cfg.ExtraAllowedTools = append(cfg.ExtraAllowedTools, rule)
		if err := s.boards.Save(boardID, cfg); err != nil {
			return err
		}
	}
	s.record(run, workshop.RunEvent{Kind: "note", Text: "Always allowed on this board: " + rule + "."})
	return nil
}

// eventAgentState tells the colony view what an agent is doing.
const eventAgentState = "agent:state"

type agentState struct {
	Agent string `json:"agent"`
	Run   string `json:"run"`
	Task  uint64 `json:"task"`
	State string `json:"state"`
}
