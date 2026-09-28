package workshop

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DiffStats is what `git diff --shortstat` counts.
type DiffStats struct {
	Files     int `json:"files"`
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
}

func (d DiffStats) Empty() bool { return d.Files == 0 && d.Additions == 0 && d.Deletions == 0 }

// Add sums two diffs.
func (d DiffStats) Add(o DiffStats) DiffStats {
	return DiffStats{d.Files + o.Files, d.Additions + o.Additions, d.Deletions + o.Deletions}
}

var shortstatPart = regexp.MustCompile(`(\d+) (file|insertion|deletion)`)

// ParseShortstat reads git's " 3 files changed, 10 insertions(+), 2
// deletions(-)". An empty line is an empty diff.
func ParseShortstat(line string) DiffStats {
	var d DiffStats
	for _, m := range shortstatPart.FindAllStringSubmatch(line, -1) {
		n, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "file":
			d.Files = n
		case "insertion":
			d.Additions = n
		case "deletion":
			d.Deletions = n
		}
	}
	return d
}

// WorktreeDiff is what a run left in its worktree: work committed on the
// branch since the base, and changes it did not commit (untracked files
// counted as files only).
type WorktreeDiff struct {
	Committed   DiffStats `json:"committed"`
	Uncommitted DiffStats `json:"uncommitted"`
}

// Diff counts a worktree's committed and uncommitted changes.
func Diff(ctx context.Context, wt Worktree) (WorktreeDiff, error) {
	committed, err := git(ctx, wt.Path, "diff", "--shortstat", wt.Base+"...HEAD")
	if err != nil {
		return WorktreeDiff{}, err
	}
	uncommitted, err := git(ctx, wt.Path, "diff", "--shortstat", "HEAD")
	if err != nil {
		return WorktreeDiff{}, err
	}
	untracked, err := git(ctx, wt.Path, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return WorktreeDiff{}, err
	}
	out := WorktreeDiff{Committed: ParseShortstat(committed), Uncommitted: ParseShortstat(uncommitted)}
	if untracked != "" {
		out.Uncommitted.Files += len(strings.Split(untracked, "\n"))
	}
	return out, nil
}

// EndStatus is how a run ended, from its process's outcome.
func EndStatus(out Outcome) string {
	switch {
	case out.Stopped:
		return "stopped"
	case out.Result != nil && out.Result.Subtype == "success" && !out.Result.IsError:
		return "succeeded"
	default:
		return "failed"
	}
}

// summaryMax is how much of Claude's final text a run keeps.
const summaryMax = 2000

// Summary is what a run says it did: Claude's final result, or the last
// text it wrote, cut to summaryMax characters.
func Summary(out Outcome) string {
	s := out.LastText
	if out.Result != nil && strings.TrimSpace(out.Result.Text) != "" {
		s = out.Result.Text
	}
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > summaryMax {
		s = string(r[:summaryMax]) + "…"
	}
	return s
}

// RunReport is what the finish comment says about a run.
type RunReport struct {
	AgentName string
	Branch    string
	Status    string
	Diff      WorktreeDiff
	CostUSD   float64
	Summary   string
	// Problem is the last error line of a run that failed or was stopped.
	Problem string
}

var statusWords = map[string]string{
	"succeeded": "done",
	"failed":    "failed",
	"stopped":   "stopped",
}

// FinishComment is the comment a finished run leaves on its task, in
// markdown.
func FinishComment(r RunReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** finished on `%s` — %s", r.AgentName, r.Branch, statusWords[r.Status])
	d := r.Diff.Committed
	fmt.Fprintf(&b, ", +%d −%d in %d %s", d.Additions, d.Deletions, d.Files, plural(d.Files, "file", "files"))
	fmt.Fprintf(&b, ", $%.2f\n", r.CostUSD)
	if u := r.Diff.Uncommitted; !u.Empty() {
		fmt.Fprintf(&b, "\n%s did not commit everything: %d %s with changes left in the worktree.\n",
			r.AgentName, u.Files, plural(u.Files, "file", "files"))
	}
	if r.Status != "succeeded" && strings.TrimSpace(r.Problem) != "" {
		fmt.Fprintf(&b, "\n> %s\n", strings.TrimSpace(r.Problem))
	}
	if s := strings.TrimSpace(r.Summary); s != "" {
		b.WriteString("\n" + s + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
