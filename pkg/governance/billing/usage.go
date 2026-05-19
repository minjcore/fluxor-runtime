package billing

import "time"

// UsageEvent represents a metering event for billing (e.g. API call, storage, compute).
type UsageEvent struct {
	TenantID   string    `json:"tenant_id,omitempty"`
	SubjectID  string    `json:"subject_id"`  // user or service that incurred usage
	Resource   string    `json:"resource"`    // e.g. "api_calls", "storage_gb"
	Quantity   float64   `json:"quantity"`   // amount (e.g. 1, 0.5)
	Unit       string    `json:"unit"`       // e.g. "request", "gb"
	Timestamp  time.Time `json:"timestamp"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

// Recorder records usage events for billing/metering.
type Recorder interface {
	Record(event UsageEvent) error
}
