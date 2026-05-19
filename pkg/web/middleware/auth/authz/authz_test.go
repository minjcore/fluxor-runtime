package authz

import (
	"testing"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
)

func TestDecision_String(t *testing.T) {
	tests := []struct {
		name     string
		decision Decision
		want     string
	}{
		{
			name:     "deny",
			decision: DecisionDeny,
			want:     "deny",
		},
		{
			name:     "allow",
			decision: DecisionAllow,
			want:     "allow",
		},
		{
			name:     "abstain",
			decision: DecisionAbstain,
			want:     "abstain",
		},
		{
			name:     "unknown",
			decision: Decision(999),
			want:     "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.decision.String(); got != tt.want {
				t.Errorf("Decision.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequest(t *testing.T) {
	principal := &authn.Principal{
		ID: "user1",
		Attributes: map[string]interface{}{
			"role": "admin",
		},
	}

	req := &Request{
		Principal: principal,
		Action:    "read",
		Resource:  "user:123",
		Context: map[string]interface{}{
			"ip": "127.0.0.1",
		},
	}

	if req.Principal.ID != "user1" {
		t.Errorf("Request.Principal.ID = %v, want %v", req.Principal.ID, "user1")
	}

	if req.Action != "read" {
		t.Errorf("Request.Action = %v, want %v", req.Action, "read")
	}

	if req.Resource != "user:123" {
		t.Errorf("Request.Resource = %v, want %v", req.Resource, "user:123")
	}

	if req.Context["ip"] != "127.0.0.1" {
		t.Errorf("Request.Context[ip] = %v, want %v", req.Context["ip"], "127.0.0.1")
	}
}
