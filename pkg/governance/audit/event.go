package audit

import "time"

// Event represents an immutable audit event for compliance (SOX, HIPAA, GDPR).
type Event struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Actor     string                 `json:"actor"`   // who performed the action
	Action    string                 `json:"action"`  // e.g. "login", "delete", "export"
	Resource  string                 `json:"resource"` // e.g. "user:123", "file:/path"
	Outcome   string                 `json:"outcome"`  // "success", "failure", "denied"
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

// Logger writes audit events. Implementations should append-only (immutable trail).
type Logger interface {
	Log(event Event) error
}
