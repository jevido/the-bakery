package domain

// Trait is a fixed personality or working-style option. Each adds one line
// to the agent's instructions.
type Trait struct {
	Key         string
	Label       string
	Description string
	Instruction string
}

// Traits is the fixed list, in the order the app shows them.
var Traits = []Trait{
	{"careful", "Careful", "Checks work twice before calling it done.", "Check your work twice before you call it done."},
	{"tidy", "Tidy", "Leaves code cleaner than found, without drive-by rewrites.", "Leave the code a little cleaner than you found it, but make no unrelated rewrites."},
	{"test-first", "Test first", "Writes a failing test before a fix.", "Before fixing a bug, write a test that fails because of it."},
	{"terse", "Terse", "Short summaries, no filler.", "Keep summaries short and leave out filler."},
	{"curious", "Curious", "Reads around the code before changing it.", "Read the code around a change before you make it."},
	{"cautious-with-commands", "Cautious with commands", "Asks before running anything that changes state outside the worktree.", "Ask before running any command that changes state outside your worktree."},
	{"fast-worker", "Fast worker", "Picks the simplest path, with fewer checks.", "Take the simplest path that works and keep checks to the essentials."},
}

// conflictingTraits cannot be combined, like opposed traits in RimWorld.
var conflictingTraits = [][2]string{{"careful", "fast-worker"}}

func traitKnown(key string) bool {
	for _, t := range Traits {
		if t.Key == key {
			return true
		}
	}
	return false
}

// TraitsConflict reports whether a and b cannot be combined.
func TraitsConflict(a, b string) bool {
	for _, c := range conflictingTraits {
		if (c[0] == a && c[1] == b) || (c[0] == b && c[1] == a) {
			return true
		}
	}
	return false
}

// ConflictsOf lists the traits that cannot go with key.
func ConflictsOf(key string) []string {
	var out []string
	for _, c := range conflictingTraits {
		switch key {
		case c[0]:
			out = append(out, c[1])
		case c[1]:
			out = append(out, c[0])
		}
	}
	return out
}
