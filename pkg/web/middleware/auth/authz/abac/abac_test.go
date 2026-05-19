package abac

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz"
)

func TestEngine_Evaluate(t *testing.T) {
	engine := NewEngine()

	// Add allow policy for admins
	engine.AddPolicy(&Policy{
		ID:     "policy1",
		Name:   "Allow admins",
		Effect: "allow",
		PrincipalConditions: map[string]interface{}{
			"role": "admin",
		},
	})

	// Add deny policy for specific action
	engine.AddPolicy(&Policy{
		ID:     "policy2",
		Name:   "Deny delete",
		Effect: "deny",
		ActionConditions: map[string]interface{}{
			"action": "delete",
		},
	})

	ctx := context.Background()

	tests := []struct {
		name     string
		principal *authn.Principal
		action   string
		resource string
		want     authz.Decision
		wantErr  bool
	}{
		{
			name: "admin allowed",
			principal: &authn.Principal{
				ID: "admin1",
				Attributes: map[string]interface{}{
					"role": "admin",
				},
			},
			action:   "read",
			resource: "user:123",
			want:     authz.DecisionAllow,
			wantErr:  false,
		},
		{
			name: "deny always wins",
			principal: &authn.Principal{
				ID: "admin1",
				Attributes: map[string]interface{}{
					"role": "admin",
				},
			},
			action:   "delete",
			resource: "user:123",
			want:     authz.DecisionDeny,
			wantErr:  false,
		},
		{
			name: "no matching policy",
			principal: &authn.Principal{
				ID: "user1",
				Attributes: map[string]interface{}{
					"role": "user",
				},
			},
			action:   "read",
			resource: "user:123",
			want:     authz.DecisionDeny,
			wantErr:  false,
		},
		{
			name:     "no principal",
			principal: nil,
			action:   "read",
			resource: "user:123",
			want:     authz.DecisionDeny,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &authz.Request{
				Principal: tt.principal,
				Action:    tt.action,
				Resource:  tt.resource,
			}

			got, err := engine.Evaluate(ctx, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Engine.Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_AddRemovePolicy(t *testing.T) {
	engine := NewEngine()

	policy1 := &Policy{
		ID:     "policy1",
		Name:   "Policy 1",
		Effect: "allow",
	}

	policy2 := &Policy{
		ID:     "policy2",
		Name:   "Policy 2",
		Effect: "allow",
	}

	engine.AddPolicy(policy1)
	engine.AddPolicy(policy2)

	if len(engine.Policies) != 2 {
		t.Errorf("AddPolicy() len = %v, want %v", len(engine.Policies), 2)
	}

	engine.RemovePolicy("policy1")

	if len(engine.Policies) != 1 {
		t.Errorf("RemovePolicy() len = %v, want %v", len(engine.Policies), 1)
	}

	if engine.Policies[0].ID != "policy2" {
		t.Errorf("RemovePolicy() remaining policy ID = %v, want %v", engine.Policies[0].ID, "policy2")
	}
}

func TestEngine_Authorizer(t *testing.T) {
	engine := NewEngine()
	authorizer := engine.Authorizer()

	if authorizer == nil {
		t.Error("Authorizer() should return non-nil authorizer")
	}

	ctx := context.Background()
	req := &authz.Request{
		Principal: &authn.Principal{
			ID: "user1",
			Attributes: map[string]interface{}{
				"role": "user",
			},
		},
		Action:   "read",
		Resource: "user:123",
	}

	decision, err := authorizer.Authorize(ctx, req)
	if err != nil {
		t.Errorf("Authorizer.Authorize() error = %v", err)
	}
	if decision != authz.DecisionDeny {
		t.Errorf("Authorizer.Authorize() = %v, want %v", decision, authz.DecisionDeny)
	}
}

func TestEngine_matchesConditions(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name       string
		conditions map[string]interface{}
		attributes map[string]interface{}
		want       bool
	}{
		{
			name:       "empty conditions match",
			conditions: map[string]interface{}{},
			attributes: map[string]interface{}{"role": "admin"},
			want:       true,
		},
		{
			name: "matching conditions",
			conditions: map[string]interface{}{
				"role": "admin",
			},
			attributes: map[string]interface{}{
				"role": "admin",
			},
			want: true,
		},
		{
			name: "non-matching conditions",
			conditions: map[string]interface{}{
				"role": "admin",
			},
			attributes: map[string]interface{}{
				"role": "user",
			},
			want: false,
		},
		{
			name: "missing attribute",
			conditions: map[string]interface{}{
				"role": "admin",
			},
			attributes: map[string]interface{}{},
			want: false,
		},
		{
			name: "nil attributes",
			conditions: map[string]interface{}{
				"role": "admin",
			},
			attributes: nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engine.matchesConditions(tt.conditions, tt.attributes); got != tt.want {
				t.Errorf("Engine.matchesConditions() = %v, want %v", got, tt.want)
			}
		})
	}
}
