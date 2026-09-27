package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const goTests = "---\nname: go-tests\ndescription: Write table-driven Go tests.\n---\n\nUse t.Run for each case.\n"

func vera() Profile {
	return Profile{
		Name: "Vera", Title: "Backend engineer", Traits: []string{"careful", "tidy"},
		WorkPriorities: map[string]int{"coding": 1, "testing": 2},
		Files:          []File{{Path: "go-tests/SKILL.md", Content: goTests}},
	}
}

func TestNewAgent(t *testing.T) {
	a, err := NewAgent(1, "vera", vera())
	if err != nil {
		t.Fatal(err)
	}
	if a.Revision != 1 || a.Model != "sonnet" || a.PermissionMode != "manual" {
		t.Errorf("defaults: %+v", a)
	}
	if skills := a.Skills(); len(skills) != 1 || skills[0].Description != "Write table-driven Go tests." {
		t.Errorf("skills = %+v", skills)
	}
}

func TestProfileRules(t *testing.T) {
	tests := []struct {
		name    string
		change  func(p *Profile)
		wantErr error
	}{
		{"empty name", func(p *Profile) { p.Name = " " }, ErrInvalidName},
		{"long title", func(p *Profile) { p.Title = strings.Repeat("x", 61) }, ErrInvalidTitle},
		{"long backstory", func(p *Profile) { p.Backstory = strings.Repeat("x", 2001) }, ErrInvalidBackstory},
		{"unknown trait", func(p *Profile) { p.Traits = []string{"lazy"} }, ErrUnknownTrait},
		{"conflicting traits", func(p *Profile) { p.Traits = []string{"careful", "fast-worker"} }, ErrConflictingTraits},
		{"bad model", func(p *Profile) { p.Model = "gpt-4" }, ErrInvalidModel},
		{"full model id", func(p *Profile) { p.Model = "claude-sonnet-5" }, nil},
		{"bypass permissions", func(p *Profile) { p.PermissionMode = "bypassPermissions" }, ErrInvalidPermissionMode},
		{"plan mode", func(p *Profile) { p.PermissionMode = "plan" }, nil},
		{"multi-line tool", func(p *Profile) { p.AllowedTools = []string{"Bash(ls)\nrm"} }, ErrInvalidTool},
		{"too many tools", func(p *Profile) { p.AllowedTools = make([]string, 101) }, ErrTooManyTools},
		{"priority 5", func(p *Profile) { p.WorkPriorities = map[string]int{"coding": 5} }, ErrInvalidPriority},
		{"bad work type key", func(p *Profile) { p.WorkPriorities = map[string]int{"Coding": 1} }, ErrInvalidPriority},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := vera()
			tt.change(&p)
			_, err := NewAgent(1, "vera", p)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
	if _, err := NewAgent(1, "Vera!", vera()); !errors.Is(err, ErrInvalidSlug) {
		t.Errorf("bad slug error = %v", err)
	}
}

func TestSkillsetRules(t *testing.T) {
	skill := func(path, content string) File { return File{Path: path, Content: content} }
	tests := []struct {
		name  string
		files []File
		path  string // the path the error names; "" for no error
	}{
		{"one skill", []File{skill("go-tests/SKILL.md", goTests), skill("go-tests/examples/table.go", "package x")}, ""},
		{"missing SKILL.md", []File{skill("go-tests/notes.md", "hi")}, "go-tests/"},
		{"no front matter", []File{skill("go-tests/SKILL.md", "# Go tests")}, "go-tests/SKILL.md"},
		{"name differs", []File{skill("go-tests/SKILL.md", strings.Replace(goTests, "name: go-tests", "name: tests", 1))}, "go-tests/SKILL.md"},
		{"no description", []File{skill("go-tests/SKILL.md", "---\nname: go-tests\n---\n")}, "go-tests/SKILL.md"},
		{"dot dot", []File{skill("go-tests/../x/SKILL.md", goTests)}, "go-tests/../x/SKILL.md"},
		{"absolute", []File{skill("/etc/passwd", "x")}, "/etc/passwd"},
		{"top-level file", []File{skill("SKILL.md", goTests)}, "SKILL.md"},
		{"hidden directory", []File{skill("go-tests/SKILL.md", goTests), skill("go-tests/.git/config", "x")}, "go-tests/.git/config"},
		{"bad skill name", []File{skill("Go_Tests/SKILL.md", goTests)}, "Go_Tests/SKILL.md"},
		{"binary", []File{skill("go-tests/SKILL.md", goTests), skill("go-tests/x.bin", "\x00\x01")}, "go-tests/x.bin"},
		{"too big", []File{skill("go-tests/SKILL.md", goTests), skill("go-tests/big.txt", strings.Repeat("x", 256<<10+1))}, "go-tests/big.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CheckSkillset(tt.files)
			var se *SkillsetError
			if tt.path == "" {
				if err != nil {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if !errors.As(err, &se) || se.Path != tt.path {
				t.Errorf("error = %v, want one about %q", err, tt.path)
			}
		})
	}
}

func TestFrontMatterAsSkillsWriteIt(t *testing.T) {
	// The shape of real skills: extra keys, quotes, folded descriptions.
	manifest := "---\nname: accessibility\ndescription: >\n  Audit and improve web accessibility.\n  Use for WCAG work.\nlicense: MIT\nmetadata:\n  author: someone\n  version: \"1.1\"\n---\n# Accessibility\n"
	skills, err := CheckSkillset([]File{{Path: "accessibility/SKILL.md", Content: manifest}})
	if err != nil {
		t.Fatal(err)
	}
	if skills[0].Description != "Audit and improve web accessibility. Use for WCAG work." {
		t.Errorf("description = %q", skills[0].Description)
	}
	quoted := "---\nname: \"diagnose-crash\"\ndescription: 'Find why it crashed.'\n---\n"
	if _, err := CheckSkillset([]File{{Path: "diagnose-crash/SKILL.md", Content: quoted}}); err != nil {
		t.Errorf("quoted values: %v", err)
	}
}

func TestReviseAndDelete(t *testing.T) {
	a, _ := NewAgent(1, "vera", vera())
	p := vera()
	p.Title = "Staff engineer"
	if err := a.Revise(1, p); err != nil || a.Revision != 2 || a.Title != "Staff engineer" {
		t.Fatalf("Revise = %v; %+v", err, a)
	}
	if err := a.Revise(1, p); !errors.Is(err, ErrStale) {
		t.Errorf("stale revise error = %v", err)
	}
	if err := a.Delete(1, time.Now()); !errors.Is(err, ErrStale) {
		t.Errorf("stale delete error = %v", err)
	}
	if err := a.Delete(2, time.Now()); err != nil || a.DeletedAt == nil || a.Revision != 3 {
		t.Errorf("Delete = %v; %+v", err, a)
	}
	if err := a.Revise(3, p); !errors.Is(err, ErrDeleted) {
		t.Errorf("revising a deleted agent error = %v", err)
	}
}

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{"Vera": "vera", "Vera the Builder": "vera-the-builder", "  Ivo!! 2 ": "ivo-2", "???": "agent"} {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRecruitAndPull(t *testing.T) {
	origin, _ := NewAgent(1, "vera", vera())
	origin.ID = 10
	copy, err := Recruit(origin, 2, "vera")
	if err != nil {
		t.Fatal(err)
	}
	if copy.OwnerID != 2 || copy.Revision != 1 || *copy.OriginAgentID != 10 || *copy.OriginRevision != 1 {
		t.Fatalf("copy = %+v", copy)
	}
	copy.WorkPriorities["coding"] = 4
	if origin.WorkPriorities["coding"] != 1 {
		t.Error("tuning the copy changed the origin")
	}

	p := vera()
	p.Title = "Principal engineer"
	if err := origin.Revise(1, p); err != nil {
		t.Fatal(err)
	}
	if err := copy.PullOrigin(1, origin); err != nil {
		t.Fatal(err)
	}
	if copy.Title != "Principal engineer" || copy.WorkPriorities["coding"] != 4 || *copy.OriginRevision != 2 || copy.Revision != 2 {
		t.Errorf("after pull: %+v", copy)
	}

	origin.DeletedAt = &time.Time{}
	if _, err := Recruit(origin, 3, "vera"); !errors.Is(err, ErrDeleted) {
		t.Errorf("recruiting a deleted agent error = %v", err)
	}
}

func TestSlugFor(t *testing.T) {
	taken := map[string]bool{"vera": true, "vera-2": true}
	if got := SlugFor("vera", func(s string) bool { return taken[s] }); got != "vera-3" {
		t.Errorf("SlugFor = %q", got)
	}
	if got := SlugFor("ivo", func(s string) bool { return taken[s] }); got != "ivo" {
		t.Errorf("SlugFor = %q", got)
	}
}
