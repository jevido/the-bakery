package agents

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
)

// Choice is how a member settles a conflict.
type Choice string

const (
	// KeepLocal sends the folder's version over the server's.
	KeepLocal Choice = "local"
	// TakeServer replaces the folder with the server's version.
	TakeServer Choice = "server"
	// KeepBoth takes the server's version into the folder and makes the
	// local version a new agent, <slug>-local.
	KeepBoth Choice = "both"
)

var ErrNoConflict = errors.New("that agent has no conflict")

// FileDiff is one file that differs between the two versions. Missing on a
// side means the file does not exist there.
type FileDiff struct {
	Path          string `json:"path"`
	Local         string `json:"local"`
	Server        string `json:"server"`
	MissingLocal  bool   `json:"missing_local"`
	MissingServer bool   `json:"missing_server"`
}

// Diff lists the files that differ between the folder and the server's
// version of a conflicted agent (agent.toml included).
func (e *Engine) Diff(slug string) ([]FileDiff, error) {
	local, err := ReadFolder(filepath.Join(e.Dir(), slug))
	if err != nil {
		return nil, err
	}
	server, err := ReadFolder(filepath.Join(e.conflictsDir(), slug))
	if err != nil {
		return nil, err
	}
	contents := func(f Folder) map[string]string {
		out := map[string]string{}
		if b, err := os.ReadFile(filepath.Join(f.Dir, manifestName)); err == nil {
			out[manifestName] = string(b)
		}
		for p, c := range f.Files {
			out[skillsDir+"/"+p] = c
		}
		return out
	}
	l, s := contents(local), contents(server)
	paths := map[string]bool{}
	for p := range l {
		paths[p] = true
	}
	for p := range s {
		paths[p] = true
	}
	var out []FileDiff
	for p := range paths {
		lc, lok := l[p]
		sc, sok := s[p]
		if lok && sok && lc == sc {
			continue
		}
		out = append(out, FileDiff{Path: p, Local: lc, Server: sc, MissingLocal: !lok, MissingServer: !sok})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// Resolve settles a conflict. Afterwards the folder (and, for KeepBoth, the
// new <slug>-local folder) matches the server.
func (e *Engine) Resolve(ctx context.Context, memberID uint64, slug string, choice Choice) error {
	if err := e.scope(memberID); err != nil {
		return err
	}
	st, err := e.loadState()
	if err != nil {
		return err
	}
	at := -1
	for i, c := range st.Conflicts {
		if c.Slug == slug {
			at = i
		}
	}
	if at < 0 {
		return ErrNoConflict
	}
	c := st.Conflicts[at]
	dir := filepath.Join(e.Dir(), slug)

	switch choice {
	case KeepLocal:
		f, err := ReadFolder(dir)
		if err != nil {
			return err
		}
		if err := Check(f); err != nil {
			return err
		}
		current, err := e.remote.Get(ctx, c.AgentID)
		if err != nil {
			return err
		}
		a, err := e.remote.Revise(ctx, c.AgentID, current.Revision, f.Write())
		var staleErr *api.StaleError
		if errors.As(err, &staleErr) {
			// It moved on again: show the newer server version instead.
			if err := e.conflict(ctx, &st, slug, c.AgentID); err != nil {
				return err
			}
			return e.saveState(st)
		}
		if err != nil {
			return err
		}
		if err := e.recordSynced(f, a); err != nil {
			return err
		}
	case TakeServer:
		a, err := e.remote.Get(ctx, c.AgentID)
		if err != nil {
			return err
		}
		if err := WriteAgent(dir, a); err != nil {
			return err
		}
	case KeepBoth:
		copySlug := slug + "-local"
		for n := 2; ; n++ {
			if _, err := os.Stat(filepath.Join(e.Dir(), copySlug)); os.IsNotExist(err) {
				break
			}
			copySlug = fmt.Sprintf("%s-local-%d", slug, n)
		}
		copyDir := filepath.Join(e.Dir(), copySlug)
		if err := os.Rename(dir, copyDir); err != nil {
			return err
		}
		// Without .sync.json the next sync creates it as a new agent.
		if err := os.Remove(filepath.Join(copyDir, syncName)); err != nil && !os.IsNotExist(err) {
			return err
		}
		a, err := e.remote.Get(ctx, c.AgentID)
		if err != nil {
			return err
		}
		if err := WriteAgent(dir, a); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown choice %q", choice)
	}
	st.Conflicts = append(st.Conflicts[:at], st.Conflicts[at+1:]...)
	if err := os.RemoveAll(filepath.Join(e.conflictsDir(), slug)); err != nil {
		return err
	}
	return e.saveState(st)
}
