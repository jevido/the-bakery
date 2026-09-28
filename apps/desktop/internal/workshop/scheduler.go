package workshop

// AgentState is an agent enabled on a board, as the scheduler sees it.
type AgentState struct {
	Slug    string
	AgentID uint64
	// Priorities map a work type key to 1 (first) … 4 (last); missing is off.
	Priorities map[string]int
	// Busy is true while the agent has a run going anywhere on this
	// machine.
	Busy bool
}

// TaskState is a task in the board's ready column, in board order.
type TaskState struct {
	ID       uint64
	WorkType string
	// Taken is true while a claim holds the task, from any machine.
	Taken              bool
	Forbidden          bool
	PrioritizedAgentID uint64
}

// Assignment puts one agent on one task.
type Assignment struct {
	TaskID    uint64
	AgentSlug string
	AgentID   uint64
}

// PickNext hands free tasks to idle agents, like colonists taking jobs:
// a task prioritized for an agent goes to that agent first; then, one
// priority at a time (1 first), each idle agent takes the first free task
// of a work type it has at that priority. An agent never takes work it has
// off, and nothing without a work type unless it is prioritized for it.
// running is how many runs the board has going; at most limit run at once.
func PickNext(agents []AgentState, tasks []TaskState, running, limit int) []Assignment {
	slots := limit - running
	if slots <= 0 {
		return nil
	}
	var out []Assignment
	busy := map[string]bool{}
	taken := map[uint64]bool{}
	for _, a := range agents {
		busy[a.Slug] = a.Busy
	}
	give := func(a AgentState, t TaskState) {
		out = append(out, Assignment{TaskID: t.ID, AgentSlug: a.Slug, AgentID: a.AgentID})
		busy[a.Slug], taken[t.ID] = true, true
		slots--
	}
	free := func(t TaskState) bool { return !t.Taken && !t.Forbidden && !taken[t.ID] }

	for _, t := range tasks {
		if slots == 0 {
			return out
		}
		if t.PrioritizedAgentID == 0 || !free(t) {
			continue
		}
		for _, a := range agents {
			if a.AgentID == t.PrioritizedAgentID && !busy[a.Slug] {
				give(a, t)
				break
			}
		}
	}
	for p := 1; p <= 4 && slots > 0; p++ {
		for _, a := range agents {
			if slots == 0 {
				break
			}
			if busy[a.Slug] {
				continue
			}
			for _, t := range tasks {
				if free(t) && t.PrioritizedAgentID == 0 && t.WorkType != "" && a.Priorities[t.WorkType] == p {
					give(a, t)
					break
				}
			}
		}
	}
	return out
}
