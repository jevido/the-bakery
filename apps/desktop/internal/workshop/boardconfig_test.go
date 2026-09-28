package workshop

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// gitRepo makes a repository with one commit on main.
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.name=Test", "-c", "user.email=test@bakery.test", "commit", "-q", "--allow-empty", "-m", "first"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func TestValidate(t *testing.T) {
	repo := gitRepo(t)
	plain := t.TempDir()
	with := func(f func(*BoardConfig)) BoardConfig {
		c := DefaultBoardConfig()
		c.Repo = repo
		f(&c)
		return c
	}
	tests := []struct {
		name  string
		cfg   BoardConfig
		field string // "" means valid
	}{
		{"linked repo", with(func(*BoardConfig) {}), ""},
		{"not linked", DefaultBoardConfig(), ""},
		{"not a git repo", with(func(c *BoardConfig) { c.Repo = plain }), "repo"},
		{"missing folder", with(func(c *BoardConfig) { c.Repo = filepath.Join(plain, "gone") }), "repo"},
		{"relative path", with(func(c *BoardConfig) { c.Repo = "colony-site" }), "repo"},
		{"unknown branch", with(func(c *BoardConfig) { c.BaseBranch = "develop" }), "base_branch"},
		{"empty branch", with(func(c *BoardConfig) { c.BaseBranch = " " }), "base_branch"},
		{"no runs", with(func(c *BoardConfig) { c.MaxConcurrentRuns = 0 }), "max_concurrent_runs"},
		{"too many runs", with(func(c *BoardConfig) { c.MaxConcurrentRuns = 9 }), "max_concurrent_runs"},
		{"free runs", with(func(c *BoardConfig) { c.MaxBudgetUSD = 0 }), "max_budget_usd"},
		{"relative worktree root", with(func(c *BoardConfig) { c.WorktreeRoot = "trees" }), "worktree_root"},
		{"fast below normal", with(func(c *BoardConfig) { c.MaxConcurrentRunsFast = 1 }), "max_concurrent_runs_fast"},
		{"made-up speed", with(func(c *BoardConfig) { c.Speed = "ludicrous" }), "speed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate(t.Context())
			if tt.field == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			var ce *ConfigError
			if !errors.As(err, &ce) || ce.Field != tt.field {
				t.Fatalf("Validate() = %v, want an error on %s", err, tt.field)
			}
		})
	}
}

func TestLoadGivesDefaults(t *testing.T) {
	b := NewBoards(t.TempDir(), ReleaseAPIHost)
	cfg, err := b.Load(7)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, DefaultBoardConfig()) {
		t.Fatalf("Load() = %+v, want the defaults", cfg)
	}
	if cfg.Linked() {
		t.Fatal("a board without config is linked")
	}
}

func TestSaveAndLoad(t *testing.T) {
	base := t.TempDir()
	b := NewBoards(base, "127.0.0.1:4810")
	want := BoardConfig{
		Repo: "/home/ada/Projects/colony-site", BaseBranch: "trunk", WorktreeRoot: "/tmp/trees",
		MaxConcurrentRuns: 3, MaxBudgetUSD: 1.5, FinishColumn: "", IsolateUserSettings: false,
		ReadyColumn: "Ready", Agents: []string{"vera", "ivo"}, MaxConcurrentRunsFast: 5, Speed: SpeedFast,
		ExtraAllowedTools: []string{"Bash(curl:*)"},
	}
	if err := b.Save(12, want); err != nil {
		t.Fatal(err)
	}
	got, err := b.Load(12)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
	dir := filepath.Join(base, "boards-127.0.0.1_4810", "12")
	for _, p := range []string{"board.toml", "skills", "runs"} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	if b.Dir(12) != dir {
		t.Errorf("Dir() = %s, want %s", b.Dir(12), dir)
	}
}

func TestReleaseBoardsAreUnsuffixed(t *testing.T) {
	base := t.TempDir()
	if got, want := NewBoards(base, ReleaseAPIHost).Dir(1), filepath.Join(base, "boards", "1"); got != want {
		t.Fatalf("Dir() = %s, want %s", got, want)
	}
}

func TestWorktreesDefault(t *testing.T) {
	c := BoardConfig{Repo: "/home/ada/Projects/colony-site/"}
	if got, want := c.Worktrees(), "/home/ada/Projects/.bakery-worktrees/colony-site"; got != want {
		t.Fatalf("Worktrees() = %s, want %s", got, want)
	}
}
