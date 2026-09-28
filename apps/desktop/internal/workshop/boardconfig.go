// Package workshop turns an agent, a task and this machine's board config
// into a Claude CLI run in its own git worktree. Everything here stays on the
// machine; runs are reported to the API by the caller.
package workshop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	configName = "board.toml"
	// ReleaseAPIHost is the API release builds talk to. Its boards live in
	// <base>/boards; any other API's in <base>/boards-<host>, so a dev
	// board 1 never picks up the settings of prod's board 1.
	ReleaseAPIHost = "api.bakery.jevido.app"
)

// BoardConfig is board.toml: how this machine works one board.
type BoardConfig struct {
	// Repo is the git repository runs work in; empty means not linked.
	Repo       string `toml:"repo" json:"repo"`
	BaseBranch string `toml:"base_branch" json:"base_branch"`
	// WorktreeRoot is where run worktrees go; empty means
	// <repo>/../.bakery-worktrees/<repo name>.
	WorktreeRoot      string  `toml:"worktree_root" json:"worktree_root"`
	MaxConcurrentRuns int     `toml:"max_concurrent_runs" json:"max_concurrent_runs"`
	MaxBudgetUSD      float64 `toml:"max_budget_usd" json:"max_budget_usd"`
	// FinishColumn is the column a finished run moves its task to; empty
	// leaves the task where it is.
	FinishColumn string `toml:"finish_column" json:"finish_column"`
	// IsolateUserSettings runs Claude with --setting-sources project,local,
	// so the member's own ~/.claude skills, hooks and plugins stay out.
	IsolateUserSettings bool `toml:"isolate_user_settings" json:"isolate_user_settings"`

	// The colony: which agents take work here by themselves, from which
	// column, how many at fast speed, and the speed (paused, normal, fast).
	ReadyColumn           string   `toml:"ready_column" json:"ready_column"`
	Agents                []string `toml:"agents" json:"agents"`
	MaxConcurrentRunsFast int      `toml:"max_concurrent_runs_fast" json:"max_concurrent_runs_fast"`
	Speed                 string   `toml:"speed" json:"speed"`
}

// The time controls.
const (
	SpeedPaused = "paused"
	SpeedNormal = "normal"
	SpeedFast   = "fast"
)

// RunLimit is how many runs the board may have going at its speed.
func (c BoardConfig) RunLimit() int {
	switch c.Speed {
	case SpeedFast:
		return c.MaxConcurrentRunsFast
	case SpeedNormal:
		return c.MaxConcurrentRuns
	}
	return 0
}

// DefaultBoardConfig is what a board has before this machine saved one.
func DefaultBoardConfig() BoardConfig {
	return BoardConfig{
		BaseBranch:          "main",
		MaxConcurrentRuns:   2,
		MaxBudgetUSD:        2,
		FinishColumn:        "Review",
		IsolateUserSettings: true,

		ReadyColumn:           "To do",
		Agents:                []string{},
		MaxConcurrentRunsFast: 4,
		Speed:                 SpeedPaused,
	}
}

// Linked reports whether a repository is set. Validate says whether it is
// a usable one.
func (c BoardConfig) Linked() bool { return c.Repo != "" }

// Worktrees is where this board's run worktrees go.
func (c BoardConfig) Worktrees() string {
	if c.WorktreeRoot != "" {
		return c.WorktreeRoot
	}
	repo := filepath.Clean(c.Repo)
	return filepath.Join(filepath.Dir(repo), ".bakery-worktrees", filepath.Base(repo))
}

// ConfigError names the field that is wrong, for the settings panel.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string { return e.Message }

// Validate checks the config against this machine: the repo is a git
// repository and the base branch exists in it.
func (c BoardConfig) Validate(ctx context.Context) error {
	if c.MaxConcurrentRuns < 1 || c.MaxConcurrentRuns > 8 {
		return &ConfigError{"max_concurrent_runs", "at most 1 to 8 runs at once"}
	}
	if c.MaxBudgetUSD <= 0 {
		return &ConfigError{"max_budget_usd", "the budget per run must be more than $0"}
	}
	if c.MaxConcurrentRunsFast < c.MaxConcurrentRuns || c.MaxConcurrentRunsFast > 8 {
		return &ConfigError{"max_concurrent_runs_fast", "at fast speed, as many runs as at normal speed up to 8"}
	}
	switch c.Speed {
	case SpeedPaused, SpeedNormal, SpeedFast:
	default:
		return &ConfigError{"speed", "speed is paused, normal or fast"}
	}
	if !c.Linked() {
		return nil
	}
	if !filepath.IsAbs(c.Repo) {
		return &ConfigError{"repo", "the repository path must be absolute"}
	}
	if st, err := os.Stat(c.Repo); err != nil || !st.IsDir() {
		return &ConfigError{"repo", fmt.Sprintf("%s is not a folder", c.Repo)}
	}
	if _, err := git(ctx, c.Repo, "rev-parse", "--show-toplevel"); err != nil {
		return &ConfigError{"repo", fmt.Sprintf("%s is not a git repository", c.Repo)}
	}
	if strings.TrimSpace(c.BaseBranch) == "" {
		return &ConfigError{"base_branch", "name the branch runs start from"}
	}
	if _, err := git(ctx, c.Repo, "rev-parse", "--verify", "--quiet", c.BaseBranch+"^{commit}"); err != nil {
		return &ConfigError{"base_branch", fmt.Sprintf("the repository has no branch %s", c.BaseBranch)}
	}
	if c.WorktreeRoot != "" && !filepath.IsAbs(c.WorktreeRoot) {
		return &ConfigError{"worktree_root", "the worktree folder must be an absolute path"}
	}
	return nil
}

// Boards finds board config folders under one config base
// (~/.config/the-bakery on Linux) for one API host.
type Boards struct {
	root string
}

// NewBoards keeps the release API's boards in <base>/boards and another
// API's in <base>/boards-<host>.
func NewBoards(base, apiHost string) *Boards {
	name := "boards"
	if apiHost != ReleaseAPIHost {
		name += "-" + strings.NewReplacer(":", "_", "/", "_", "\\", "_").Replace(apiHost)
	}
	return &Boards{root: filepath.Join(base, name)}
}

// Root is the folder that holds every board's config folder.
func (b *Boards) Root() string { return b.root }

// Dir is the config folder of one board.
func (b *Boards) Dir(boardID uint64) string {
	return filepath.Join(b.root, strconv.FormatUint(boardID, 10))
}

// Load reads a board's config, or the defaults when this machine has none.
func (b *Boards) Load(boardID uint64) (BoardConfig, error) {
	cfg := DefaultBoardConfig()
	raw, err := os.ReadFile(filepath.Join(b.Dir(boardID), configName))
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", filepath.Join(b.Dir(boardID), configName), err)
	}
	return cfg, nil
}

// Save writes a board's config (atomically) and makes the skills/ and runs/
// folders next to it. It does not validate; the caller does.
func (b *Boards) Save(boardID uint64, cfg BoardConfig) error {
	dir := b.Dir(boardID)
	for _, d := range []string{dir, filepath.Join(dir, "skills"), filepath.Join(dir, "runs")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	var buf bytes.Buffer
	buf.WriteString("# How this machine works this board. It never leaves the machine.\n")
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return err
	}
	tmp := filepath.Join(dir, configName+".tmp")
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, configName))
}

// git runs git in dir and returns its trimmed output.
func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errOut.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(out.String()), nil
}
