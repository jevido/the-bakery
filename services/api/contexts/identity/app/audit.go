package app

import "context"

// AuditLog records a sensitive action in the platform's audit log (the
// moderation context's, set when the contexts are wired). Recording never
// fails the action.
type AuditLog interface {
	Record(ctx context.Context, e AuditRecord)
}

// AuditRecord is one sensitive action by a member.
type AuditRecord struct {
	ActorID    uint64
	Action     string
	TargetKind string
	TargetID   uint64
	Meta       map[string]any
}

type noAudit struct{}

func (noAudit) Record(context.Context, AuditRecord) {}

// SetAuditLog sets where sensitive actions are recorded.
func (s *Service) SetAuditLog(a AuditLog) { s.audit = a }

func (s *Service) auditLog() AuditLog {
	if s.audit == nil {
		return noAudit{}
	}
	return s.audit
}
