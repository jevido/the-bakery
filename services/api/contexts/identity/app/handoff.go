package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"
)

// HandoffTTL is how long a handoff code can be redeemed.
const HandoffTTL = 60 * time.Second

var ErrHandoffInvalid = errors.New("this sign-in link has expired or was already used; sign in on the website")

// Handoffs stores handoff codes by their hash. Take marks the code used and
// returns its member only when it is unused and not expired; two takes of
// the same code never both succeed.
type Handoffs interface {
	Add(ctx context.Context, codeHash string, memberID uint64, expiresAt time.Time) error
	Take(ctx context.Context, codeHash string, now time.Time) (memberID uint64, ok bool, err error)
}

// CreateHandoff makes a one-time code that signs memberID in on the website.
func (s *Service) CreateHandoff(ctx context.Context, memberID uint64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	code := base64.RawURLEncoding.EncodeToString(raw)
	if err := s.handoffs.Add(ctx, hashCode(code), memberID, time.Now().Add(HandoffTTL)); err != nil {
		return "", err
	}
	return code, nil
}

// RedeemHandoff uses up a code and returns the member it signs in.
func (s *Service) RedeemHandoff(ctx context.Context, code string) (uint64, error) {
	if code == "" {
		return 0, ErrHandoffInvalid
	}
	memberID, ok, err := s.handoffs.Take(ctx, hashCode(code), time.Now())
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrHandoffInvalid
	}
	return memberID, nil
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
