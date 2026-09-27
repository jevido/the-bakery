// Package agents mirrors the signed-in member's agents as folders and keeps
// them in step with the API:
//
//	<base>/agents/<slug>/
//	  agent.toml       who the agent is and how it runs
//	  skills/<skill>/  its skillset (.claude/skills), SKILL.md and more
//	  .sync.json       agent id, revision and file hashes at the last sync
//
// The folders are a copy the app syncs; the API owns the agents.
package agents

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
)

const (
	manifestName = "agent.toml"
	syncName     = ".sync.json"
	skillsDir    = "skills"
)

// Manifest is agent.toml: everything about an agent except its skillset.
type Manifest struct {
	Name           string         `toml:"name" json:"name"`
	Title          string         `toml:"title" json:"title"`
	Backstory      string         `toml:"backstory,multiline" json:"backstory"`
	Traits         []string       `toml:"traits" json:"traits"`
	Model          string         `toml:"model" json:"model"`
	PermissionMode string         `toml:"permission_mode" json:"permission_mode"`
	AllowedTools   []string       `toml:"allowed_tools" json:"allowed_tools"`
	PortraitSeed   string         `toml:"portrait_seed" json:"portrait_seed"`
	WorkPriorities map[string]int `toml:"work_priorities" json:"work_priorities"`
}

// SyncFile is .sync.json: what the folder held at its last sync.
type SyncFile struct {
	AgentID  uint64            `json:"agent_id"`
	Revision int               `json:"revision"`
	Hashes   map[string]string `json:"hashes"`
}

// Folder is one agent's folder as read from disk.
type Folder struct {
	Slug     string
	Dir      string
	Manifest Manifest
	// Files are the skillset, keyed by their API path (`<skill>/<file>`).
	Files map[string]string
	// Hashes are the sha256 of agent.toml and every skills/ file, keyed by
	// their path in the folder.
	Hashes map[string]string
	Sync   *SyncFile
	// ManifestErr is set when agent.toml cannot be read.
	ManifestErr error
}

// Changed reports whether the folder differs from its last sync.
func (f Folder) Changed() bool {
	if f.Sync == nil {
		return true
	}
	if len(f.Hashes) != len(f.Sync.Hashes) {
		return true
	}
	for k, v := range f.Hashes {
		if f.Sync.Hashes[k] != v {
			return true
		}
	}
	return false
}

// Write is the folder as an agent to send.
func (f Folder) Write() api.AgentWrite {
	m := f.Manifest
	w := api.AgentWrite{
		Slug: f.Slug, Name: m.Name, Title: m.Title, Backstory: m.Backstory, Traits: nonNil(m.Traits),
		Model: m.Model, PermissionMode: m.PermissionMode, AllowedTools: nonNil(m.AllowedTools),
		PortraitSeed: m.PortraitSeed, WorkPriorities: m.WorkPriorities, Files: []api.AgentFile{},
	}
	if w.WorkPriorities == nil {
		w.WorkPriorities = map[string]int{}
	}
	paths := make([]string, 0, len(f.Files))
	for p := range f.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		w.Files = append(w.Files, api.AgentFile{Path: p, Content: f.Files[p]})
	}
	return w
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ignored is true for files an editor or the OS leaves behind.
func ignored(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasSuffix(name, "~") || name == "4913" ||
		strings.HasSuffix(name, ".swp") || strings.HasSuffix(name, ".tmp")
}

func hash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// ReadFolder reads one agent folder.
func ReadFolder(dir string) (Folder, error) {
	f := Folder{Slug: filepath.Base(dir), Dir: dir, Files: map[string]string{}, Hashes: map[string]string{}}
	manifest, err := os.ReadFile(filepath.Join(dir, manifestName))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		f.ManifestErr = fmt.Errorf("%s has no %s", f.Slug, manifestName)
	case err != nil:
		return Folder{}, err
	default:
		f.Hashes[manifestName] = hash(manifest)
		if err := toml.Unmarshal(manifest, &f.Manifest); err != nil {
			f.ManifestErr = fmt.Errorf("%s/%s: %w", f.Slug, manifestName, err)
		}
	}
	skills := filepath.Join(dir, skillsDir)
	err = filepath.WalkDir(skills, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ignored(d.Name()) || !d.Type().IsRegular() {
			return nil
		}
		rel, _ := filepath.Rel(skills, path)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		apiPath := filepath.ToSlash(rel)
		f.Files[apiPath] = string(content)
		f.Hashes[skillsDir+"/"+apiPath] = hash(content)
		return nil
	})
	if err != nil {
		return Folder{}, err
	}
	if b, err := os.ReadFile(filepath.Join(dir, syncName)); err == nil {
		var s SyncFile
		if json.Unmarshal(b, &s) == nil && s.AgentID != 0 {
			f.Sync = &s
		}
	}
	return f, nil
}

// manifestOf is agent.toml for an agent from the API.
func manifestOf(a api.Agent) ([]byte, error) {
	m := Manifest{
		Name: a.Name, Title: a.Title, Backstory: a.Backstory, Traits: nonNil(a.Traits), Model: a.Model,
		PermissionMode: a.PermissionMode, AllowedTools: nonNil(a.AllowedTools), PortraitSeed: a.PortraitSeed,
		WorkPriorities: a.WorkPriorities,
	}
	if m.WorkPriorities == nil {
		m.WorkPriorities = map[string]int{}
	}
	return toml.Marshal(m)
}

// WriteAgent writes an agent from the API as the folder dir, replacing
// what is there. It builds the folder next to dir first and swaps it in, so
// a half-written agent never exists.
func WriteAgent(dir string, a api.Agent) error {
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".writing-"+filepath.Base(dir)+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	hashes := map[string]string{}
	manifest, err := manifestOf(a)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, manifestName), manifest, 0o644); err != nil {
		return err
	}
	hashes[manifestName] = hash(manifest)
	for _, file := range a.Files {
		if err := safePath(file.Path); err != nil {
			return err
		}
		path := filepath.Join(tmp, skillsDir, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(file.Content), 0o644); err != nil {
			return err
		}
		hashes[skillsDir+"/"+file.Path] = hash([]byte(file.Content))
	}
	if err := writeSync(tmp, SyncFile{AgentID: a.ID, Revision: a.Revision, Hashes: hashes}); err != nil {
		return err
	}
	// Swap: the old folder moves aside, the new one takes its name.
	old := ""
	if _, err := os.Stat(dir); err == nil {
		old = filepath.Join(parent, ".replaced-"+filepath.Base(dir)+"-"+randomSuffix())
		if err := os.Rename(dir, old); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, dir); err != nil {
		if old != "" {
			_ = os.Rename(old, dir)
		}
		return err
	}
	if old != "" {
		return os.RemoveAll(old)
	}
	return nil
}

func writeSync(dir string, s SyncFile) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, syncName), b, 0o644)
}

// safePath refuses a server path that would leave the skills directory.
func safePath(p string) error {
	if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "\\") {
		return fmt.Errorf("unsafe path %q", p)
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("unsafe path %q", p)
		}
	}
	return nil
}

func randomSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
