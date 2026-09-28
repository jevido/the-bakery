// Package app holds the identity use cases: register, log in and look up
// the current member.
package app

import (
	"context"
	"errors"

	"github.com/jevido/the-bakery/services/api/contexts/identity/domain"
)

var (
	ErrEmailTaken     = errors.New("an account with this email already exists")
	ErrBadCredentials = errors.New("email or password is incorrect")
	ErrMemberNotFound = errors.New("member not found")
)

// Members stores members. Add returns ErrEmailTaken when the email is in use.
type Members interface {
	Add(ctx context.Context, m domain.Member) (domain.Member, error)
	ByEmail(ctx context.Context, email string) (domain.Member, bool, error)
	ByID(ctx context.Context, id uint64) (domain.Member, bool, error)
	// ByIDs returns the members that exist among ids, in no particular order.
	ByIDs(ctx context.Context, ids []uint64) ([]domain.Member, error)
}

type Service struct {
	members  Members
	hasher   domain.PasswordHasher
	handoffs Handoffs
	tokens   Tokens
	secrets  Secrets
	audit    AuditLog
}

func NewService(members Members, hasher domain.PasswordHasher, handoffs Handoffs, tokens Tokens, secrets Secrets) *Service {
	return &Service{members: members, hasher: hasher, handoffs: handoffs, tokens: tokens, secrets: secrets}
}

func (s *Service) Register(ctx context.Context, email, displayName, password string) (domain.Member, error) {
	m, err := domain.NewMember(email, displayName, password, s.hasher)
	if err != nil {
		return domain.Member{}, err
	}
	if _, found, err := s.members.ByEmail(ctx, m.Email); err != nil {
		return domain.Member{}, err
	} else if found {
		return domain.Member{}, ErrEmailTaken
	}
	return s.members.Add(ctx, m)
}

// Login never says whether the email exists: every failure is
// ErrBadCredentials.
func (s *Service) Login(ctx context.Context, email, password string) (domain.Member, error) {
	email, err := domain.NormaliseEmail(email)
	if err != nil {
		return domain.Member{}, ErrBadCredentials
	}
	m, found, err := s.members.ByEmail(ctx, email)
	if err != nil {
		return domain.Member{}, err
	}
	if !found || !m.PasswordMatches(password, s.hasher) {
		return domain.Member{}, ErrBadCredentials
	}
	return m, nil
}

func (s *Service) CurrentMember(ctx context.Context, id uint64) (domain.Member, error) {
	m, found, err := s.members.ByID(ctx, id)
	if err != nil {
		return domain.Member{}, err
	}
	if !found {
		return domain.Member{}, ErrMemberNotFound
	}
	return m, nil
}

// MemberIDByEmail resolves an email to a member id, for other contexts that
// only know a person by their email.
func (s *Service) MemberIDByEmail(ctx context.Context, email string) (uint64, bool, error) {
	email, err := domain.NormaliseEmail(email)
	if err != nil {
		return 0, false, nil
	}
	m, found, err := s.members.ByEmail(ctx, email)
	return m.ID, found, err
}

// DisplayNames maps member ids to display names, for other contexts that show
// members. Unknown ids are left out.
func (s *Service) DisplayNames(ctx context.Context, ids []uint64) (map[uint64]string, error) {
	members, err := s.members.ByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	names := make(map[uint64]string, len(members))
	for _, m := range members {
		names[m.ID] = m.DisplayName
	}
	return names, nil
}
