package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
)

// Remote is the API as the engine needs it, already signed in.
type Remote interface {
	List(ctx context.Context, since *time.Time) ([]api.Agent, error)
	Get(ctx context.Context, id uint64) (api.Agent, error)
	Create(ctx context.Context, a api.AgentWrite) (api.Agent, error)
	// Revise returns a *api.StaleError when the agent moved on.
	Revise(ctx context.Context, id uint64, basedOn int, a api.AgentWrite) (api.Agent, error)
	Delete(ctx context.Context, id uint64, basedOn int) error
}

// Conflict is an agent changed both here and on the server. The folder
// keeps the local version; the server's is under .conflicts/<slug>/.
type Conflict struct {
	Slug           string `json:"slug"`
	AgentID        uint64 `json:"agent_id"`
	ServerRevision int    `json:"server_revision"`
}

// state is .state.json in the agents folder.
type state struct {
	// Owner says whose agents the folder holds: the API's host and the
	// member's id there (member ids of dev and prod are unrelated).
	Owner     string            `json:"owner"`
	LastSync  *time.Time        `json:"last_sync,omitempty"`
	Known     map[uint64]string `json:"known"` // agent id → slug
	Conflicts []Conflict        `json:"conflicts"`
}

// Report is what one sync did.
type Report struct {
	Pushed     []string
	Pulled     []string
	Created    []string
	Trashed    []string
	Conflicts  []Conflict
	Problems   []error // folders not sent because they break a rule
	UpToDate   bool
	FinishedAt time.Time
}

// Engine keeps <base>/agents in step with the member's agents on the API.
type Engine struct {
	base   string
	host   string
	remote Remote
	now    func() time.Time
}

// NewEngine keeps agents under base (e.g. ~/.config/the-bakery) for the API
// at host.
func NewEngine(base, host string, remote Remote) *Engine {
	return &Engine{base: base, host: safeName(host), remote: remote, now: time.Now}
}

// safeName keeps letters, digits, dots and hyphens, for a folder name.
func safeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// Dir is the signed-in member's agents folder.
func (e *Engine) Dir() string { return filepath.Join(e.base, "agents") }

func (e *Engine) statePath() string    { return filepath.Join(e.Dir(), ".state.json") }
func (e *Engine) conflictsDir() string { return filepath.Join(e.Dir(), ".conflicts") }
func (e *Engine) trashDir() string     { return filepath.Join(e.Dir(), ".trash") }

func (e *Engine) loadState() (state, error) {
	st := state{Known: map[uint64]string{}}
	b, err := os.ReadFile(e.statePath())
	if errors.Is(err, fs.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return state{Known: map[uint64]string{}}, nil
	}
	if st.Known == nil {
		st.Known = map[uint64]string{}
	}
	return st, nil
}

func (e *Engine) saveState(st state) error {
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := e.statePath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, e.statePath())
}

// scope makes <base>/agents the member's folder. Another member's (or
// another API's) folder moves aside to agents-<host>-<id>, and this
// member's comes back if it was set aside before. Nothing is deleted.
func (e *Engine) scope(memberID uint64) error {
	if err := os.MkdirAll(e.base, 0o755); err != nil {
		return err
	}
	st, err := e.loadState()
	if err != nil {
		return err
	}
	owner := fmt.Sprintf("%s-%d", e.host, memberID)
	if _, err := os.Stat(e.Dir()); err == nil && st.Owner != "" && st.Owner != owner {
		if err := os.Rename(e.Dir(), filepath.Join(e.base, "agents-"+st.Owner)); err != nil {
			return err
		}
	}
	if _, err := os.Stat(e.Dir()); errors.Is(err, fs.ErrNotExist) {
		mine := filepath.Join(e.base, "agents-"+owner)
		if _, err := os.Stat(mine); err == nil {
			if err := os.Rename(mine, e.Dir()); err != nil {
				return err
			}
		} else if err := os.MkdirAll(e.Dir(), 0o755); err != nil {
			return err
		}
	}
	st, err = e.loadState()
	if err != nil {
		return err
	}
	if st.Owner == "" {
		st.Owner = owner
		return e.saveState(st)
	}
	return nil
}

// folders reads every agent folder (hidden ones are the engine's own).
func (e *Engine) folders() (map[string]Folder, error) {
	entries, err := os.ReadDir(e.Dir())
	if err != nil {
		return nil, err
	}
	out := map[string]Folder{}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		f, err := ReadFolder(filepath.Join(e.Dir(), entry.Name()))
		if err != nil {
			return nil, err
		}
		out[f.Slug] = f
	}
	return out, nil
}

// Conflicts lists the agents waiting for the member to choose a version.
func (e *Engine) Conflicts() ([]Conflict, error) {
	st, err := e.loadState()
	return st.Conflicts, err
}

// Sync runs one pass for the member:
//   - an agent unchanged on both sides is left alone;
//   - changed only here: sent, based on the revision last synced;
//   - changed only on the server: written into its folder;
//   - changed on both, or the server refused the revision: a conflict;
//   - a folder without .sync.json: created as a new agent;
//   - deleted on the server: its folder moves to .trash;
//   - a folder the member removed by hand: downloaded again (agents are
//     deleted from the app, never by a missing folder).
func (e *Engine) Sync(ctx context.Context, memberID uint64) (Report, error) {
	var rep Report
	if err := e.scope(memberID); err != nil {
		return rep, err
	}
	st, err := e.loadState()
	if err != nil {
		return rep, err
	}
	started := e.now()
	var since *time.Time
	if st.LastSync != nil {
		// A little slack for clocks that disagree.
		t := st.LastSync.Add(-5 * time.Second)
		since = &t
	}
	changed, err := e.remote.List(ctx, since)
	if err != nil {
		return rep, err
	}
	local, err := e.folders()
	if err != nil {
		return rep, err
	}
	byID := map[uint64]Folder{}
	for _, f := range local {
		if f.Sync != nil {
			byID[f.Sync.AgentID] = f
		}
	}
	conflicted := map[string]bool{}
	for _, c := range st.Conflicts {
		conflicted[c.Slug] = true
	}
	handled := map[string]bool{}
	remoteSeen := map[uint64]bool{}

	for _, r := range changed {
		remoteSeen[r.ID] = true
		f, here := byID[r.ID]
		switch {
		case r.Deleted:
			if here {
				if err := e.trash(f); err != nil {
					return rep, err
				}
				rep.Trashed = append(rep.Trashed, f.Slug)
				handled[f.Slug] = true
			}
			delete(st.Known, r.ID)
		case !here:
			if other, taken := local[r.Slug]; taken && other.Sync == nil {
				// A new local folder already uses the slug: keep it, and put
				// the server's agent aside as a conflict.
				if err := e.conflict(ctx, &st, other.Slug, r.ID); err != nil {
					return rep, err
				}
				handled[other.Slug] = true
				continue
			}
			slug := r.Slug
			if _, taken := local[slug]; taken {
				// Another agent's folder has the name (renamed elsewhere).
				slug = fmt.Sprintf("%s-%d", r.Slug, r.ID)
			}
			if err := e.download(ctx, r.ID, slug); err != nil {
				return rep, err
			}
			st.Known[r.ID] = slug
			rep.Pulled = append(rep.Pulled, slug)
			handled[slug] = true
		case conflicted[f.Slug]:
			handled[f.Slug] = true
		case r.Revision == f.Sync.Revision:
			// Nothing new on the server; local changes are handled below.
		case f.Changed():
			if err := e.conflict(ctx, &st, f.Slug, r.ID); err != nil {
				return rep, err
			}
			handled[f.Slug] = true
		default:
			if err := e.download(ctx, r.ID, f.Slug); err != nil {
				return rep, err
			}
			rep.Pulled = append(rep.Pulled, f.Slug)
			handled[f.Slug] = true
		}
	}

	slugs := make([]string, 0, len(local))
	for s := range local {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)
	for _, slug := range slugs {
		f := local[slug]
		if handled[slug] || conflicted[slug] {
			continue
		}
		if f.Sync != nil && !f.Changed() {
			continue
		}
		if err := Check(f); err != nil {
			rep.Problems = append(rep.Problems, err)
			continue
		}
		if f.Sync == nil {
			a, err := e.remote.Create(ctx, f.Write())
			if err != nil {
				var apiErr *api.Error
				if errors.As(err, &apiErr) && apiErr.Status == http.StatusUnprocessableEntity {
					rep.Problems = append(rep.Problems, &Problem{slug, apiErr.Message})
					continue
				}
				return rep, err
			}
			if err := e.recordSynced(f, a); err != nil {
				return rep, err
			}
			st.Known[a.ID] = slug
			rep.Created = append(rep.Created, slug)
			continue
		}
		a, err := e.remote.Revise(ctx, f.Sync.AgentID, f.Sync.Revision, f.Write())
		var staleErr *api.StaleError
		switch {
		case errors.As(err, &staleErr):
			if err := e.conflict(ctx, &st, slug, f.Sync.AgentID); err != nil {
				return rep, err
			}
			continue
		case err != nil:
			var apiErr *api.Error
			if errors.As(err, &apiErr) && apiErr.Status == http.StatusUnprocessableEntity {
				rep.Problems = append(rep.Problems, &Problem{slug, apiErr.Message})
				continue
			}
			if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
				// Deleted on the server since: the next pass trashes it.
				continue
			}
			return rep, err
		}
		if err := e.recordSynced(f, a); err != nil {
			return rep, err
		}
		rep.Pushed = append(rep.Pushed, slug)
	}

	// Known agents whose folder went missing come back.
	for id, slug := range st.Known {
		if _, here := byID[id]; here || remoteSeen[id] {
			continue
		}
		if _, taken := local[slug]; taken {
			continue
		}
		err := e.download(ctx, id, slug)
		var apiErr *api.Error
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			delete(st.Known, id)
			continue
		}
		if err != nil {
			return rep, err
		}
		rep.Pulled = append(rep.Pulled, slug)
	}

	st.LastSync = &started
	rep.Conflicts = st.Conflicts
	rep.UpToDate = len(rep.Pushed)+len(rep.Pulled)+len(rep.Created)+len(rep.Trashed) == 0
	rep.FinishedAt = e.now()
	return rep, e.saveState(st)
}

// recordSynced notes in .sync.json that the folder now matches the server
// at a's revision.
func (e *Engine) recordSynced(f Folder, a api.Agent) error {
	fresh, err := ReadFolder(f.Dir)
	if err != nil {
		return err
	}
	return writeSync(f.Dir, SyncFile{AgentID: a.ID, Revision: a.Revision, Hashes: fresh.Hashes})
}

func (e *Engine) download(ctx context.Context, id uint64, slug string) error {
	a, err := e.remote.Get(ctx, id)
	if err != nil {
		return err
	}
	return WriteAgent(filepath.Join(e.Dir(), slug), a)
}

// conflict keeps the local folder and writes the server's version under
// .conflicts/<slug>/ for the member to choose from.
func (e *Engine) conflict(ctx context.Context, st *state, slug string, id uint64) error {
	a, err := e.remote.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := WriteAgent(filepath.Join(e.conflictsDir(), slug), a); err != nil {
		return err
	}
	for i, c := range st.Conflicts {
		if c.Slug == slug {
			st.Conflicts[i] = Conflict{Slug: slug, AgentID: id, ServerRevision: a.Revision}
			return nil
		}
	}
	st.Conflicts = append(st.Conflicts, Conflict{Slug: slug, AgentID: id, ServerRevision: a.Revision})
	return nil
}

func (e *Engine) trash(f Folder) error {
	if err := os.MkdirAll(e.trashDir(), 0o755); err != nil {
		return err
	}
	return os.Rename(f.Dir, filepath.Join(e.trashDir(), f.Slug+"-"+e.now().Format("20060102-150405")))
}
