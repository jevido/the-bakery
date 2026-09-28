package domain

import (
	"testing"
	"time"
)

func TestNewSignal(t *testing.T) {
	now := time.Now()
	if _, err := NewSignal(SignalLimitHit, "", 3, "203.0.113.7", now); err == nil {
		t.Fatal("a limit hit without a limit name was accepted")
	}
	if _, err := NewSignal("visit", "", 3, "203.0.113.7", now); err == nil {
		t.Fatal("an unknown kind was accepted")
	}
	s, err := NewSignal(SignalSignUp, "login", 3, "203.0.113.7", now)
	if err != nil || s.Limit != "" || !s.At.Equal(now) {
		t.Fatalf("sign-up: %+v, %v", s, err)
	}
	s, err = NewSignal(SignalLimitHit, "register", 0, "203.0.113.7", now)
	if err != nil || s.Limit != "register" || s.MemberID != 0 {
		t.Fatalf("limit hit: %+v, %v", s, err)
	}
}
