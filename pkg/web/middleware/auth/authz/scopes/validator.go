package scopes

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz"
)

// Validator validates OAuth2-style scopes
type Validator struct {
	// ScopeDefinitions defines what each scope allows
	ScopeDefinitions map[string]*ScopeDefinition

	// GetScopes extracts scopes from a principal
	GetScopes func(principal *authn.Principal) []string
}

// ScopeDefinition defines what a scope allows
type ScopeDefinition struct {
	// Name is the scope name (e.g., "read", "write", "admin")
	Name string

	// Description describes what the scope allows
	Description string

	// Actions are the actions this scope allows
	Actions []string

	// Resources are the resource patterns this scope applies to
	Resources []string
}

// NewValidator creates a new scope validator
func NewValidator(getScopes func(*authn.Principal) []string) *Validator {
	return &Validator{
		ScopeDefinitions: make(map[string]*ScopeDefinition),
		GetScopes:        getScopes,
	}
}

// DefineScope defines a scope
func (v *Validator) DefineScope(scope *ScopeDefinition) {
	v.ScopeDefinitions[scope.Name] = scope
}

// HasScope checks if a principal has a specific scope
func (v *Validator) HasScope(principal *authn.Principal, scope string) bool {
	scopes := v.GetScopes(principal)
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasAnyScope checks if a principal has any of the specified scopes
func (v *Validator) HasAnyScope(principal *authn.Principal, requiredScopes ...string) bool {
	scopes := v.GetScopes(principal)
	scopeMap := make(map[string]bool)
	for _, s := range scopes {
		scopeMap[s] = true
	}

	for _, required := range requiredScopes {
		if scopeMap[required] {
			return true
		}
	}

	return false
}

// HasAllScopes checks if a principal has all of the specified scopes
func (v *Validator) HasAllScopes(principal *authn.Principal, requiredScopes ...string) bool {
	scopes := v.GetScopes(principal)
	scopeMap := make(map[string]bool)
	for _, s := range scopes {
		scopeMap[s] = true
	}

	for _, required := range requiredScopes {
		if !scopeMap[required] {
			return false
		}
	}

	return true
}

// CheckScope checks if a principal's scopes allow an action on a resource
func (v *Validator) CheckScope(principal *authn.Principal, action, resource string) bool {
	scopes := v.GetScopes(principal)

	for _, scopeName := range scopes {
		def, ok := v.ScopeDefinitions[scopeName]
		if !ok {
			// If no definition, assume scope name matches action
			if scopeName == action || scopeName == "*" {
				return true
			}
			continue
		}

		// Check if action matches
		actionMatch := false
		if len(def.Actions) == 0 {
			actionMatch = true // No action restriction
		} else {
			for _, a := range def.Actions {
				if a == "*" || a == action {
					actionMatch = true
					break
				}
			}
		}

		// Check if resource matches
		resourceMatch := false
		if len(def.Resources) == 0 {
			resourceMatch = true // No resource restriction
		} else {
			for _, r := range def.Resources {
				if r == "*" || r == resource || v.matchesResourcePattern(r, resource) {
					resourceMatch = true
					break
				}
			}
		}

		if actionMatch && resourceMatch {
			return true
		}
	}

	return false
}

// matchesResourcePattern checks if a resource pattern matches a resource
func (v *Validator) matchesResourcePattern(pattern, resource string) bool {
	if pattern == "*" {
		return true
	}

	// Support prefix matching
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(resource, prefix)
	}

	return pattern == resource
}

// Authorizer creates an Authorizer from the validator
func (v *Validator) Authorizer(requiredScopes ...string) authz.Authorizer {
	return &scopeAuthorizer{
		validator:      v,
		requiredScopes: requiredScopes,
	}
}

type scopeAuthorizer struct {
	validator      *Validator
	requiredScopes []string
}

func (s *scopeAuthorizer) Authorize(ctx context.Context, req *authz.Request) (authz.Decision, error) {
	if req.Principal == nil {
		return authz.DecisionDeny, fmt.Errorf("principal is required: %w", authz.ErrInvalidRequest)
	}

	// If specific scopes are required, check them
	if len(s.requiredScopes) > 0 {
		if !s.validator.HasAllScopes(req.Principal, s.requiredScopes...) {
			return authz.DecisionDeny, nil
		}
	}

	// Check if scopes allow the action on the resource
	if req.Action != "" && req.Resource != "" {
		allowed := s.validator.CheckScope(req.Principal, req.Action, req.Resource)
		if allowed {
			return authz.DecisionAllow, nil
		}
	}

	// If we have required scopes and principal has them, allow
	if len(s.requiredScopes) > 0 {
		return authz.DecisionAllow, nil
	}

	return authz.DecisionDeny, nil
}
