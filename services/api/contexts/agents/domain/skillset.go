package domain

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// The skillset's limits.
const (
	maxSkills        = 50
	maxFiles         = 300
	maxFileBytes     = 256 << 10
	maxSkillsetBytes = 2 << 20
)

// File is one file of an agent's skillset, by its path under
// `.claude/skills` (`<skill>/<file…>`).
type File struct {
	Path    string
	Content string
}

// SkillsetError says which file breaks which rule.
type SkillsetError struct {
	Path   string
	Reason string
}

func (e *SkillsetError) Error() string {
	if e.Path == "" {
		return e.Reason
	}
	return e.Path + ": " + e.Reason
}

var skillName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Skill is what a skill says about itself in its SKILL.md front matter.
type Skill struct {
	Name        string
	Description string
}

// CheckSkillset applies the skillset rules and returns the skills in the
// order they first appear.
func CheckSkillset(files []File) ([]Skill, error) {
	if len(files) > maxFiles {
		return nil, &SkillsetError{Reason: fmt.Sprintf("at most %d files", maxFiles)}
	}
	total := 0
	seen := map[string]bool{}
	var order []string
	manifests := map[string]string{}
	for _, f := range files {
		skill, err := checkPath(f.Path)
		if err != nil {
			return nil, err
		}
		if seen[f.Path] {
			return nil, &SkillsetError{f.Path, "appears twice"}
		}
		seen[f.Path] = true
		if !utf8.ValidString(f.Content) || strings.ContainsRune(f.Content, 0) {
			return nil, &SkillsetError{f.Path, "must be UTF-8 text"}
		}
		if len(f.Content) > maxFileBytes {
			return nil, &SkillsetError{f.Path, "is larger than 256 KB"}
		}
		total += len(f.Content)
		if _, ok := manifests[skill]; !ok {
			order = append(order, skill)
			manifests[skill] = ""
		}
		if f.Path == skill+"/SKILL.md" {
			manifests[skill] = f.Content
		}
	}
	if total > maxSkillsetBytes {
		return nil, &SkillsetError{Reason: "a skillset is at most 2 MB"}
	}
	if len(order) > maxSkills {
		return nil, &SkillsetError{Reason: fmt.Sprintf("at most %d skills", maxSkills)}
	}
	skills := make([]Skill, 0, len(order))
	for _, name := range order {
		manifest := manifests[name]
		if !seen[name+"/SKILL.md"] {
			return nil, &SkillsetError{name + "/", "has no SKILL.md"}
		}
		fields, ok := frontMatter(manifest)
		if !ok {
			return nil, &SkillsetError{name + "/SKILL.md", "needs front matter between --- lines"}
		}
		if fields["name"] != name {
			return nil, &SkillsetError{name + "/SKILL.md", fmt.Sprintf("its name must be %q, the directory's name", name)}
		}
		if fields["description"] == "" {
			return nil, &SkillsetError{name + "/SKILL.md", "needs a description"}
		}
		skills = append(skills, Skill{Name: name, Description: fields["description"]})
	}
	return skills, nil
}

// checkPath returns the skill a path belongs to.
func checkPath(path string) (string, error) {
	if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "\\") {
		return "", &SkillsetError{path, "must be a relative path like <skill>/SKILL.md"}
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", &SkillsetError{path, "must sit inside a skill directory"}
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return "", &SkillsetError{path, "has an empty, . or .. part"}
		}
	}
	for _, dir := range parts[:len(parts)-1] {
		if strings.HasPrefix(dir, ".") {
			return "", &SkillsetError{path, "is inside a hidden directory"}
		}
	}
	if len(parts[0]) > 64 || !skillName.MatchString(parts[0]) {
		return "", &SkillsetError{path, "skill names are lowercase letters, digits and hyphens, at most 64"}
	}
	return parts[0], nil
}

// frontMatter reads the top-level `key: value` lines of a YAML front matter
// block. It understands what SKILL.md files use: plain, quoted and folded
// (`>` / `|`) values. Other keys are kept in the file as they are; only
// name and description are read. The domain has no YAML library.
func frontMatter(content string) (map[string]string, bool) {
	content = strings.TrimPrefix(content, string(rune(0xFEFF)))
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, false
	}
	fields := map[string]string{}
	var block string // key of a folded value being collected
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return fields, true
		}
		if block != "" && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || strings.TrimSpace(line) == "") {
			if t := strings.TrimSpace(line); t != "" {
				fields[block] = strings.TrimSpace(fields[block] + " " + t)
			}
			continue
		}
		block = ""
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch {
		case value == ">" || value == "|" || value == ">-" || value == "|-":
			block = key
			fields[key] = ""
		case len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0]:
			fields[key] = value[1 : len(value)-1]
		default:
			fields[key] = value
		}
	}
	return nil, false
}
