package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jevido/the-bakery/apps/desktop/internal/workshop"
)

// supervisorTimeout bounds one supervisor answer.
const supervisorTimeout = 2 * time.Minute

// ChatEntry is one message in a board's supervisor chat.
type ChatEntry struct {
	ID string `json:"id"`
	// From is "member" or "supervisor".
	From      string         `json:"from"`
	Text      string         `json:"text"`
	At        time.Time      `json:"at"`
	CostUSD   float64        `json:"cost_usd,omitempty"`
	Proposals []ChatProposal `json:"proposals,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// ChatProposal is a proposal in the chat, with what became of it: "" while
// it waits for the member, else what approving or declining did.
type ChatProposal struct {
	workshop.SupervisorProposal
	Summary string `json:"summary"`
	Outcome string `json:"outcome"`
}

// SupervisorService is each board's supervisor chat on this machine. It
// sends the member's message and a fresh snapshot of the board to Claude
// (no tools, no repository) and keeps the conversation in the board's
// config folder. It changes nothing on the board: approving a proposal
// goes through the same services as the member's own clicks.
type SupervisorService struct {
	work     *WorkshopService
	settings *SettingsService
	mu       sync.Mutex
	thinking map[uint64]bool
}

func NewSupervisorService(work *WorkshopService, settings *SettingsService) *SupervisorService {
	return &SupervisorService{work: work, settings: settings, thinking: map[uint64]bool{}}
}

func (s *SupervisorService) historyPath(boardID uint64) string {
	return filepath.Join(s.work.boards.Dir(boardID), "supervisor.jsonl")
}

func (s *SupervisorService) sessionPath(boardID uint64) string {
	return filepath.Join(s.work.boards.Dir(boardID), "supervisor-session")
}

// History is the board's chat, oldest first.
func (s *SupervisorService) History(boardID uint64) ([]ChatEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read(boardID)
}

func (s *SupervisorService) read(boardID uint64) ([]ChatEntry, error) {
	f, err := os.Open(s.historyPath(boardID))
	if errors.Is(err, os.ErrNotExist) {
		return []ChatEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := []ChatEntry{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var e ChatEntry
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			out = append(out, e)
		}
	}
	return out, sc.Err()
}

func (s *SupervisorService) write(boardID uint64, entries []ChatEntry) error {
	var buf bytes.Buffer
	for _, e := range entries {
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		buf.Write(raw)
		buf.WriteByte('\n')
	}
	dir := s.work.boards.Dir(boardID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp := s.historyPath(boardID) + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.historyPath(boardID))
}

// Forget clears the board's conversation: the history and the session.
func (s *SupervisorService) Forget(boardID uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range []string{s.historyPath(boardID), s.sessionPath(boardID)} {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// Send asks the supervisor and returns its answer, already in the
// history. The member's message is kept even when the answer fails.
func (s *SupervisorService) Send(ctx context.Context, boardID uint64, text string) (ChatEntry, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return ChatEntry{}, errors.New("say something to the supervisor first")
	}
	if len(text) > 4000 {
		return ChatEntry{}, errors.New("keep a message under 4000 characters")
	}
	s.mu.Lock()
	if s.thinking[boardID] {
		s.mu.Unlock()
		return ChatEntry{}, errors.New("the supervisor is still answering the last message")
	}
	s.thinking[boardID] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.thinking, boardID)
		s.mu.Unlock()
	}()

	snap, err := s.snapshot(ctx, boardID)
	if err != nil {
		return ChatEntry{}, err
	}
	s.mu.Lock()
	history, err := s.read(boardID)
	if err == nil {
		history = append(history, ChatEntry{ID: newUUID(), From: "member", Text: text, At: time.Now()})
		err = s.write(boardID, history)
	}
	s.mu.Unlock()
	if err != nil {
		return ChatEntry{}, err
	}

	answer := s.ask(ctx, boardID, snap, withOutcomes(history, text))
	s.mu.Lock()
	defer s.mu.Unlock()
	history, err = s.read(boardID)
	if err != nil {
		return ChatEntry{}, err
	}
	history = append(history, answer)
	return answer, s.write(boardID, history)
}

// withOutcomes tells the supervisor what became of its last proposals, so
// it does not propose them again.
func withOutcomes(history []ChatEntry, text string) string {
	for i := len(history) - 1; i >= 0; i-- {
		e := history[i]
		if e.From != "supervisor" || len(e.Proposals) == 0 {
			continue
		}
		var lines []string
		for _, p := range e.Proposals {
			outcome := p.Outcome
			if outcome == "" {
				outcome = "not decided yet"
			}
			lines = append(lines, "- "+p.Summary+": "+outcome)
		}
		return "What became of your last proposals:\n" + strings.Join(lines, "\n") + "\n\n" + text
	}
	return text
}

// ask runs Claude once and turns the result into the supervisor's entry;
// a failure becomes an entry with the error.
func (s *SupervisorService) ask(ctx context.Context, boardID uint64, snap workshop.SupervisorSnapshot, text string) ChatEntry {
	entry := ChatEntry{ID: newUUID(), From: "supervisor", At: time.Now()}
	settings, _ := s.settings.Get()
	session, resume := s.session(boardID)
	spec, err := workshop.BuildSupervisorSpec(workshop.SupervisorInput{
		Snapshot: snap, Message: text, Model: settings.SupervisorModel, Session: session, Resume: resume, Dir: s.work.boards.Dir(boardID),
	})
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	if _, err := exec.LookPath(s.work.claudeBinary()); err != nil {
		entry.Error = "claude is not installed or not on the PATH of this app"
		return entry
	}
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), supervisorTimeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, s.work.claudeBinary(), spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Stdin = strings.NewReader(spec.Stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	answer, perr := workshop.ParseSupervisorOutput(stdout.Bytes(), snap)
	switch {
	case perr != nil:
		entry.Error = strings.TrimSpace(lastLine(stderr.String()))
		if entry.Error == "" && runErr != nil {
			entry.Error = runErr.Error()
		}
		if entry.Error == "" {
			entry.Error = perr.Error()
		}
		if cctx.Err() != nil {
			entry.Error = "the supervisor took longer than two minutes; try again"
		}
		return entry
	case answer.Error != "":
		entry.Error, entry.CostUSD = answer.Error, answer.CostUSD
		return entry
	}
	// The session exists now; later messages carry it on.
	_ = os.WriteFile(s.sessionPath(boardID), []byte(session+"\nresume\n"), 0o600)
	entry.Text, entry.CostUSD = answer.Reply, answer.CostUSD
	for _, p := range answer.Proposals {
		entry.Proposals = append(entry.Proposals, ChatProposal{SupervisorProposal: p, Summary: p.Describe(snap)})
	}
	return entry
}

// session is the board's conversation id, and whether Claude already has
// it (then it is resumed).
func (s *SupervisorService) session(boardID uint64) (string, bool) {
	raw, err := os.ReadFile(s.sessionPath(boardID))
	if err == nil {
		lines := strings.Fields(string(raw))
		if len(lines) > 0 {
			return lines[0], len(lines) > 1 && lines[1] == "resume"
		}
	}
	id := newUUID()
	_ = os.MkdirAll(s.work.boards.Dir(boardID), 0o700)
	_ = os.WriteFile(s.sessionPath(boardID), []byte(id+"\n"), 0o600)
	return id, false
}

// snapshot is the board as the supervisor sees it right now.
func (s *SupervisorService) snapshot(ctx context.Context, boardID uint64) (workshop.SupervisorSnapshot, error) {
	token := s.work.session.Token()
	if token == "" {
		return workshop.SupervisorSnapshot{}, ErrSignedOut
	}
	view, err := s.work.client.GetBoard(ctx, token, boardID)
	if err != nil {
		return workshop.SupervisorSnapshot{}, err
	}
	cfg, err := s.work.boards.Load(boardID)
	if err != nil {
		return workshop.SupervisorSnapshot{}, err
	}
	snap := workshop.SupervisorSnapshot{Board: view.Board.Name, ReadyColumn: cfg.ReadyColumn}
	if wts, err := s.work.client.ListWorkTypes(ctx, token, view.Board.GuildID); err == nil {
		for _, w := range wts {
			snap.WorkTypes = append(snap.WorkTypes, w.Key)
		}
	}
	busy := map[string]bool{}
	for _, r := range s.work.Runs() {
		if r.Status == "running" {
			busy[r.AgentSlug] = true
		}
	}
	slugOf := map[uint64]string{}
	for _, slug := range cfg.Agents {
		f, err := s.work.agents.Folder(slug)
		if err != nil {
			continue
		}
		if f.Sync != nil {
			slugOf[f.Sync.AgentID] = slug
		}
		snap.Agents = append(snap.Agents, workshop.SupervisorAgent{
			Slug: slug, Name: f.Manifest.Name, WorkPriorities: f.Manifest.WorkPriorities, Busy: busy[slug],
		})
	}
	for _, c := range view.Columns {
		col := workshop.SupervisorColumn{Name: c.Name, Tasks: []workshop.SupervisorTask{}}
		for _, t := range c.Tasks {
			st := workshop.SupervisorTask{ID: t.ID, Title: t.Title, Description: short(t.Description, 400),
				SubtasksDone: t.SubtasksDone, SubtasksTotal: t.SubtasksTotal, Claimed: t.Claim != nil, Forbidden: t.Forbidden}
			if t.WorkType != nil {
				st.WorkType = *t.WorkType
			}
			if t.PrioritizedAgentID != nil {
				st.PrioritizedFor = slugOf[*t.PrioritizedAgentID]
				if st.PrioritizedFor == "" {
					st.PrioritizedFor = fmt.Sprintf("another member's agent %d", *t.PrioritizedAgentID)
				}
			}
			col.Tasks = append(col.Tasks, st)
		}
		snap.Columns = append(snap.Columns, col)
	}
	return snap, nil
}

// Thinking reports whether the supervisor is answering on the board now.
func (s *SupervisorService) Thinking(boardID uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.thinking[boardID]
}

// SetOutcome records what became of a proposal (approved or declined).
func (s *SupervisorService) SetOutcome(boardID uint64, entryID string, index int, outcome string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	history, err := s.read(boardID)
	if err != nil {
		return err
	}
	for i := range history {
		if history[i].ID == entryID && index >= 0 && index < len(history[i].Proposals) {
			history[i].Proposals[index].Outcome = outcome
			return s.write(boardID, history)
		}
	}
	return errors.New("that proposal is not in the chat any more")
}

// short keeps a description to about n characters for the snapshot.
func short(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
