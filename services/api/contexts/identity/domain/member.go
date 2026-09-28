// Package domain holds the identity model: a Member and the rules its
// credentials must follow. It imports nothing outside the standard library.
package domain

import (
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"
)

const (
	displayNameMax    = 60
	passwordMinLength = 8
)

var (
	ErrInvalidEmail       = errors.New("email is not a valid address")
	ErrInvalidDisplayName = errors.New("display name must be 1 to 60 characters")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
)

// Member is a person with an account.
type Member struct {
	ID           uint64
	Email        string
	DisplayName  string
	PasswordHash string
	// PortraitSeed draws the member's generated portrait. Presentation
	// only: no rule depends on it, and a re-roll replaces it.
	PortraitSeed string
}

// PasswordHasher turns a password into a hash and checks one against it.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Check(password, hash string) bool
}

// NewMember creates a Member that satisfies every identity invariant. The
// password is hashed here and never kept in plain text.
func NewMember(email, displayName, password string, hasher PasswordHasher) (Member, error) {
	email, err := NormaliseEmail(email)
	if err != nil {
		return Member{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if n := utf8.RuneCountInString(displayName); n < 1 || n > displayNameMax {
		return Member{}, ErrInvalidDisplayName
	}
	if utf8.RuneCountInString(password) < passwordMinLength {
		return Member{}, ErrPasswordTooShort
	}
	hash, err := hasher.Hash(password)
	if err != nil {
		return Member{}, err
	}
	return Member{Email: email, DisplayName: displayName, PasswordHash: hash}, nil
}

// NormaliseEmail trims and lowercases an email address, rejecting anything
// that is not a bare address (no "Name <addr>" forms).
func NormaliseEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email, "@") {
		return "", ErrInvalidEmail
	}
	return email, nil
}

// PasswordMatches reports whether password is this member's password.
func (m Member) PasswordMatches(password string, hasher PasswordHasher) bool {
	return hasher.Check(password, m.PasswordHash)
}
