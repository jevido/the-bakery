package domain

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// Cases from rocicorp/fractional-indexing's own test suite.
func TestKeyBetween(t *testing.T) {
	tests := []struct {
		a, b, want string
		wantErr    bool
	}{
		{"", "", "a0", false},
		{"", "a0", "Zz", false},
		{"", "Zz", "Zy", false},
		{"a0", "", "a1", false},
		{"a1", "", "a2", false},
		{"a0", "a1", "a0V", false},
		{"a1", "a2", "a1V", false},
		{"a0V", "a1", "a0l", false},
		{"Zz", "a0", "ZzV", false},
		{"Zz", "a1", "a0", false},
		{"", "Y00", "Xzzz", false},
		{"bzz", "", "c000", false},
		{"a0", "a0V", "a0G", false},
		{"a0", "a0G", "a08", false},
		{"b125", "b129", "b127", false},
		{"a0", "a1V", "a1", false},
		{"Zz", "a01", "a0", false},
		{"", "a0V", "a0", false},
		{"", "b999", "b99", false},
		{"", "A00000000000000000000000000", "", true},
		{"", "A000000000000000000000000001", "A000000000000000000000000000V", false},
		{"zzzzzzzzzzzzzzzzzzzzzzzzzzy", "", "zzzzzzzzzzzzzzzzzzzzzzzzzzz", false},
		{"zzzzzzzzzzzzzzzzzzzzzzzzzzz", "", "zzzzzzzzzzzzzzzzzzzzzzzzzzzV", false},
		{"a00", "", "", true},
		{"a00", "a1", "", true},
		{"0", "1", "", true},
		{"a1", "a0", "", true},
	}
	for _, tt := range tests {
		got, err := KeyBetween(tt.a, tt.b)
		if (err != nil) != tt.wantErr {
			t.Errorf("KeyBetween(%q, %q) error = %v, wantErr %v", tt.a, tt.b, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("KeyBetween(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
		}
	}
}

// Random inserts into a list must always produce a key strictly between
// its neighbours, so the list stays sorted.
func TestKeyBetweenKeepsOrder(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	keys := []string{}
	for range 2000 {
		i := r.IntN(len(keys) + 1)
		a, b := "", ""
		if i > 0 {
			a = keys[i-1]
		}
		if i < len(keys) {
			b = keys[i]
		}
		k, err := KeyBetween(a, b)
		if err != nil {
			t.Fatalf("KeyBetween(%q, %q): %v", a, b, err)
		}
		if (a != "" && k <= a) || (b != "" && k >= b) {
			t.Fatalf("KeyBetween(%q, %q) = %q, not between", a, b, k)
		}
		keys = slices.Insert(keys, i, k)
	}
	if !slices.IsSorted(keys) {
		t.Fatal("keys not sorted")
	}
}
