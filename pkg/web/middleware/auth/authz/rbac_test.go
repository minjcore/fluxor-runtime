package authz

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
)

func TestRBACAuthorizer_Authorize(t *testing.T) {
	getRoles := func(p *authn.Principal) []string {
		roles, _ := p.GetAttribute("roles")
		if rolesSlice, ok := roles.([]string); ok {
			return rolesSlice
		}
		return []string{}
	}

	rbac := NewRBACAuthorizer(getRoles)
	rbac.AssignRole("admin", []Permission{
		{Action: "*", Resource: "*"},
	})
	rbac.AssignRole("user", []Permission{
		{Action: "read", Resource: "user:*"},
		{Action: "write", Resource: "user:own"},
	})

	ctx := context.Background()

	tests := []struct {
		name     string
		principal *authn.Principal
		action   string
		resource string
		want     Decision
		wantErr  bool
	}{
		{
			name: "admin can do anything",
			principal: &authn.Principal{
				ID: "admin1",
				Attributes: map[string]interface{}{
					"roles": []string{"admin"},
				},
			},
			action:   "delete",
			resource: "user:123",
			want:     DecisionAllow,
			wantErr:  false,
		},
		{
			name: "user can read any user",
			principal: &authn.Principal{
				ID: "user1",
				Attributes: map[string]interface{}{
					"roles": []string{"user"},
				},
			},
			action:   "read",
			resource: "user:123",
			want:     DecisionAllow,
			wantErr:  false,
		},
		{
			name: "user cannot delete",
			principal: &authn.Principal{
				ID: "user1",
				Attributes: map[string]interface{}{
					"roles": []string{"user"},
				},
			},
			action:   "delete",
			resource: "user:123",
			want:     DecisionDeny,
			wantErr:  false,
		},
		{
			name: "no principal",
			principal: nil,
			action:   "read",
			resource: "user:123",
			want:     DecisionDeny,
			wantErr:  true,
		},
		{
			name: "no roles",
			principal: &authn.Principal{
				ID: "user1",
				Attributes: map[string]interface{}{},
			},
			action:   "read",
			resource: "user:123",
			want:     DecisionDeny,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{
				Principal: tt.principal,
				Action:    tt.action,
				Resource:  tt.resource,
			}

			got, err := rbac.Authorize(ctx, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("RBACAuthorizer.Authorize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RBACAuthorizer.Authorize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSimpleRoleAuthorizer(t *testing.T) {
	authorizer := RequireRole("admin")
	ctx := context.Background()

	tests := []struct {
		name     string
		principal *authn.Principal
		want     Decision
	}{
		{
			name: "has required role",
			principal: &authn.Principal{
				ID: "admin1",
				Attributes: map[string]interface{}{
					"roles": []string{"admin", "user"},
				},
			},
			want: DecisionAllow,
		},
		{
			name: "does not have required role",
			principal: &authn.Principal{
				ID: "user1",
				Attributes: map[string]interface{}{
					"roles": []string{"user"},
				},
			},
			want: DecisionDeny,
		},
		{
			name: "no roles",
			principal: &authn.Principal{
				ID: "user1",
				Attributes: map[string]interface{}{},
			},
			want: DecisionDeny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{
				Principal: tt.principal,
				Action:    "read",
				Resource:  "user:123",
			}

			got, err := authorizer.Authorize(ctx, req)
			if err != nil {
				t.Errorf("SimpleRoleAuthorizer.Authorize() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("SimpleRoleAuthorizer.Authorize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRBACAuthorizer_matchesResourcePattern(t *testing.T) {
	rbac := NewRBACAuthorizer(func(p *authn.Principal) []string { return []string{} })

	tests := []struct {
		name     string
		pattern  string
		resource string
		want     bool
	}{
		{
			name:     "wildcard matches anything",
			pattern:  "*",
			resource: "user:123",
			want:     true,
		},
		{
			name:     "exact match",
			pattern:  "user:123",
			resource: "user:123",
			want:     true,
		},
		{
			name:     "prefix match",
			pattern:  "user:*",
			resource: "user:123",
			want:     true,
		},
		{
			name:     "no match",
			pattern:  "file:*",
			resource: "user:123",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rbac.matchesResourcePattern(tt.pattern, tt.resource); got != tt.want {
				t.Errorf("RBACAuthorizer.matchesResourcePattern() = %v, want %v", got, tt.want)
			}
		})
	}
}
