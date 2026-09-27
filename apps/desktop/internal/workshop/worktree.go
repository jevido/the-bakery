package workshop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// skillsExclude keeps the skills a run places in its worktree out of git.
// It goes in the repository's common info/exclude, which every worktree
// reads, so one line covers every run and the repo's own committed skills
// stay visible.
const skillsExclude = "/.claude/skills/bakery-*/"

// Worktree is the checkout one task's runs happen in.
type Worktree struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
	// Base is the branch the worktree's branch started from.
	Base string `json:"base"`
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Slug makes a branch- and folder-safe name from a task title: lowercase
// letters, digits and hyphens, at most 40 characters.
func Slug(title string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if len(s) > 40 {
		s = strings.TrimRight(s[:40], "-")
	}
	if s == "" {
		s = "task"
	}
	return s
}

// BranchPrefix is how every branch of one task starts: bakery/<task-id>-.
func BranchPrefix(taskID uint64) string {
	return "bakery/" + strconv.FormatUint(taskID, 10) + "-"
}

// WorktreeBusyError is a task whose branch is checked out somewhere that is
// not the task's worktree.
type WorktreeBusyError struct{ Path string }

func (e *WorktreeBusyError) Error() string {
	return "this task already has a worktree at " + e.Path
}

// PrepareWorktree gives a task its worktree on branch
// bakery/<task-id>-<slug>: the one it already has, or a new one from the
// base branch. A task keeps its first branch when its title changes later.
func PrepareWorktree(ctx context.Context, cfg BoardConfig, taskID uint64, title string) (Worktree, error) {
	if !cfg.Linked() {
		return Worktree{}, errors.New("this board is not linked to a repository on this machine")
	}
	repo := cfg.Repo
	prefix := BranchPrefix(taskID)

	// A worktree this task already has, whatever its title was then.
	trees, err := worktrees(ctx, repo)
	if err != nil {
		return Worktree{}, err
	}
	for _, t := range trees {
		if strings.HasPrefix(t.Branch, prefix) {
			if _, err := os.Stat(t.Path); err != nil {
				// Deleted by hand: let git forget it and start over.
				if _, err := git(ctx, repo, "worktree", "prune"); err != nil {
					return Worktree{}, err
				}
				break
			}
			return Worktree{Path: t.Path, Branch: t.Branch, Base: cfg.BaseBranch}, ensureExclude(ctx, repo)
		}
	}

	branch, err := existingBranch(ctx, repo, prefix)
	if err != nil {
		return Worktree{}, err
	}
	if branch == "" {
		branch = prefix + Slug(title)
	}
	path := filepath.Join(cfg.Worktrees(), strings.TrimPrefix(branch, "bakery/"))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Worktree{}, err
	}
	if _, err := git(ctx, repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		_, err = git(ctx, repo, "worktree", "add", path, branch)
		if err != nil {
			return Worktree{}, fmt.Errorf("making the worktree: %w", err)
		}
	} else if _, err := git(ctx, repo, "worktree", "add", "-b", branch, path, cfg.BaseBranch); err != nil {
		return Worktree{}, fmt.Errorf("making the worktree: %w", err)
	}
	return Worktree{Path: path, Branch: branch, Base: cfg.BaseBranch}, ensureExclude(ctx, repo)
}

// RemoveWorktree deletes a worktree's folder and keeps its branch.
func RemoveWorktree(ctx context.Context, repo, path string) error {
	if _, err := git(ctx, repo, "worktree", "remove", "--force", path); err != nil {
		return err
	}
	_, err := git(ctx, repo, "worktree", "prune")
	return err
}

type worktreeEntry struct{ Path, Branch string }

// worktrees lists the repository's worktrees with the branch each has out.
func worktrees(ctx context.Context, repo string) ([]worktreeEntry, error) {
	out, err := git(ctx, repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var list []worktreeEntry
	var cur worktreeEntry
	for line := range strings.SplitSeq(out+"\n", "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			cur = worktreeEntry{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case line == "":
			if cur.Path != "" {
				list = append(list, cur)
			}
			cur = worktreeEntry{}
		}
	}
	return list, nil
}

// existingBranch finds a branch the task made before, or "".
func existingBranch(ctx context.Context, repo, prefix string) (string, error) {
	out, err := git(ctx, repo, "for-each-ref", "--format=%(refname:short)", "refs/heads/"+prefix+"*")
	if err != nil {
		return "", err
	}
	first, _, _ := strings.Cut(out, "\n")
	return strings.TrimSpace(first), nil
}

// ensureExclude writes the skills line into the common info/exclude once.
func ensureExclude(ctx context.Context, repo string) error {
	common, err := git(ctx, repo, "rev-parse", "--git-common-dir")
	if err != nil {
		return err
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(repo, common)
	}
	path := filepath.Join(common, "info", "exclude")
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for line := range strings.SplitSeq(string(raw), "\n") {
		if strings.TrimSpace(line) == skillsExclude {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	text := string(raw)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += "# Skills The Bakery places for its runs\n" + skillsExclude + "\n"
	return os.WriteFile(path, []byte(text), 0o644)
}
