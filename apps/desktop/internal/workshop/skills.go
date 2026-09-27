package workshop

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// AssembleSkills places the agent's skills and the board's skills in the
// worktree's .claude/skills, where Claude CLI finds them: each agent skill
// as bakery-<agent>-<name>, each board skill as bakery-board-<name>, with the
// name in its SKILL.md changed to match. Whatever an earlier run placed
// there goes first. It returns the names placed, sorted.
//
// agentSkills and boardSkills are the skills/ folders; either may be
// missing.
func AssembleSkills(wt Worktree, agentSlug, agentSkills, boardSkills string) ([]string, error) {
	dest := filepath.Join(wt.Path, ".claude", "skills")
	old, err := filepath.Glob(filepath.Join(dest, "bakery-*"))
	if err != nil {
		return nil, err
	}
	for _, dir := range old {
		if err := os.RemoveAll(dir); err != nil {
			return nil, err
		}
	}
	var names []string
	for _, src := range []struct{ dir, prefix string }{
		{agentSkills, "bakery-" + agentSlug + "-"},
		{boardSkills, "bakery-board-"},
	} {
		placed, err := placeSkills(src.dir, dest, src.prefix)
		if err != nil {
			return nil, err
		}
		names = append(names, placed...)
	}
	sort.Strings(names)
	return names, nil
}

// placeSkills copies every skill folder under src to dest/<prefix><name>.
func placeSkills(src, dest, prefix string) ([]string, error) {
	if src == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(src)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		from := filepath.Join(src, e.Name())
		manifest, err := os.ReadFile(filepath.Join(from, "SKILL.md"))
		if err != nil {
			return nil, fmt.Errorf("skill %s has no SKILL.md", from)
		}
		name := prefix + e.Name()
		renamed, err := renameSkill(string(manifest), name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Join(from, "SKILL.md"), err)
		}
		to := filepath.Join(dest, name)
		if err := copyTree(from, to); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(to, "SKILL.md"), []byte(renamed), 0o644); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

// renameSkill sets the name in a SKILL.md's front matter. Claude CLI skips
// a skill without front matter naming and describing it, so that is an
// error here rather than a skill that silently goes missing.
func renameSkill(manifest, name string) (string, error) {
	text := strings.TrimPrefix(strings.ReplaceAll(manifest, "\r\n", "\n"), string(rune(0xFEFF)))
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", errors.New("needs front matter (a --- block with name and description)")
	}
	nameAt, described := -1, false
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "---" {
			if nameAt < 0 {
				return "", errors.New("front matter needs a name")
			}
			if !described {
				return "", errors.New("front matter needs a description")
			}
			lines[nameAt] = "name: " + name
			return strings.Join(lines, "\n"), nil
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		switch strings.TrimSpace(key) {
		case "name":
			nameAt = i
		case "description":
			// A folded value (description: >) goes on the next lines.
			described = strings.TrimSpace(value) != "" && strings.Trim(strings.TrimSpace(value), `"'`) != ""
		}
	}
	return "", errors.New("front matter is not closed with ---")
}

// copyTree copies the regular files under src to dst.
func copyTree(src, dst string) error {
	root, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		to := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(to, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(to, b, 0o644)
	})
}
