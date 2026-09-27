package agents

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// The API's skillset rules, checked before sending so a mistake shows up
// with its path instead of as a refused sync. The API stays the judge.
const (
	maxSkills    = 50
	maxFiles     = 300
	maxFileBytes = 256 << 10
	maxAllBytes  = 2 << 20
)

var (
	skillName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	slugName  = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// Problem is something wrong with a folder, by the file it is about.
type Problem struct {
	Path   string
	Reason string
}

func (p *Problem) Error() string { return p.Path + ": " + p.Reason }

// Check applies the rules to a folder before it is sent.
func Check(f Folder) error {
	if f.ManifestErr != nil {
		return &Problem{f.Slug + "/" + manifestName, f.ManifestErr.Error()}
	}
	if len(f.Slug) > 40 || !slugName.MatchString(f.Slug) {
		return &Problem{f.Slug, "a folder name is 1 to 40 lowercase letters, digits and hyphens"}
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(f.Manifest.Name)); n < 1 || n > 40 {
		return &Problem{f.Slug + "/" + manifestName, "name must be 1 to 40 characters"}
	}
	if len(f.Files) > maxFiles {
		return &Problem{f.Slug + "/" + skillsDir, fmt.Sprintf("at most %d files", maxFiles)}
	}
	total := 0
	skills := map[string]string{}
	for path, content := range f.Files {
		where := f.Slug + "/" + skillsDir + "/" + path
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return &Problem{where, "files go inside a skill's directory"}
		}
		for _, dir := range parts[:len(parts)-1] {
			if strings.HasPrefix(dir, ".") {
				return &Problem{where, "is inside a hidden directory"}
			}
		}
		if len(parts[0]) > 64 || !skillName.MatchString(parts[0]) {
			return &Problem{where, "skill names are lowercase letters, digits and hyphens, at most 64"}
		}
		if !utf8.ValidString(content) || strings.ContainsRune(content, 0) {
			return &Problem{where, "must be UTF-8 text"}
		}
		if len(content) > maxFileBytes {
			return &Problem{where, "is larger than 256 KB"}
		}
		total += len(content)
		if _, ok := skills[parts[0]]; !ok {
			skills[parts[0]] = ""
		}
		if len(parts) == 2 && parts[1] == "SKILL.md" {
			skills[parts[0]] = content
		}
	}
	if total > maxAllBytes {
		return &Problem{f.Slug + "/" + skillsDir, "a skillset is at most 2 MB"}
	}
	if len(skills) > maxSkills {
		return &Problem{f.Slug + "/" + skillsDir, fmt.Sprintf("at most %d skills", maxSkills)}
	}
	for name, manifest := range skills {
		where := f.Slug + "/" + skillsDir + "/" + name + "/SKILL.md"
		if _, ok := f.Files[name+"/SKILL.md"]; !ok {
			return &Problem{f.Slug + "/" + skillsDir + "/" + name, "has no SKILL.md"}
		}
		fields, ok := frontMatter(manifest)
		if !ok {
			return &Problem{where, "needs front matter between --- lines"}
		}
		if fields["name"] != name {
			return &Problem{where, fmt.Sprintf("its name must be %q, the directory's name", name)}
		}
		if fields["description"] == "" {
			return &Problem{where, "needs a description"}
		}
	}
	return nil
}

// SkillOf reads a skill's name and description from its SKILL.md.
func SkillOf(manifest string) (name, description string, ok bool) {
	fields, ok := frontMatter(manifest)
	return fields["name"], fields["description"], ok
}

// frontMatter reads the top-level keys of a YAML front matter block, as the
// API does: plain, quoted and folded values.
func frontMatter(content string) (map[string]string, bool) {
	content = strings.TrimPrefix(content, string(rune(0xFEFF)))
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, false
	}
	fields := map[string]string{}
	block := ""
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
