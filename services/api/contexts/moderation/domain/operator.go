// Package domain is the moderation context's model: operators, sanctions,
// reports and the audit log. It depends on nothing outside the standard
// library.
package domain

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

var (
	ErrInvalidOperatorEmail = errors.New("an operator needs a valid email address")
	ErrWeakOperatorPassword = errors.New("an operator's password must be at least 12 characters")
	ErrOperatorConfirmed    = errors.New("this operator has already confirmed their authenticator")
	ErrOperatorUnconfirmed  = errors.New("confirm the authenticator first")
)

const operatorPasswordMin = 12

// Operator is an account for the console: the platform owner or someone
// they appoint. It is not a member. TOTPSecret is kept encrypted; the
// domain never sees it in the clear.
type Operator struct {
	ID           uint64
	Email        string
	PasswordHash string
	TOTPSecret   string
	ConfirmedAt  *time.Time
}

// NewOperator makes an operator who still has to confirm their
// authenticator with a first code.
func NewOperator(email, password, passwordHash, encryptedSecret string) (Operator, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if a, err := mail.ParseAddress(email); err != nil || a.Address != email {
		return Operator{}, ErrInvalidOperatorEmail
	}
	if len([]rune(password)) < operatorPasswordMin {
		return Operator{}, ErrWeakOperatorPassword
	}
	return Operator{Email: email, PasswordHash: passwordHash, TOTPSecret: encryptedSecret}, nil
}

// CheckPasswordLength is the rule NewOperator applies, for callers that
// hash first.
func CheckPasswordLength(password string) error {
	if len([]rune(password)) < operatorPasswordMin {
		return ErrWeakOperatorPassword
	}
	return nil
}

// Confirm marks the authenticator as set up; the caller has checked the
// first code.
func (o *Operator) Confirm(now time.Time) error {
	if o.ConfirmedAt != nil {
		return ErrOperatorConfirmed
	}
	o.ConfirmedAt = &now
	return nil
}

// MaySignIn refuses an operator whose authenticator is not set up.
func (o Operator) MaySignIn() error {
	if o.ConfirmedAt == nil {
		return ErrOperatorUnconfirmed
	}
	return nil
}
