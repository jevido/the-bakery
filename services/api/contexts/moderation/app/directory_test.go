package app

import (
	"testing"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

func TestMergeNewestFirst(t *testing.T) {
	e := func(ids ...uint64) []domain.AuditEntry {
		out := make([]domain.AuditEntry, len(ids))
		for i, id := range ids {
			out[i] = domain.AuditEntry{ID: id}
		}
		return out
	}
	got := mergeNewestFirst(e(9, 5, 2), e(8, 5, 1), 10)
	want := []uint64{9, 8, 5, 2, 1}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("entry %d is %d, want %d", i, got[i].ID, id)
		}
	}
	if n := len(mergeNewestFirst(e(3, 2, 1), e(6, 5, 4), 4)); n != 4 {
		t.Fatalf("limit ignored: %d entries", n)
	}
}
