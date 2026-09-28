package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewOperator(t *testing.T) {
	tests := []struct {
		email, password string
		want            error
	}{
		{" Owner@Bakery.test ", "a long enough secret", nil},
		{"not an email", "a long enough secret", ErrInvalidOperatorEmail},
		{"Owner <owner@bakery.test>", "a long enough secret", ErrInvalidOperatorEmail},
		{"owner@bakery.test", "short", ErrWeakOperatorPassword},
	}
	for _, tt := range tests {
		o, err := NewOperator(tt.email, tt.password, "hash", "enc")
		if !errors.Is(err, tt.want) {
			t.Errorf("NewOperator(%q) = %v, want %v", tt.email, err, tt.want)
		}
		if err == nil && (o.Email != "owner@bakery.test" || o.ConfirmedAt != nil) {
			t.Errorf("operator = %+v", o)
		}
	}
}

func TestOperatorConfirm(t *testing.T) {
	o, _ := NewOperator("owner@bakery.test", "a long enough secret", "h", "e")
	if err := o.MaySignIn(); !errors.Is(err, ErrOperatorUnconfirmed) {
		t.Fatalf("unconfirmed may sign in: %v", err)
	}
	if err := o.Confirm(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := o.MaySignIn(); err != nil {
		t.Fatal(err)
	}
	if err := o.Confirm(time.Now()); !errors.Is(err, ErrOperatorConfirmed) {
		t.Fatalf("confirmed twice: %v", err)
	}
}
