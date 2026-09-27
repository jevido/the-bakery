package workshop

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func linked(t *testing.T) BoardConfig {
	t.Helper()
	cfg := DefaultBoardConfig()
	cfg.Repo = gitRepo(t)
	cfg.WorktreeRoot = filepath.Join(t.TempDir(), "trees")
	return cfg
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestSlug(t *testing.T) {
	tests := map[string]string{
		"Build a research bench":         "build-a-research-bench",
		"  Fix: the Freezer!! (again) ":  "fix-the-freezer-again",
		"Ünïcode only ☃":                 "n-code-only",
		"":                               "task",
		strings.Repeat("abcdefghij ", 6): "abcdefghij-abcdefghij-abcdefghij-abcdefg",
	}
	for in, want := range tests {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPrepareWorktreeNewBranch(t *testing.T) {
	cfg := linked(t)
	wt, err := PrepareWorktree(t.Context(), cfg, 12, "Wall in the freezer")
	if err != nil {
		t.Fatal(err)
	}
	if wt.Branch != "bakery/12-wall-in-the-freezer" {
		t.Errorf("branch = %s", wt.Branch)
	}
	if want := filepath.Join(cfg.WorktreeRoot, "12-wall-in-the-freezer"); wt.Path != want {
		t.Errorf("path = %s, want %s", wt.Path, want)
	}
	if got := gitOut(t, wt.Path, "branch", "--show-current"); got != wt.Branch {
		t.Errorf("checked out %s", got)
	}
}

func TestPrepareWorktreeReuses(t *testing.T) {
	cfg := linked(t)
	first, err := PrepareWorktree(t.Context(), cfg, 3, "Dig a cooler")
	if err != nil {
		t.Fatal(err)
	}
	// Renamed since: still the same worktree and branch.
	again, err := PrepareWorktree(t.Context(), cfg, 3, "Dig a bigger cooler")
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Fatalf("second prepare = %+v, want %+v", again, first)
	}
	// Worktree removed, branch kept: the branch comes back in a new worktree.
	if err := RemoveWorktree(t.Context(), cfg.Repo, first.Path); err != nil {
		t.Fatal(err)
	}
	third, err := PrepareWorktree(t.Context(), cfg, 3, "Something else")
	if err != nil {
		t.Fatal(err)
	}
	if third.Branch != first.Branch {
		t.Fatalf("branch = %s, want %s", third.Branch, first.Branch)
	}
	if _, err := os.Stat(third.Path); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareWorktreeAfterFolderDeletedByHand(t *testing.T) {
	cfg := linked(t)
	first, err := PrepareWorktree(t.Context(), cfg, 4, "Haul steel")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(first.Path); err != nil {
		t.Fatal(err)
	}
	again, err := PrepareWorktree(t.Context(), cfg, 4, "Haul steel")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(again.Path, ".git")); err != nil {
		t.Fatalf("worktree not back: %v", err)
	}
}

func TestPrepareWorktreeNotLinked(t *testing.T) {
	if _, err := PrepareWorktree(t.Context(), DefaultBoardConfig(), 1, "x"); err == nil {
		t.Fatal("a board without a repo made a worktree")
	}
}

func TestExcludeWrittenOnce(t *testing.T) {
	cfg := linked(t)
	for id := uint64(1); id <= 2; id++ {
		if _, err := PrepareWorktree(t.Context(), cfg, id, "task"); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(cfg.Repo, ".git", "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(raw), skillsExclude); n != 1 {
		t.Fatalf("exclude line %d times:\n%s", n, raw)
	}
}

func writeSkill(t *testing.T, dir, name, manifest string) {
	t.Helper()
	d := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Join(d, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "SKILL.md"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "examples", "one.md"), []byte("an example"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleSkills(t *testing.T) {
	cfg := linked(t)
	wt, err := PrepareWorktree(t.Context(), cfg, 5, "Tests")
	if err != nil {
		t.Fatal(err)
	}
	agent, board := t.TempDir(), t.TempDir()
	writeSkill(t, agent, "go-tests", "---\nname: go-tests\ndescription: Write Go tests.\n---\nUse t.Run.\n")
	writeSkill(t, board, "house-style", "---\ndescription: >\n  The house style.\nname: house-style\n---\n")
	// A skill the repo commits itself stays as it is.
	writeSkill(t, filepath.Join(wt.Path, ".claude", "skills"), "repo-own", "---\nname: repo-own\ndescription: Ours.\n---\n")
	gitOut(t, wt.Path, "add", ".claude")
	gitOut(t, wt.Path, "-c", "user.name=T", "-c", "user.email=t@bakery.test", "commit", "-qm", "own skill")
	// Left over from an earlier run.
	writeSkill(t, filepath.Join(wt.Path, ".claude", "skills"), "bakery-vera-gone", "---\nname: x\ndescription: y\n---\n")

	names, err := AssembleSkills(wt, "vera", agent, board)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"bakery-board-house-style", "bakery-vera-go-tests"}; !slices.Equal(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	skills := filepath.Join(wt.Path, ".claude", "skills")
	if _, err := os.Stat(filepath.Join(skills, "bakery-vera-gone")); !errors.Is(err, os.ErrNotExist) {
		t.Error("an earlier run's skill is still there")
	}
	got, _ := os.ReadFile(filepath.Join(skills, "bakery-vera-go-tests", "SKILL.md"))
	if !strings.HasPrefix(string(got), "---\nname: bakery-vera-go-tests\ndescription: Write Go tests.\n---") {
		t.Errorf("SKILL.md = %q", got)
	}
	if _, err := os.Stat(filepath.Join(skills, "bakery-vera-go-tests", "examples", "one.md")); err != nil {
		t.Error("supporting file not copied")
	}
	if _, err := os.Stat(filepath.Join(skills, "repo-own", "SKILL.md")); err != nil {
		t.Error("the repo's own skill went missing")
	}
	if st := gitOut(t, wt.Path, "status", "--porcelain"); st != "" {
		t.Errorf("git status not clean:\n%s", st)
	}
}

func TestAssembleSkillsRefusesBadManifest(t *testing.T) {
	tests := map[string]string{
		"no front matter": "# Just text\n",
		"no name":         "---\ndescription: d\n---\n",
		"no description":  "---\nname: x\n---\n",
		"empty describe":  "---\nname: x\ndescription: \"\"\n---\n",
		"never closed":    "---\nname: x\ndescription: d\n",
	}
	for name, manifest := range tests {
		t.Run(name, func(t *testing.T) {
			wt := Worktree{Path: t.TempDir()}
			agent := t.TempDir()
			writeSkill(t, agent, "bad", manifest)
			_, err := AssembleSkills(wt, "vera", agent, "")
			if err == nil || !strings.Contains(err.Error(), filepath.Join(agent, "bad", "SKILL.md")) {
				t.Fatalf("err = %v, want one naming the SKILL.md", err)
			}
		})
	}
}
