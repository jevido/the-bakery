package domain

import (
	"errors"
	"strings"
	"time"
)

// Actor kinds and target kinds in the audit log.
const (
	ActorMember   = "member"
	ActorOperator = "operator"

	TargetMember   = "member"
	TargetGuild    = "guild"
	TargetToken    = "personal_token"
	TargetSanction = "sanction"
	TargetReport   = "report"
	TargetOperator = "operator"
)

var ErrInvalidAuditEntry = errors.New("an audit entry needs an actor and an action")

// AuditEntry is one sensitive action on the platform. It is written once
// and never changed or removed.
type AuditEntry struct {
	ID         uint64
	ActorKind  string
	ActorID    uint64
	Action     string
	TargetKind string
	TargetID   uint64
	Reason     string
	Meta       map[string]any
	IP         string
	At         time.Time
}

// NewAuditEntry checks an entry before it is written.
func NewAuditEntry(e AuditEntry, at time.Time) (AuditEntry, error) {
	e.Action = strings.TrimSpace(e.Action)
	if e.Action == "" || (e.ActorKind != ActorMember && e.ActorKind != ActorOperator) || e.ActorID == 0 {
		return AuditEntry{}, ErrInvalidAuditEntry
	}
	if e.Meta == nil {
		e.Meta = map[string]any{}
	}
	e.At = at
	return e, nil
}
