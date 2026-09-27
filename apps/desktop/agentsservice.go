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
	Seed     string           `json:"portrait_seed"`
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
		sum := AgentSummary{Slug: f.Slug, Name: f.Manifest.Name, Title: f.Manifest.Title, Seed: f.Manifest.PortraitSeed, Skills: []api.AgentSkill{},
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

// AgentDetail is one agent for the character card.
type AgentDetail struct {
	Slug     string          `json:"slug"`
	Manifest agents.Manifest `json:"manifest"`
	Skills   []SkillEntry    `json:"skills"`
	// AgentID is 0 until the agent's first sync.
	AgentID    uint64   `json:"agent_id"`
	Revision   int      `json:"revision"`
	SharedWith []uint64 `json:"shared_with"`
	// Recruited copies: whose original, and whether it has changed since.
	Recruited    bool   `json:"recruited"`
	OriginNewer  bool   `json:"origin_newer"`
	OriginGone   bool   `json:"origin_gone"`
	OwnerName    string `json:"owner_name"`
	Problem      string `json:"problem"`
	Dir          string `json:"dir"`
	Unsynced     bool   `json:"unsynced"`
	ConflictOpen bool   `json:"conflict"`
}

// SkillEntry is one skill of an agent or of ~/.claude/skills.
type SkillEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Dir         string `json:"dir,omitempty"`
}

func skillsOf(f agents.Folder) []SkillEntry {
	out := []SkillEntry{}
	for path, content := range f.Files {
		dir, file := filepath.Split(path)
		if file == "SKILL.md" && filepath.Dir(filepath.Clean(dir)) == "." {
			if name, desc, ok := agents.SkillOf(content); ok {
				out = append(out, SkillEntry{Name: name, Description: desc})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Get reads one agent for the card, with its sharing and origin from the API
// when it has been synced.
func (s *AgentsService) Get(ctx context.Context, slug string) (AgentDetail, error) {
	f, err := s.engine.Folder(slug)
	if err != nil {
		return AgentDetail{}, err
	}
	d := AgentDetail{Slug: slug, Manifest: f.Manifest, Skills: skillsOf(f), SharedWith: []uint64{}, Dir: f.Dir, Unsynced: f.Changed()}
	if err := agents.Check(f); err != nil {
		d.Problem = err.Error()
	}
	conflicts, _ := s.engine.Conflicts()
	for _, c := range conflicts {
		d.ConflictOpen = d.ConflictOpen || c.Slug == slug
	}
	if f.Sync == nil {
		return d, nil
	}
	d.AgentID, d.Revision = f.Sync.AgentID, f.Sync.Revision
	a, shared, err := callPair(s.session, func(t string) (api.Agent, []uint64, error) { return s.client.GetAgent(ctx, t, f.Sync.AgentID) })
	if err != nil {
		// Offline or not synced yet: the folder is still worth showing.
		return d, nil
	}
	if shared != nil {
		d.SharedWith = shared
	}
	if a.OriginAgentID != nil {
		d.Recruited = true
		if o, err := call(s.session, func(t string) (api.Origin, error) { return s.client.AgentOrigin(ctx, t, a.ID) }); err == nil {
			d.OriginNewer, d.OriginGone = o.Newer, o.Gone
		}
	}
	return d, nil
}

func callPair[A, B any](sess *session.Session, f func(token string) (A, B, error)) (A, B, error) {
	var b B
	a, err := call(sess, func(t string) (A, error) {
		var x A
		var err error
		x, b, err = f(t)
		return x, err
	})
	return a, b, err
}

// Create makes a new agent's folder; it is sent on the next sync.
func (s *AgentsService) Create(m agents.Manifest) (string, error) {
	slug, err := s.engine.NewAgent(m)
	if err == nil {
		s.Trigger()
	}
	return slug, err
}

// Save writes the card's fields into agent.toml; the sync sends them.
func (s *AgentsService) Save(slug string, m agents.Manifest) error {
	err := s.engine.SaveManifest(slug, m)
	if err == nil {
		s.Trigger()
	}
	return err
}

// Delete deletes the agent on the server and moves its folder to .trash.
func (s *AgentsService) Delete(ctx context.Context, slug string) error {
	m := s.session.Member()
	if m == nil {
		return ErrSignedOut
	}
	if err := s.engine.Delete(ctx, m.ID, slug); err != nil {
		return err
	}
	if s.app != nil {
		s.app.Event.Emit(eventAgentsChanged, nil)
	}
	return nil
}

// Traits lists the traits and the permission modes, from the API.
func (s *AgentsService) Traits(ctx context.Context) (TraitList, error) {
	var modes []string
	traits, err := call(s.session, func(t string) ([]api.Trait, error) {
		tr, m, err := s.client.Traits(ctx, t)
		modes = m
		return tr, err
	})
	return TraitList{Traits: traits, PermissionModes: modes}, err
}

type TraitList struct {
	Traits          []api.Trait `json:"traits"`
	PermissionModes []string    `json:"permission_modes"`
}

// ClaudeSkills lists the skills in ~/.claude/skills, to import from.
func (s *AgentsService) ClaudeSkills() ([]SkillEntry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(home, ".claude", "skills")
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []SkillEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []SkillEntry{}
	for _, e := range entries {
		dir := filepath.Join(root, e.Name())
		b, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		if err != nil {
			continue
		}
		name, desc, ok := agents.SkillOf(string(b))
		if !ok {
			continue
		}
		out = append(out, SkillEntry{Name: name, Description: desc, Dir: dir})
	}
	return out, nil
}

// ImportSkills copies skills (by their directory) into the agent. It stops
// at the first one that breaks a rule and says which.
func (s *AgentsService) ImportSkills(slug string, dirs []string) error {
	for _, dir := range dirs {
		if _, err := s.engine.AddSkill(slug, dir); err != nil {
			return err
		}
	}
	s.Trigger()
	return nil
}

// ImportSkillFolder asks for a skill folder and copies it into the agent.
// It returns "" when the member cancels.
func (s *AgentsService) ImportSkillFolder(slug string) (string, error) {
	dir, err := s.app.Dialog.OpenFile().SetTitle("Choose a skill folder (with a SKILL.md)").
		CanChooseDirectories(true).CanChooseFiles(false).PromptForSingleSelection()
	if err != nil || dir == "" {
		return "", err
	}
	name, err := s.engine.AddSkill(slug, dir)
	if err == nil {
		s.Trigger()
	}
	return name, err
}

// RemoveSkill deletes one of the agent's skills.
func (s *AgentsService) RemoveSkill(slug, skill string) error {
	err := s.engine.RemoveSkill(slug, skill)
	if err == nil {
		s.Trigger()
	}
	return err
}

// OpenFolder shows the agent's folder in the file manager.
func (s *AgentsService) OpenFolder(slug string) error {
	f, err := s.engine.Folder(slug)
	if err != nil {
		return err
	}
	return s.app.Browser.OpenFile(f.Dir)
}

// Share shares (on) or unshares a synced agent with a guild.
func (s *AgentsService) Share(ctx context.Context, slug string, guildID uint64, on bool) error {
	f, err := s.engine.Folder(slug)
	if err != nil {
		return err
	}
	if f.Sync == nil {
		return errors.New("this agent has not synced yet; share it once it has")
	}
	_, err = call(s.session, func(t string) (struct{}, error) {
		return struct{}{}, s.client.ShareAgent(ctx, t, f.Sync.AgentID, guildID, on)
	})
	return err
}

// GuildAgents lists the agents shared with a guild, to recruit from.
func (s *AgentsService) GuildAgents(ctx context.Context, guildID uint64) ([]api.Agent, error) {
	as, err := call(s.session, func(t string) ([]api.Agent, error) { return s.client.GuildAgents(ctx, t, guildID) })
	if as == nil {
		as = []api.Agent{}
	}
	return as, err
}

// Recruit copies a shared agent into my roster; it arrives with the sync
// that follows.
func (s *AgentsService) Recruit(ctx context.Context, agentID, guildID uint64) error {
	_, err := call(s.session, func(t string) (api.Agent, error) { return s.client.RecruitAgent(ctx, t, agentID, guildID) })
	if err == nil {
		s.Trigger()
	}
	return err
}

// PullOrigin updates a recruited copy from its original; the folder
// follows with the next sync.
func (s *AgentsService) PullOrigin(ctx context.Context, slug string) error {
	f, err := s.engine.Folder(slug)
	if err != nil {
		return err
	}
	if f.Sync == nil {
		return errors.New("this agent has not synced yet")
	}
	if f.Changed() {
		return errors.New("this agent has changes that have not synced yet; wait a moment and try again")
	}
	_, err = call(s.session, func(t string) (api.Agent, error) {
		return s.client.PullOrigin(ctx, t, f.Sync.AgentID, f.Sync.Revision)
	})
	if err == nil {
		s.Trigger()
	}
	return err
}
