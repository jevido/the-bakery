package workshop

import (
	"slices"
	"testing"
)

func TestPickNext(t *testing.T) {
	vera := AgentState{Slug: "vera", AgentID: 1, Priorities: map[string]int{"coding": 1, "testing": 2}}
	ivo := AgentState{Slug: "ivo", AgentID: 2, Priorities: map[string]int{"research": 1, "coding": 2}}
	pip := AgentState{Slug: "pip", AgentID: 3, Priorities: map[string]int{"coding": 1}}
	busy := func(a AgentState) AgentState { a.Busy = true; return a }
	coding := func(id uint64) TaskState { return TaskState{ID: id, WorkType: "coding"} }
	tests := []struct {
		name    string
		agents  []AgentState
		tasks   []TaskState
		running int
		limit   int
		want    []string // "task:agent"
	}{
		{"each takes its first priority",
			[]AgentState{vera, ivo}, []TaskState{coding(1), {ID: 2, WorkType: "research"}}, 0, 2,
			[]string{"1:vera", "2:ivo"}},
		{"priority 1 before someone else's priority 2",
			[]AgentState{ivo, vera}, []TaskState{coding(1)}, 0, 2,
			[]string{"1:vera"}},
		{"board order within a priority",
			[]AgentState{vera}, []TaskState{coding(5), coding(3)}, 0, 1,
			[]string{"5:vera"}},
		{"off never takes it",
			[]AgentState{ivo}, []TaskState{{ID: 1, WorkType: "writing"}}, 0, 2,
			nil},
		{"no work type, nobody takes it",
			[]AgentState{vera}, []TaskState{{ID: 1}}, 0, 2,
			nil},
		{"forbidden and taken are skipped",
			[]AgentState{vera}, []TaskState{{ID: 1, WorkType: "coding", Forbidden: true}, {ID: 2, WorkType: "coding", Taken: true}, coding(3)}, 0, 2,
			[]string{"3:vera"}},
		{"prioritized goes to its agent first, over its own work",
			[]AgentState{vera, ivo}, []TaskState{{ID: 1, WorkType: "research"}, {ID: 2, WorkType: "writing", PrioritizedAgentID: 2}}, 0, 2,
			[]string{"2:ivo"}},
		{"prioritized for someone else is left",
			[]AgentState{vera}, []TaskState{{ID: 1, WorkType: "coding", PrioritizedAgentID: 2}}, 0, 2,
			nil},
		{"limit counts running runs",
			[]AgentState{vera, pip}, []TaskState{coding(1), coding(2)}, 1, 2,
			[]string{"1:vera"}},
		{"at the limit, nothing",
			[]AgentState{vera}, []TaskState{coding(1)}, 2, 2,
			nil},
		{"busy agents wait",
			[]AgentState{busy(vera), pip}, []TaskState{coding(1), coding(2)}, 0, 4,
			[]string{"1:pip"}},
		{"two agents, same work type, one task each",
			[]AgentState{vera, pip}, []TaskState{coding(1), coding(2), coding(3)}, 0, 4,
			[]string{"1:vera", "2:pip"}},
		{"paused",
			[]AgentState{vera}, []TaskState{coding(1)}, 0, 0,
			nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, a := range PickNext(tt.agents, tt.tasks, tt.running, tt.limit) {
				got = append(got, itoa(a.TaskID)+":"+a.AgentSlug)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("PickNext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

func TestRunLimit(t *testing.T) {
	c := DefaultBoardConfig()
	if c.RunLimit() != 0 {
		t.Error("a paused board may run")
	}
	c.Speed = SpeedNormal
	if c.RunLimit() != 2 {
		t.Errorf("normal = %d", c.RunLimit())
	}
	c.Speed = SpeedFast
	if c.RunLimit() != 4 {
		t.Errorf("fast = %d", c.RunLimit())
	}
}
