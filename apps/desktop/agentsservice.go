package main

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jevido/the-bakery/apps/desktop/internal/agents"
	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/session"
)

// Events AgentsService sends the frontend.
const (
	// eventAgentsChanged says the roster or a folder changed: reload.
	eventAgentsChanged = "agents:changed"
	// eventAgentsStatus carries an AgentSyncStatus.
	eventAgentsStatus = "agents:status"
)

// periodicSync is the fallback trigger when nothing else fires.
const periodicSync = 5 * time.Minute

// AgentSyncStatus is the sync's state, for the sidebar.
type AgentSyncStatus struct {
	// State is "idle", "syncing", "offline", "error" or "signed-out".
	State     string     `json:"state"`
	LastSync  *time.Time `json:"last_sync"`
	Conflicts int        `json:"conflicts"`
	Error     string     `json:"error,omitempty"`
	// Problems are folders that were not sent because they break a rule.
	Problems []string `json:"problems"`
}

// AgentsService keeps the member's agent folders in step with the API on
// its own: at start, when a folder changes, when the window gets focus and
// every five minutes. One sync runs at a time; a trigger during a sync
// queues exactly one more.
type AgentsService struct {
	client  *api.Client
	session *session.Session
	logger  *slog.Logger
	app     *application.App
	engine  *agents.Engine

	trigger chan struct{}
	mu      sync.Mutex
	status  AgentSyncStatus
	backoff time.Duration
	cancel  context.CancelFunc
}

// agentsBase is where agent folders live: ~/.config/the-bakery on Linux.
func agentsBase() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "the-bakery")
}

func NewAgentsService(client *api.Client, s *session.Session, logger *slog.Logger) *AgentsService {
	svc := &AgentsService{client: client, session: s, logger: logger, trigger: make(chan struct{}, 1)}
	host := client.BaseURL()
	if u, err := url.Parse(host); err == nil && u.Host != "" {
		host = u.Host
	}
	svc.engine = agents.NewEngine(agentsBase(), host, remote{svc})
	svc.status = AgentSyncStatus{State: "idle", Problems: []string{}}
	return svc
}

// remote is the engine's view of the API, signed in as the member.
type remote struct{ s *AgentsService }

func (r remote) List(ctx context.Context, since *time.Time) ([]api.Agent, error) {
	return call(r.s.session, func(t string) ([]api.Agent, error) { return r.s.client.ListAgents(ctx, t, since) })
}

func (r remote) Get(ctx context.Context, id uint64) (api.Agent, error) {
	return call(r.s.session, func(t string) (api.Agent, error) {
		a, _, err := r.s.client.GetAgent(ctx, t, id)
		return a, err
	})
}

func (r remote) Create(ctx context.Context, a api.AgentWrite) (api.Agent, error) {
	return call(r.s.session, func(t string) (api.Agent, error) { return r.s.client.CreateAgent(ctx, t, a) })
}

func (r remote) Revise(ctx context.Context, id uint64, basedOn int, a api.AgentWrite) (api.Agent, error) {
	return call(r.s.session, func(t string) (api.Agent, error) { return r.s.client.ReviseAgent(ctx, t, id, basedOn, a) })
}

func (r remote) Delete(ctx context.Context, id uint64, basedOn int) error {
	_, err := call(r.s.session, func(t string) (struct{}, error) {
		return struct{}{}, r.s.client.DeleteAgent(ctx, t, id, basedOn)
	})
	return err
}

func (s *AgentsService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	ctx, s.cancel = context.WithCancel(context.Background())
	if err := agents.Watch(ctx, s.engine.Dir(), func(string) { s.Trigger() }); err != nil {
		s.logger.Warn("watching agent folders failed; syncing on focus and every 5 minutes only", "err", err)
	}
	go s.loop(ctx)
	s.Trigger()
	return nil
}

func (s *AgentsService) ServiceShutdown() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

// Trigger asks for a sync soon; several triggers at once make one sync.
func (s *AgentsService) Trigger() {
	select {
	case s.trigger <- struct{}{}:
	default:
	}
}

func (s *AgentsService) loop(ctx context.Context) {
	tick := time.NewTicker(periodicSync)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		case <-s.trigger:
		}
		s.syncOnce(ctx)
	}
}

func (s *AgentsService) setStatus(change func(st *AgentSyncStatus)) {
	s.mu.Lock()
	change(&s.status)
	st := s.status
	s.mu.Unlock()
	if s.app != nil {
		s.app.Event.Emit(eventAgentsStatus, st)
	}
}

func (s *AgentsService) syncOnce(ctx context.Context) {
	m := s.session.Member()
	if m == nil {
		s.setStatus(func(st *AgentSyncStatus) { st.State = "signed-out" })
		return
	}
	s.setStatus(func(st *AgentSyncStatus) { st.State = "syncing" })
	rep, err := s.engine.Sync(ctx, m.ID)
	var unreachable *api.Unreachable
	switch {
	case errors.As(err, &unreachable):
		// Local edits stay local until the API is back.
		s.mu.Lock()
		s.backoff = min(max(2*s.backoff, 5*time.Second), 2*time.Minute)
		wait := s.backoff
		s.mu.Unlock()
		s.setStatus(func(st *AgentSyncStatus) { st.State = "offline"; st.Error = "" })
		time.AfterFunc(wait, s.Trigger)
		return
	case errors.Is(err, ErrSignedOut):
		s.setStatus(func(st *AgentSyncStatus) { st.State = "signed-out" })
		return
	case err != nil:
		s.logger.Error("agent sync failed", "err", err)
		s.setStatus(func(st *AgentSyncStatus) { st.State = "error"; st.Error = err.Error() })
		return
	}
	s.mu.Lock()
	s.backoff = 0
	s.mu.Unlock()
	problems := []string{}
	for _, p := range rep.Problems {
		problems = append(problems, p.Error())
	}
	finished := rep.FinishedAt
	s.setStatus(func(st *AgentSyncStatus) {
		st.State, st.Error, st.LastSync, st.Conflicts, st.Problems = "idle", "", &finished, len(rep.Conflicts), problems
	})
	if !rep.UpToDate && s.app != nil {
		s.app.Event.Emit(eventAgentsChanged, nil)
	}
}

// SyncNow runs a sync soon (the app calls it after signing in, too).
func (s *AgentsService) SyncNow() { s.Trigger() }

// Status is the sync's state now.
func (s *AgentsService) Status() AgentSyncStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// AgentSummary is one agent of the roster as its folder holds it.
type AgentSummary struct {
	Slug     string           `json:"slug"`
	Name     string           `json:"name"`
	Title    string           `json:"title"`
	Skills   []api.AgentSkill `json:"skills"`
	Conflict bool             `json:"conflict"`
	Synced   bool             `json:"synced"`
}

// List reads the roster from the agent folders.
func (s *AgentsService) List() ([]AgentSummary, error) {
	entries, err := os.ReadDir(s.engine.Dir())
	if errors.Is(err, os.ErrNotExist) {
		return []AgentSummary{}, nil
	}
	if err != nil {
		return nil, err
	}
	conflicts, _ := s.engine.Conflicts()
	inConflict := map[string]bool{}
	for _, c := range conflicts {
		inConflict[c.Slug] = true
	}
	out := []AgentSummary{}
	for _, e := range entries {
		if !e.IsDir() || e.Name()[0] == '.' {
			continue
		}
		f, err := agents.ReadFolder(filepath.Join(s.engine.Dir(), e.Name()))
		if err != nil {
			return nil, err
		}
		sum := AgentSummary{Slug: f.Slug, Name: f.Manifest.Name, Title: f.Manifest.Title, Skills: []api.AgentSkill{},
			Conflict: inConflict[f.Slug], Synced: f.Sync != nil && !f.Changed()}
		for path, content := range f.Files {
			if filepath.Base(path) == "SKILL.md" && filepath.Dir(path) != "." && filepath.Dir(filepath.Dir(path)) == "." {
				if name, desc, ok := agents.SkillOf(content); ok {
					sum.Skills = append(sum.Skills, api.AgentSkill{Name: name, Description: desc})
				}
			}
		}
		sort.Slice(sum.Skills, func(i, j int) bool { return sum.Skills[i].Name < sum.Skills[j].Name })
		out = append(out, sum)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// AgentConflict is a conflict with the files that differ.
type AgentConflict struct {
	Slug  string            `json:"slug"`
	Files []agents.FileDiff `json:"files"`
}

// Conflicts lists the agents changed both here and on the server, with the
// files that differ.
func (s *AgentsService) Conflicts() ([]AgentConflict, error) {
	cs, err := s.engine.Conflicts()
	if err != nil {
		return nil, err
	}
	out := []AgentConflict{}
	for _, c := range cs {
		files, err := s.engine.Diff(c.Slug)
		if err != nil {
			return nil, err
		}
		out = append(out, AgentConflict{Slug: c.Slug, Files: files})
	}
	return out, nil
}

// ResolveConflict settles a conflict: "local" sends this device's version,
// "server" takes the server's, "both" keeps the server's in place and adds
// this device's as a new agent <slug>-local.
func (s *AgentsService) ResolveConflict(ctx context.Context, slug, choice string) error {
	m := s.session.Member()
	if m == nil {
		return ErrSignedOut
	}
	if err := s.engine.Resolve(ctx, m.ID, slug, agents.Choice(choice)); err != nil {
		return err
	}
	s.Trigger()
	if s.app != nil {
		s.app.Event.Emit(eventAgentsChanged, nil)
	}
	return nil
}

// Dir is the folder the agents are mirrored in.
func (s *AgentsService) Dir() string { return s.engine.Dir() }
