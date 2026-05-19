package scopes

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz"
)

func TestValidator_DefineScope(t *testing.T) {
	validator := NewValidator(func(p *authn.Principal) []string { return []string{} })

	scope := &ScopeDefinition{
		Name:        "read",
		Description: "Read access",
		Actions:     []string{"read", "get"},
		Resources:   []string{"*"},
	}

	validator.DefineScope(scope)

	if validator.ScopeDefinitions["read"] == nil {
		t.Error("DefineScope() should add scope")
	}
}

func TestValidator_HasScope(t *testing.T) {
	getScopes := func(p *authn.Principal) []string {
		scopes, _ := p.GetAttribute("scopes")
		if scopesSlice, ok := scopes.([]string); ok {
			return scopesSlice
		}
		return []string{}
	}

	validator := NewValidator(getScopes)

	principal := &authn.Principal{
		ID: "user1",
		Attributes: map[string]interface{}{
			"scopes": []string{"read", "write"},
		},
	}

	if !validator.HasScope(principal, "read") {
		t.Error("HasScope() should return true for existing scope")
	}

	if validator.HasScope(principal, "delete") {
		t.Error("HasScope() should return false for non-existing scope")
	}
}

func TestValidator_HasAnyScope(t *testing.T) {
	getScopes := func(p *authn.Principal) []string {
		scopes, _ := p.GetAttribute("scopes")
		if scopesSlice, ok := scopes.([]string); ok {
			return scopesSlice
		}
		return []string{}
	}

	validator := NewValidator(getScopes)

	principal := &authn.Principal{
		ID: "user1",
		Attributes: map[string]interface{}{
			"scopes": []string{"read"},
		},
	}

	if !validator.HasAnyScope(principal, "read", "write") {
		t.Error("HasAnyScope() should return true when principal has any scope")
	}

	if validator.HasAnyScope(principal, "delete", "admin") {
		t.Error("HasAnyScope() should return false when principal has none of the scopes")
	}
}

func TestValidator_HasAllScopes(t *testing.T) {
	getScopes := func(p *authn.Principal) []string {
		scopes, _ := p.GetAttribute("scopes")
		if scopesSlice, ok := scopes.([]string); ok {
			return scopesSlice
		}
		return []string{}
	}

	validator := NewValidator(getScopes)

	principal := &authn.Principal{
		ID: "user1",
		Attributes: map[string]interface{}{
			"scopes": []string{"read", "write"},
		},
	}

	if !validator.HasAllScopes(principal, "read", "write") {
		t.Error("HasAllScopes() should return true when principal has all scopes")
	}

	if validator.HasAllScopes(principal, "read", "write", "admin") {
		t.Error("HasAllScopes() should return false when principal missing any scope")
	}
}

func TestValidator_CheckScope(t *testing.T) {
	getScopes := func(p *authn.Principal) []string {
		scopes, _ := p.GetAttribute("scopes")
		if scopesSlice, ok := scopes.([]string); ok {
			return scopesSlice
		}
		return []string{}
	}

	validator := NewValidator(getScopes)

	validator.DefineScope(&ScopeDefinition{
		Name:    "read",
		Actions: []string{"read", "get"},
		Resources: []string{"user:*"},
	})

	principal := &authn.Principal{
		ID: "user1",
		Attributes: map[string]interface{}{
			"scopes": []string{"read"},
		},
	}

	if !validator.CheckScope(principal, "read", "user:123") {
		t.Error("CheckScope() should return true for allowed action")
	}

	if validator.CheckScope(principal, "delete", "user:123") {
		t.Error("CheckScope() should return false for disallowed action")
	}
}

func TestValidator_Authorizer(t *testing.T) {
	getScopes := func(p *authn.Principal) []string {
		scopes, _ := p.GetAttribute("scopes")
		if scopesSlice, ok := scopes.([]string); ok {
			return scopesSlice
		}
		return []string{}
	}

	validator := NewValidator(getScopes)
	authorizer := validator.Authorizer("read", "write")

	if authorizer == nil {
		t.Error("Authorizer() should return non-nil authorizer")
	}

	ctx := context.Background()

	tests := []struct {
		name     string
		principal *authn.Principal
		want     authz.Decision
	}{
		{
			name: "has required scopes",
			principal: &authn.Principal{
				ID: "user1",
				Attributes: map[string]interface{}{
					"scopes": []string{"read", "write"},
				},
			},
			want: authz.DecisionAllow,
		},
		{
			name: "missing required scopes",
			principal: &authn.Principal{
				ID: "user1",
				Attributes: map[string]interface{}{
					"scopes": []string{"read"},
				},
			},
			want: authz.DecisionDeny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &authz.Request{
				Principal: tt.principal,
				Action:    "read",
				Resource:  "user:123",
			}

			decision, err := authorizer.Authorize(ctx, req)
			if err != nil {
				t.Errorf("Authorizer.Authorize() error = %v", err)
				return
			}
			if decision != tt.want {
				t.Errorf("Authorizer.Authorize() = %v, want %v", decision, tt.want)
			}
		})
	}
}
