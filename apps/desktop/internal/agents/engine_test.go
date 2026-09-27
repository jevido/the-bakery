package agents

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
)

// fakeRemote is the API in memory, with its revision rule.
type fakeRemote struct {
	agents  map[uint64]api.Agent
	next    uint64
	creates int
	revises int
	clock   time.Time
}

func newFake() *fakeRemote {
	return &fakeRemote{agents: map[uint64]api.Agent{}, next: 1, clock: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)}
}

func (f *fakeRemote) tick() time.Time { f.clock = f.clock.Add(time.Minute); return f.clock }

func (f *fakeRemote) List(_ context.Context, since *time.Time) ([]api.Agent, error) {
	var out []api.Agent
	for _, a := range f.agents {
		if since == nil && a.Deleted {
			continue
		}
		if since == nil || a.UpdatedAt.After(*since) {
			a.Files = nil
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeRemote) Get(_ context.Context, id uint64) (api.Agent, error) {
	a, ok := f.agents[id]
	if !ok || a.Deleted {
		return api.Agent{}, &api.Error{Status: http.StatusNotFound, Message: "agent not found"}
	}
	return a, nil
}

func fromWrite(w api.AgentWrite) api.Agent {
	return api.Agent{
		Slug: w.Slug, Name: w.Name, Title: w.Title, Backstory: w.Backstory, Traits: w.Traits, Model: w.Model,
		PermissionMode: w.PermissionMode, AllowedTools: w.AllowedTools, PortraitSeed: w.PortraitSeed,
		WorkPriorities: w.WorkPriorities, Files: w.Files,
	}
}

func (f *fakeRemote) Create(_ context.Context, w api.AgentWrite) (api.Agent, error) {
	f.creates++
	a := fromWrite(w)
	a.ID, a.Revision, a.UpdatedAt = f.next, 1, f.tick()
	f.next++
	f.agents[a.ID] = a
	return a, nil
}

func (f *fakeRemote) Revise(_ context.Context, id uint64, basedOn int, w api.AgentWrite) (api.Agent, error) {
	f.revises++
	cur, ok := f.agents[id]
	if !ok || cur.Deleted {
		return api.Agent{}, &api.Error{Status: http.StatusNotFound}
	}
	if cur.Revision != basedOn {
		return api.Agent{}, &api.StaleError{Current: cur}
	}
	a := fromWrite(w)
	a.ID, a.Slug, a.Revision, a.UpdatedAt = id, cur.Slug, cur.Revision+1, f.tick()
	f.agents[id] = a
	return a, nil
}

func (f *fakeRemote) Delete(_ context.Context, id uint64, basedOn int) error {
	cur := f.agents[id]
	if cur.Revision != basedOn {
		return &api.StaleError{Current: cur}
	}
	cur.Deleted, cur.Revision, cur.UpdatedAt = true, cur.Revision+1, f.tick()
	f.agents[id] = cur
	return nil
}

// serverEdit changes an agent on the server, as another device would.
func (f *fakeRemote) serverEdit(id uint64, change func(a *api.Agent)) {
	a := f.agents[id]
	change(&a)
	a.Revision++
	a.UpdatedAt = f.tick()
	f.agents[id] = a
}

const skill = "---\nname: go-tests\ndescription: Go tests.\n---\n\nBody.\n"

func (f *fakeRemote) seed(slug, name string) uint64 {
	a, _ := f.Create(context.Background(), api.AgentWrite{
		Slug: slug, Name: name, Model: "sonnet", PermissionMode: "manual",
		WorkPriorities: map[string]int{"coding": 1},
		Files:          []api.AgentFile{{Path: "go-tests/SKILL.md", Content: skill}},
	})
	return a.ID
}

func setup(t *testing.T) (*Engine, *fakeRemote) {
	t.Helper()
	remote := newFake()
	e := NewEngine(t.TempDir(), "api.test", remote)
	e.now = func() time.Time { return remote.clock }
	return e, remote
}

func mustSync(t *testing.T, e *Engine) Report {
	t.Helper()
	rep, err := e.Sync(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFirstSyncDownloads(t *testing.T) {
	e, remote := setup(t)
	remote.seed("vera", "Vera")
	rep := mustSync(t, e)
	if len(rep.Pulled) != 1 {
		t.Fatalf("report = %+v", rep)
	}
	if got := read(t, filepath.Join(e.Dir(), "vera/skills/go-tests/SKILL.md")); got != skill {
		t.Errorf("SKILL.md = %q", got)
	}
	if got := read(t, filepath.Join(e.Dir(), "vera/agent.toml")); !contains(got, `name = 'Vera'`) && !contains(got, `name = "Vera"`) {
		t.Errorf("agent.toml = %s", got)
	}
	if rep := mustSync(t, e); !rep.UpToDate {
		t.Errorf("second sync did something: %+v", rep)
	}
}

func TestLocalChangeIsPushed(t *testing.T) {
	e, remote := setup(t)
	id := remote.seed("vera", "Vera")
	mustSync(t, e)
	write(t, filepath.Join(e.Dir(), "vera/skills/go-tests/SKILL.md"), skill+"\nMore.\n")
	rep := mustSync(t, e)
	if len(rep.Pushed) != 1 || remote.agents[id].Revision != 2 {
		t.Fatalf("report = %+v, revision %d", rep, remote.agents[id].Revision)
	}
	if remote.agents[id].Files[0].Content != skill+"\nMore.\n" {
		t.Error("server did not get the edit")
	}
	if rep := mustSync(t, e); !rep.UpToDate {
		t.Errorf("after a push the folder should be in step: %+v", rep)
	}
}

func TestRemoteChangeIsPulled(t *testing.T) {
	e, remote := setup(t)
	id := remote.seed("vera", "Vera")
	mustSync(t, e)
	remote.serverEdit(id, func(a *api.Agent) { a.Title = "Staff engineer" })
	rep := mustSync(t, e)
	if len(rep.Pulled) != 1 || !contains(read(t, filepath.Join(e.Dir(), "vera/agent.toml")), "Staff engineer") {
		t.Fatalf("report = %+v", rep)
	}
	if remote.revises != 0 {
		t.Error("pulling sent something back")
	}
}

func TestBothChangedIsAConflict(t *testing.T) {
	e, remote := setup(t)
	id := remote.seed("vera", "Vera")
	mustSync(t, e)
	write(t, filepath.Join(e.Dir(), "vera/skills/go-tests/SKILL.md"), skill+"\nLocal.\n")
	remote.serverEdit(id, func(a *api.Agent) {
		a.Files = []api.AgentFile{{Path: "go-tests/SKILL.md", Content: skill + "\nServer.\n"}}
	})
	rep := mustSync(t, e)
	if len(rep.Conflicts) != 1 || rep.Conflicts[0].Slug != "vera" {
		t.Fatalf("report = %+v", rep)
	}
	if !contains(read(t, filepath.Join(e.Dir(), "vera/skills/go-tests/SKILL.md")), "Local.") {
		t.Error("the local version must stay in place")
	}
	if !contains(read(t, filepath.Join(e.Dir(), ".conflicts/vera/skills/go-tests/SKILL.md")), "Server.") {
		t.Error("the server version must be kept aside")
	}
	if remote.revises != 0 {
		t.Error("a conflict must never overwrite the server")
	}
	// It stays a conflict, untouched, until resolved.
	mustSync(t, e)
	if remote.revises != 0 {
		t.Error("a later sync pushed a conflicted agent")
	}
}

func TestStaleRevisionIsAConflict(t *testing.T) {
	// The server moved on between listing and sending: the 409 path.
	e, remote := setup(t)
	id := remote.seed("vera", "Vera")
	mustSync(t, e)
	write(t, filepath.Join(e.Dir(), "vera/skills/go-tests/SKILL.md"), skill+"\nLocal.\n")
	stale := remote.agents[id]
	stale.Revision = 5
	remote.agents[id] = stale // a newer revision the list missed (same UpdatedAt)
	rep := mustSync(t, e)
	if len(rep.Conflicts) != 1 {
		t.Fatalf("report = %+v", rep)
	}
}

func TestNewFolderIsCreated(t *testing.T) {
	e, remote := setup(t)
	mustSync(t, e)
	write(t, filepath.Join(e.Dir(), "ivo/agent.toml"), "name = 'Ivo'\ntitle = 'Researcher'\n")
	write(t, filepath.Join(e.Dir(), "ivo/skills/go-tests/SKILL.md"), skill)
	rep := mustSync(t, e)
	if len(rep.Created) != 1 || remote.creates != 1 {
		t.Fatalf("report = %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(e.Dir(), "ivo", syncName)); err != nil {
		t.Error("a created agent gets its .sync.json")
	}
	if rep := mustSync(t, e); !rep.UpToDate || remote.creates != 1 {
		t.Errorf("created twice: %+v", rep)
	}
}

func TestInvalidFolderIsNotSent(t *testing.T) {
	e, remote := setup(t)
	mustSync(t, e)
	write(t, filepath.Join(e.Dir(), "ivo/agent.toml"), "name = 'Ivo'\n")
	write(t, filepath.Join(e.Dir(), "ivo/skills/go-tests/notes.md"), "no SKILL.md here")
	rep := mustSync(t, e)
	var p *Problem
	if len(rep.Problems) != 1 || !errors.As(rep.Problems[0], &p) || p.Path != "ivo/skills/go-tests" {
		t.Fatalf("problems = %v", rep.Problems)
	}
	if remote.creates != 0 {
		t.Error("an invalid folder was sent")
	}
}

func TestServerDeleteMovesToTrash(t *testing.T) {
	e, remote := setup(t)
	id := remote.seed("vera", "Vera")
	mustSync(t, e)
	if err := remote.Delete(context.Background(), id, 1); err != nil {
		t.Fatal(err)
	}
	rep := mustSync(t, e)
	if len(rep.Trashed) != 1 {
		t.Fatalf("report = %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(e.Dir(), "vera")); !os.IsNotExist(err) {
		t.Error("the folder is still there")
	}
	trash, _ := os.ReadDir(filepath.Join(e.Dir(), ".trash"))
	if len(trash) != 1 {
		t.Errorf("trash = %v", trash)
	}
}

func TestRemovedFolderComesBack(t *testing.T) {
	e, remote := setup(t)
	id := remote.seed("vera", "Vera")
	mustSync(t, e)
	if err := os.RemoveAll(filepath.Join(e.Dir(), "vera")); err != nil {
		t.Fatal(err)
	}
	mustSync(t, e)
	if _, err := os.Stat(filepath.Join(e.Dir(), "vera", manifestName)); err != nil {
		t.Error("the removed folder was not downloaded again")
	}
	if remote.agents[id].Deleted {
		t.Error("removing a folder deleted the agent")
	}
}

func TestEditorLeftoversAreIgnored(t *testing.T) {
	e, remote := setup(t)
	remote.seed("vera", "Vera")
	mustSync(t, e)
	write(t, filepath.Join(e.Dir(), "vera/skills/go-tests/.SKILL.md.swp"), "swap")
	write(t, filepath.Join(e.Dir(), "vera/skills/go-tests/SKILL.md~"), "backup")
	if rep := mustSync(t, e); !rep.UpToDate || remote.revises != 0 {
		t.Errorf("leftovers counted as a change: %+v", rep)
	}
}

func TestAnotherMembersFolderMovesAside(t *testing.T) {
	e, remote := setup(t)
	remote.seed("vera", "Vera")
	mustSync(t, e) // member 7
	if _, err := e.Sync(context.Background(), 8); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.base, "agents-api.test-7", "vera")); err != nil {
		t.Errorf("member 7's folder was not set aside: %v", err)
	}
	if _, err := e.Sync(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.Dir(), "vera")); err != nil {
		t.Errorf("member 7's folder did not come back: %v", err)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func TestRosterEditing(t *testing.T) {
	e, remote := setup(t)
	mustSync(t, e)
	slug, err := e.NewAgent(Manifest{Name: "Ivo the Researcher", Title: "Researcher", Traits: []string{"curious"}})
	if err != nil || slug != "ivo-the-researcher" {
		t.Fatalf("NewAgent = %q, %v", slug, err)
	}
	if again, _ := e.NewAgent(Manifest{Name: "Ivo the Researcher"}); again != "ivo-the-researcher-2" {
		t.Errorf("second slug = %q", again)
	}
	src := filepath.Join(t.TempDir(), "go-tests")
	write(t, filepath.Join(src, "SKILL.md"), skill)
	write(t, filepath.Join(src, "examples", "table.go"), "package x")
	write(t, filepath.Join(src, ".git", "HEAD"), "ref")
	if name, err := e.AddSkill(slug, src); err != nil || name != "go-tests" {
		t.Fatalf("AddSkill = %q, %v", name, err)
	}
	// A symlinked skill, as in ~/.claude/skills.
	real := filepath.Join(t.TempDir(), "real-notes")
	write(t, filepath.Join(real, "SKILL.md"), "---\nname: notes\ndescription: Notes.\n---\n")
	link := filepath.Join(t.TempDir(), "notes")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if name, err := e.AddSkill(slug, link); err != nil || name != "notes" {
		t.Fatalf("AddSkill of a symlink = %q, %v", name, err)
	}
	if err := e.RemoveSkill(slug, "notes"); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(t.TempDir(), "broken")
	write(t, filepath.Join(bad, "SKILL.md"), "no front matter")
	if _, err := e.AddSkill(slug, bad); err == nil {
		t.Error("a broken skill was added")
	}
	f, _ := e.Folder(slug)
	if len(f.Files) != 2 {
		t.Errorf("files = %v (hidden directories must stay out)", f.Files)
	}
	rep := mustSync(t, e)
	if len(rep.Created) != 2 {
		t.Fatalf("report = %+v", rep)
	}
	var id uint64
	for _, a := range remote.agents {
		if a.Slug == slug {
			id = a.ID
		}
	}
	if err := e.Delete(context.Background(), 7, slug); err != nil {
		t.Fatal(err)
	}
	if !remote.agents[id].Deleted {
		t.Error("deleting in the app did not delete on the server")
	}
	if _, err := e.Folder(slug); err == nil {
		t.Error("the folder is still there")
	}
	if rep := mustSync(t, e); len(rep.Pulled) != 0 {
		t.Errorf("a deleted agent came back: %+v", rep)
	}
}
