package domain

import (
	"errors"
	"time"
)

// Signal kinds: what may point at abuse, counted on the console's Signals
// screen. A signal says nothing by itself; volume does.
const (
	SignalSignUp       = "sign_up"
	SignalGuildFounded = "guild_founded"
	SignalLimitHit     = "limit_hit"
)

var ErrInvalidSignal = errors.New("a signal needs a known kind, and a limit hit needs the limit's name")

// Signal is one sign-up, guild founded or rate-limit refusal, with who (when
// known) and the client's IP. Like audit entries, signals are only added.
type Signal struct {
	ID       uint64
	Kind     string
	Limit    string
	MemberID uint64
	IP       string
	At       time.Time
}

// NewSignal checks a signal and stamps it.
func NewSignal(kind, limit string, memberID uint64, ip string, now time.Time) (Signal, error) {
	switch kind {
	case SignalSignUp, SignalGuildFounded:
		limit = ""
	case SignalLimitHit:
		if limit == "" {
			return Signal{}, ErrInvalidSignal
		}
	default:
		return Signal{}, ErrInvalidSignal
	}
	if len(ip) > 64 {
		ip = ip[:64]
	}
	return Signal{Kind: kind, Limit: limit, MemberID: memberID, IP: ip, At: now}, nil
}
