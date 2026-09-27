package domain

import (
	"errors"
	"strings"
	"testing"
)

// plainHasher stands in for the real hasher: the domain only needs the
// round trip to work.
type plainHasher struct{}

func (plainHasher) Hash(p string) (string, error) { return "hashed:" + p, nil }
func (plainHasher) Check(p, h string) bool        { return h == "hashed:"+p }

func TestNewMember(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		displayName string
		password    string
		wantErr     error
		wantEmail   string
	}{
		{"valid", "ada@bakery.test", "Ada", "firstlanding", nil, "ada@bakery.test"},
		{"email lowercased and trimmed", "  Ada@Bakery.TEST ", "Ada", "firstlanding", nil, "ada@bakery.test"},
		{"email without at", "ada.bakery.test", "Ada", "firstlanding", ErrInvalidEmail, ""},
		{"email with name", "Ada <ada@bakery.test>", "Ada", "firstlanding", ErrInvalidEmail, ""},
		{"empty email", "", "Ada", "firstlanding", ErrInvalidEmail, ""},
		{"empty display name", "ada@bakery.test", "   ", "firstlanding", ErrInvalidDisplayName, ""},
		{"display name 60 runes", "ada@bakery.test", strings.Repeat("é", 60), "firstlanding", nil, "ada@bakery.test"},
		{"display name 61 runes", "ada@bakery.test", strings.Repeat("é", 61), "firstlanding", ErrInvalidDisplayName, ""},
		{"password too short", "ada@bakery.test", "Ada", "seven77", ErrPasswordTooShort, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMember(tt.email, tt.displayName, tt.password, plainHasher{})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewMember() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if m.Email != tt.wantEmail {
				t.Errorf("Email = %q, want %q", m.Email, tt.wantEmail)
			}
			if m.PasswordHash == tt.password {
				t.Error("password stored in plain text")
			}
			if !m.PasswordMatches(tt.password, plainHasher{}) {
				t.Error("PasswordMatches() = false for the member's own password")
			}
		})
	}
}
