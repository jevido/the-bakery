// Package app holds the moderation use cases.
package app

import (
	"context"
	"errors"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

var (
	ErrOperatorExists   = errors.New("an operator with that email already exists")
	ErrOperatorNotFound = errors.New("operator not found")
	// ErrBadCredentials covers a wrong email, password or code alike, so a
	// refusal tells an attacker nothing.
	ErrBadCredentials = errors.New("wrong email, password or code")
)

type Operators interface {
	Add(ctx context.Context, o domain.Operator) (domain.Operator, error)
	Save(ctx context.Context, o domain.Operator) error
	ByID(ctx context.Context, id uint64) (domain.Operator, bool, error)
	ByEmail(ctx context.Context, email string) (domain.Operator, bool, error)
}

type Hasher interface {
	Hash(password string) (string, error)
	Check(password, hash string) bool
}

// Secrets encrypts TOTP secrets at rest.
type Secrets interface {
	Encrypt(plain string) (string, error)
	Decrypt(encrypted string) (string, error)
}

// TOTP makes and checks time-based one-time codes.
type TOTP interface {
	// Generate makes a secret for an account and the otpauth:// URL an
	// authenticator app reads.
	Generate(account string) (secret, url string, err error)
	Validate(code, secret string) bool
}

type Service struct {
	operators Operators
	hasher    Hasher
	secrets   Secrets
	totp      TOTP
	audit     AuditEntries
	sanctions Sanctions
	reports   Reports
	targets   Targets
	effects   Effects
	directory Directory
	cache     activeCache
	now       func() time.Time
}

// Deps are what the moderation use cases work with.
type Deps struct {
	Operators Operators
	Hasher    Hasher
	Secrets   Secrets
	TOTP      TOTP
	Audit     AuditEntries
	Sanctions Sanctions
	Reports   Reports
	Targets   Targets
	Effects   Effects
	Directory Directory
}

func NewService(d Deps) *Service {
	return &Service{
		operators: d.Operators, hasher: d.Hasher, secrets: d.Secrets, totp: d.TOTP, audit: d.Audit,
		sanctions: d.Sanctions, reports: d.Reports, targets: d.Targets, effects: d.Effects, directory: d.Directory,
		cache: activeCache{m: map[string]cached{}}, now: time.Now,
	}
}

// CreateOperator makes an operator and returns the otpauth:// URL for
// their authenticator; they confirm it with ConfirmOperator.
func (s *Service) CreateOperator(ctx context.Context, email, password string) (domain.Operator, string, error) {
	if err := domain.CheckPasswordLength(password); err != nil {
		return domain.Operator{}, "", err
	}
	if _, found, err := s.operators.ByEmail(ctx, email); err != nil || found {
		if err == nil {
			err = ErrOperatorExists
		}
		return domain.Operator{}, "", err
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.Operator{}, "", err
	}
	secret, url, err := s.totp.Generate(email)
	if err != nil {
		return domain.Operator{}, "", err
	}
	enc, err := s.secrets.Encrypt(secret)
	if err != nil {
		return domain.Operator{}, "", err
	}
	o, err := domain.NewOperator(email, password, hash, enc)
	if err != nil {
		return domain.Operator{}, "", err
	}
	o, err = s.operators.Add(ctx, o)
	return o, url, err
}

// ConfirmOperator checks the first code from the operator's authenticator.
func (s *Service) ConfirmOperator(ctx context.Context, email, code string) error {
	o, found, err := s.operators.ByEmail(ctx, email)
	if err != nil {
		return err
	}
	if !found {
		return ErrOperatorNotFound
	}
	if !s.validCode(o, code) {
		return ErrBadCredentials
	}
	if err := o.Confirm(s.now()); err != nil {
		return err
	}
	return s.operators.Save(ctx, o)
}

// CheckPassword is the first step of signing in: the operator whose email
// and password these are. The second step is CheckCode.
func (s *Service) CheckPassword(ctx context.Context, email, password string) (uint64, error) {
	o, found, err := s.operators.ByEmail(ctx, email)
	if err != nil {
		return 0, err
	}
	if !found || !s.hasher.Check(password, o.PasswordHash) || o.MaySignIn() != nil {
		return 0, ErrBadCredentials
	}
	return o.ID, nil
}

// CheckCode is the second step: the code from the operator's authenticator.
func (s *Service) CheckCode(ctx context.Context, operatorID uint64, code string) error {
	o, found, err := s.operators.ByID(ctx, operatorID)
	if err != nil {
		return err
	}
	if !found || o.MaySignIn() != nil || !s.validCode(o, code) {
		return ErrBadCredentials
	}
	return nil
}

func (s *Service) validCode(o domain.Operator, code string) bool {
	secret, err := s.secrets.Decrypt(o.TOTPSecret)
	return err == nil && s.totp.Validate(code, secret)
}

// Operator returns a signed-in operator.
func (s *Service) Operator(ctx context.Context, id uint64) (domain.Operator, error) {
	o, found, err := s.operators.ByID(ctx, id)
	if err != nil {
		return domain.Operator{}, err
	}
	if !found {
		return domain.Operator{}, ErrOperatorNotFound
	}
	return o, nil
}
