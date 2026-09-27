package agents

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

var ErrNoAgent = errors.New("no agent with that name here")

// Slugify makes a folder name from an agent's name, as the API does.
func Slugify(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 40 {
		s = strings.Trim(s[:40], "-")
	}
	if s == "" {
		s = "agent"
	}
	return s
}

// NewAgent makes a folder for a new agent (sent on the next sync) and
// returns its slug: the name's, or with -2, -3, … when taken.
func (e *Engine) NewAgent(m Manifest) (string, error) {
	if m.Model == "" {
		m.Model = "sonnet"
	}
	if m.PermissionMode == "" {
		m.PermissionMode = "manual"
	}
	if m.PortraitSeed == "" {
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		m.PortraitSeed = hex.EncodeToString(b)
	}
	if err := os.MkdirAll(e.Dir(), 0o755); err != nil {
		return "", err
	}
	base := Slugify(m.Name)
	slug := base
	for n := 2; ; n++ {
		if _, err := os.Stat(filepath.Join(e.Dir(), slug)); errors.Is(err, fs.ErrNotExist) {
			break
		}
		slug = fmt.Sprintf("%s-%d", base, n)
	}
	if err := os.MkdirAll(filepath.Join(e.Dir(), slug, skillsDir), 0o755); err != nil {
		return "", err
	}
	return slug, writeManifest(filepath.Join(e.Dir(), slug), m)
}

// SaveManifest rewrites an agent's agent.toml; the watcher sends it.
func (e *Engine) SaveManifest(slug string, m Manifest) error {
	dir, err := e.agentDir(slug)
	if err != nil {
		return err
	}
	return writeManifest(dir, m)
}

// writeManifest writes agent.toml in dir, replacing it in one step.
func writeManifest(dir string, m Manifest) error {
	m.Traits, m.AllowedTools = nonNil(m.Traits), nonNil(m.AllowedTools)
	if m.WorkPriorities == nil {
		m.WorkPriorities = map[string]int{}
	}
	b, err := toml.Marshal(m)
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "."+manifestName+".tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, manifestName))
}

func (e *Engine) agentDir(slug string) (string, error) {
	if slug == "" || strings.ContainsAny(slug, `/\`) || strings.HasPrefix(slug, ".") {
		return "", ErrNoAgent
	}
	dir := filepath.Join(e.Dir(), slug)
	if _, err := os.Stat(filepath.Join(dir, manifestName)); err != nil {
		return "", ErrNoAgent
	}
	return dir, nil
}

// Folder reads one agent's folder.
func (e *Engine) Folder(slug string) (Folder, error) {
	dir, err := e.agentDir(slug)
	if err != nil {
		return Folder{}, err
	}
	return ReadFolder(dir)
}

// AddSkill copies a skill directory (with its SKILL.md) into the agent,
// under the skill's own name, replacing one of that name. The agent is
// checked afterwards; a copy that breaks a rule is taken back out.
func (e *Engine) AddSkill(slug, from string) (string, error) {
	dir, err := e.agentDir(slug)
	if err != nil {
		return "", err
	}
	// Skills are often symlinked into ~/.claude/skills; WalkDir does not
	// follow a symlinked root, so resolve it first.
	if resolved, err := filepath.EvalSymlinks(from); err == nil {
		from = resolved
	}
	manifest, err := os.ReadFile(filepath.Join(from, "SKILL.md"))
	if err != nil {
		return "", fmt.Errorf("%s has no SKILL.md", from)
	}
	name, _, ok := SkillOf(string(manifest))
	if !ok || name == "" {
		name = filepath.Base(from)
	}
	target := filepath.Join(dir, skillsDir, name)
	tmp := target + ".importing"
	_ = os.RemoveAll(tmp)
	err = filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		if d.IsDir() {
			if rel != "." && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(tmp, rel), 0o755)
		}
		if ignored(d.Name()) || !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(tmp, rel), b, 0o644)
	})
	if err != nil {
		_ = os.RemoveAll(tmp)
		return "", err
	}
	old := target + ".replaced"
	_ = os.RemoveAll(old)
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, old); err != nil {
			return "", err
		}
	}
	if err := os.Rename(tmp, target); err != nil {
		return "", err
	}
	f, err := ReadFolder(dir)
	if err == nil {
		err = Check(f)
	}
	if err != nil {
		_ = os.RemoveAll(target)
		if _, statErr := os.Stat(old); statErr == nil {
			_ = os.Rename(old, target)
		}
		return "", err
	}
	_ = os.RemoveAll(old)
	return name, nil
}

// RemoveSkill deletes a skill from the agent.
func (e *Engine) RemoveSkill(slug, skill string) error {
	dir, err := e.agentDir(slug)
	if err != nil {
		return err
	}
	if skill == "" || strings.ContainsAny(skill, `/\`) || strings.HasPrefix(skill, ".") {
		return fmt.Errorf("no skill %q", skill)
	}
	return os.RemoveAll(filepath.Join(dir, skillsDir, skill))
}

// Delete deletes an agent: on the server when it was ever synced (based on
// the revision last synced), then its folder moves to .trash.
func (e *Engine) Delete(ctx context.Context, memberID uint64, slug string) error {
	if err := e.scope(memberID); err != nil {
		return err
	}
	f, err := e.Folder(slug)
	if err != nil {
		return err
	}
	if f.Sync != nil {
		if err := e.remote.Delete(ctx, f.Sync.AgentID, f.Sync.Revision); err != nil {
			return err
		}
	}
	st, err := e.loadState()
	if err != nil {
		return err
	}
	if f.Sync != nil {
		delete(st.Known, f.Sync.AgentID)
	}
	if err := e.trash(f); err != nil {
		return err
	}
	return e.saveState(st)
}
