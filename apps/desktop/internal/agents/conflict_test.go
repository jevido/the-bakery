package agents

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
)

// conflicted sets up vera changed both here ("Local.") and on the server
// ("Server.").
func conflicted(t *testing.T) (*Engine, *fakeRemote, uint64) {
	t.Helper()
	e, remote := setup(t)
	id := remote.seed("vera", "Vera")
	mustSync(t, e)
	write(t, filepath.Join(e.Dir(), "vera/skills/go-tests/SKILL.md"), skill+"\nLocal.\n")
	remote.serverEdit(id, func(a *api.Agent) {
		a.Files = []api.AgentFile{{Path: "go-tests/SKILL.md", Content: skill + "\nServer.\n"}}
	})
	if rep := mustSync(t, e); len(rep.Conflicts) != 1 {
		t.Fatalf("no conflict: %+v", rep)
	}
	return e, remote, id
}

func TestDiffShowsBothVersions(t *testing.T) {
	e, _, _ := conflicted(t)
	diffs, err := e.Diff("vera")
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].Path != "skills/go-tests/SKILL.md" || !contains(diffs[0].Local, "Local.") || !contains(diffs[0].Server, "Server.") {
		t.Errorf("diffs = %+v", diffs)
	}
}

func TestResolveKeepLocal(t *testing.T) {
	e, remote, id := conflicted(t)
	if err := e.Resolve(context.Background(), 7, "vera", KeepLocal); err != nil {
		t.Fatal(err)
	}
	if !contains(remote.agents[id].Files[0].Content, "Local.") {
		t.Error("the server did not get the local version")
	}
	if rep := mustSync(t, e); !rep.UpToDate || len(rep.Conflicts) != 0 {
		t.Errorf("not in step afterwards: %+v", rep)
	}
}

func TestResolveTakeServer(t *testing.T) {
	e, remote, id := conflicted(t)
	if err := e.Resolve(context.Background(), 7, "vera", TakeServer); err != nil {
		t.Fatal(err)
	}
	if !contains(read(t, filepath.Join(e.Dir(), "vera/skills/go-tests/SKILL.md")), "Server.") {
		t.Error("the folder did not take the server's version")
	}
	if contains(remote.agents[id].Files[0].Content, "Local.") {
		t.Error("the local version reached the server")
	}
	if rep := mustSync(t, e); !rep.UpToDate || len(rep.Conflicts) != 0 {
		t.Errorf("not in step afterwards: %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(e.Dir(), ".conflicts", "vera")); !os.IsNotExist(err) {
		t.Error("the server copy was left behind")
	}
}

func TestResolveKeepBoth(t *testing.T) {
	e, remote, id := conflicted(t)
	if err := e.Resolve(context.Background(), 7, "vera", KeepBoth); err != nil {
		t.Fatal(err)
	}
	rep := mustSync(t, e)
	if len(rep.Created) != 1 || rep.Created[0] != "vera-local" {
		t.Fatalf("report = %+v", rep)
	}
	if !contains(remote.agents[id].Files[0].Content, "Server.") {
		t.Error("the original agent lost the server's version")
	}
	var copyHasLocal bool
	for _, a := range remote.agents {
		if a.Slug == "vera-local" && contains(a.Files[0].Content, "Local.") {
			copyHasLocal = true
		}
	}
	if !copyHasLocal {
		t.Error("the local version did not become vera-local")
	}
	if rep := mustSync(t, e); !rep.UpToDate {
		t.Errorf("not in step afterwards: %+v", rep)
	}
}

func TestWatchReportsAChangedAgentOnce(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls atomic.Int32
	var last atomic.Value
	if err := Watch(ctx, dir, func(slug string) { calls.Add(1); last.Store(slug) }); err != nil {
		t.Fatal(err)
	}
	// A new skill directory inside a new agent, then a file in it: the
	// watcher has to add the new directories on the way.
	if err := os.MkdirAll(filepath.Join(dir, "vera", "skills", "go-tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	write(t, filepath.Join(dir, "vera/skills/go-tests/SKILL.md"), skill)
	write(t, filepath.Join(dir, "vera/skills/go-tests/.SKILL.md.swp"), "swap")
	write(t, filepath.Join(dir, ".state.json"), "{}")
	deadline := time.Now().Add(3 * time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(debounce + 200*time.Millisecond)
	if calls.Load() != 1 || last.Load() != "vera" {
		t.Errorf("calls = %d, last = %v; want one for vera", calls.Load(), last.Load())
	}
}

func TestWatchFollowsARenamedAgent(t *testing.T) {
	// "Keep both" renames a watched folder; edits there must still count.
	dir := t.TempDir()
	write(t, filepath.Join(dir, "vera/skills/go-tests/SKILL.md"), skill)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var slugs []string
	var mu sync.Mutex
	if err := Watch(ctx, dir, func(slug string) { mu.Lock(); slugs = append(slugs, slug); mu.Unlock() }); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "vera"), filepath.Join(dir, "vera-local")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(debounce + 300*time.Millisecond)
	mu.Lock()
	slugs = nil
	mu.Unlock()
	write(t, filepath.Join(dir, "vera-local/skills/go-tests/SKILL.md"), skill+"\nEdit.\n")
	time.Sleep(debounce + 500*time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(slugs) != 1 || slugs[0] != "vera-local" {
		t.Errorf("after the rename, an edit reported %v; want [vera-local]", slugs)
	}
}
