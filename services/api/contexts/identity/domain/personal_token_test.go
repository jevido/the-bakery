package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewPersonalToken(t *testing.T) {
	now := time.Now()
	secret := PersonalTokenPrefix + strings.Repeat("a", 32)
	tok, ev, err := NewPersonalToken(7, "  laptop claude ", secret, now)
	if err != nil {
		t.Fatal(err)
	}
	if tok.Name != "laptop claude" || ev.Name != "laptop claude" {
		t.Errorf("name = %q", tok.Name)
	}
	if tok.Hash == secret || tok.Hash != HashSecret(secret) {
		t.Error("the secret must be stored only as its hash")
	}
	if _, _, err := NewPersonalToken(7, "", secret, now); !errors.Is(err, ErrInvalidTokenName) {
		t.Errorf("empty name error = %v", err)
	}
	if _, _, err := NewPersonalToken(7, "x", "nope", now); !errors.Is(err, ErrInvalidSecret) {
		t.Errorf("bad secret error = %v", err)
	}
}

func TestPersonalTokenRevokeAndTouch(t *testing.T) {
	now := time.Now()
	tok := PersonalToken{ID: 1, MemberID: 7}
	if !tok.Authenticates() || !tok.NeedsTouch(now) {
		t.Fatal("a fresh token authenticates and needs a first touch")
	}
	tok.LastUsedAt = &now
	if tok.NeedsTouch(now.Add(30 * time.Second)) {
		t.Error("touched again within a minute")
	}
	if !tok.NeedsTouch(now.Add(61 * time.Second)) {
		t.Error("not touched after a minute")
	}
	tok.Revoke(now)
	if tok.Authenticates() {
		t.Error("a revoked token authenticates")
	}
}
