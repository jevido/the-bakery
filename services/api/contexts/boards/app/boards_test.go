package app

import (
	"errors"
	"testing"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

func ptr(v uint64) *uint64 { return &v }

func TestNeighbours(t *testing.T) {
	col := []domain.Task{{ID: 1, Position: "a0"}, {ID: 2, Position: "a1"}, {ID: 3, Position: "a2"}}
	tests := []struct {
		name                string
		moved               uint64
		after, before       *uint64
		wantAbove, wantBelo string
		wantErr             error
	}{
		{"bottom by default", 9, nil, nil, "a2", "", nil},
		{"bottom skips self", 3, nil, nil, "a1", "", nil},
		{"after first", 9, ptr(1), nil, "a0", "a1", nil},
		{"after last", 9, ptr(3), nil, "a2", "", nil},
		{"before first", 3, nil, ptr(1), "", "a0", nil},
		{"before second", 9, nil, ptr(2), "a0", "a1", nil},
		{"between", 9, ptr(1), ptr(2), "a0", "a1", nil},
		{"unknown after", 9, ptr(7), nil, "", "", ErrInvalidNeighbour},
		{"self as neighbour", 2, ptr(2), nil, "", "", ErrInvalidNeighbour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			above, below, err := neighbours(col, tt.moved, tt.after, tt.before)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if above != tt.wantAbove || below != tt.wantBelo {
				t.Errorf("got (%q, %q), want (%q, %q)", above, below, tt.wantAbove, tt.wantBelo)
			}
		})
	}
}
