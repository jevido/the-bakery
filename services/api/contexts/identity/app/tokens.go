package app

import (
	"context"
	"errors"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/identity/domain"
)

var (
	ErrTokenNotFound = errors.New("no such token")
	ErrTokenInvalid  = errors.New("this token is not valid")
)

// Tokens stores personal tokens by hash.
type Tokens interface {
	Add(ctx context.Context, t domain.PersonalToken) (domain.PersonalToken, error)
	OfMember(ctx context.Context, memberID uint64) ([]domain.PersonalToken, error)
	ByID(ctx context.Context, id uint64) (domain.PersonalToken, bool, error)
	ByHash(ctx context.Context, hash string) (domain.PersonalToken, bool, error)
	Revoke(ctx context.Context, t domain.PersonalToken) error
	Touch(ctx context.Context, id uint64, at time.Time) error
}

// Secrets makes personal token secrets.
type Secrets interface {
	NewSecret() (string, error)
}

// CreatePersonalToken makes a token for memberID and returns it with its
// secret, which is never available again.
func (s *Service) CreatePersonalToken(ctx context.Context, memberID uint64, name string) (domain.PersonalToken, string, error) {
	secret, err := s.secrets.NewSecret()
	if err != nil {
		return domain.PersonalToken{}, "", err
	}
	t, _, err := domain.NewPersonalToken(memberID, name, secret, time.Now())
	if err != nil {
		return domain.PersonalToken{}, "", err
	}
	t, err = s.tokens.Add(ctx, t)
	if err != nil {
		return domain.PersonalToken{}, "", err
	}
	s.auditLog().Record(ctx, AuditRecord{ActorID: memberID, Action: "personal_token.created", TargetKind: "personal_token", TargetID: t.ID, Meta: map[string]any{"name": t.Name}})
	return t, secret, nil
}

func (s *Service) ListPersonalTokens(ctx context.Context, memberID uint64) ([]domain.PersonalToken, error) {
	return s.tokens.OfMember(ctx, memberID)
}

// RevokePersonalToken revokes one of memberID's tokens. Someone else's token
// looks like it does not exist.
func (s *Service) RevokePersonalToken(ctx context.Context, memberID, tokenID uint64) error {
	t, found, err := s.tokens.ByID(ctx, tokenID)
	if err != nil {
		return err
	}
	if !found || t.MemberID != memberID {
		return ErrTokenNotFound
	}
	t.Revoke(time.Now())
	if err := s.tokens.Revoke(ctx, t); err != nil {
		return err
	}
	s.auditLog().Record(ctx, AuditRecord{ActorID: memberID, Action: "personal_token.revoked", TargetKind: "personal_token", TargetID: t.ID, Meta: map[string]any{"name": t.Name}})
	return nil
}

// VerifyPersonalToken returns the member a personal token belongs to, or
// ErrTokenInvalid for an unknown or revoked one.
func (s *Service) VerifyPersonalToken(ctx context.Context, secret string) (uint64, error) {
	t, found, err := s.tokens.ByHash(ctx, domain.HashSecret(secret))
	if err != nil {
		return 0, err
	}
	if !found || !t.Authenticates() {
		return 0, ErrTokenInvalid
	}
	if now := time.Now(); t.NeedsTouch(now) {
		// Last-used is a convenience; a failed write must not refuse the request.
		_ = s.tokens.Touch(ctx, t.ID, now)
	}
	return t.MemberID, nil
}
