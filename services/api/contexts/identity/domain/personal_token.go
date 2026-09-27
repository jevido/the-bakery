package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// PersonalTokenPrefix starts every personal token, so they are easy to spot
// (and to tell from the desktop's session JWT).
const PersonalTokenPrefix = "bky_"

const tokenNameMax = 60

var (
	ErrInvalidTokenName = errors.New("token name must be 1 to 60 characters")
	ErrInvalidSecret    = errors.New("invalid token secret")
)

// PersonalToken is a long-lived secret a member made for a tool. Only its
// hash is kept.
type PersonalToken struct {
	ID         uint64
	MemberID   uint64
	Name       string
	Hash       string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

// PersonalTokenCreated and PersonalTokenRevoked are announced when a token is
// made or revoked.
type PersonalTokenCreated struct {
	TokenID, MemberID uint64
	Name              string
}

type PersonalTokenRevoked struct{ TokenID, MemberID uint64 }

// NewPersonalToken makes a token for memberID from a freshly generated
// secret; the secret itself is not kept.
func NewPersonalToken(memberID uint64, name, secret string, now time.Time) (PersonalToken, PersonalTokenCreated, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > tokenNameMax {
		return PersonalToken{}, PersonalTokenCreated{}, ErrInvalidTokenName
	}
	if !strings.HasPrefix(secret, PersonalTokenPrefix) || len(secret) != len(PersonalTokenPrefix)+32 {
		return PersonalToken{}, PersonalTokenCreated{}, ErrInvalidSecret
	}
	t := PersonalToken{MemberID: memberID, Name: name, Hash: HashSecret(secret), CreatedAt: now}
	return t, PersonalTokenCreated{MemberID: memberID, Name: name}, nil
}

// HashSecret is how a personal token's secret is stored and looked up.
func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// Authenticates reports whether the token may still be used.
func (t PersonalToken) Authenticates() bool { return t.RevokedAt == nil }

// Revoke ends the token; revoking twice changes nothing.
func (t *PersonalToken) Revoke(now time.Time) PersonalTokenRevoked {
	if t.RevokedAt == nil {
		t.RevokedAt = &now
	}
	return PersonalTokenRevoked{TokenID: t.ID, MemberID: t.MemberID}
}

// NeedsTouch reports whether last-used should be written again: at most
// once a minute, so a busy tool does not write on every request.
func (t PersonalToken) NeedsTouch(now time.Time) bool {
	return t.LastUsedAt == nil || now.Sub(*t.LastUsedAt) >= time.Minute
}
