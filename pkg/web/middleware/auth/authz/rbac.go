package authz

import (
	"context"
	"fmt"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
)

// RBACAuthorizer implements Role-Based Access Control
type RBACAuthorizer struct {
	// GetRoles extracts roles from a principal
	GetRoles func(principal *authn.Principal) []string

	// RolePermissions maps roles to their permissions
	RolePermissions map[string][]Permission

	// DefaultDecision is the decision when no matching role is found
	DefaultDecision Decision
}

// Permission represents a permission (action + resource)
type Permission struct {
	Action   string
	Resource string
}

// NewRBACAuthorizer creates a new RBAC authorizer
func NewRBACAuthorizer(getRoles func(*authn.Principal) []string) *RBACAuthorizer {
	return &RBACAuthorizer{
		GetRoles:        getRoles,
		RolePermissions: make(map[string][]Permission),
		DefaultDecision: DecisionDeny,
	}
}

// AssignRole assigns permissions to a role
func (r *RBACAuthorizer) AssignRole(role string, permissions []Permission) {
	if r.RolePermissions == nil {
		r.RolePermissions = make(map[string][]Permission)
	}
	r.RolePermissions[role] = permissions
}

// Authorize checks if the principal is authorized
func (r *RBACAuthorizer) Authorize(ctx context.Context, req *Request) (Decision, error) {
	if req.Principal == nil {
		return DecisionDeny, fmt.Errorf("principal is required: %w", ErrInvalidRequest)
	}

	// Get roles for the principal
	roles := r.GetRoles(req.Principal)
	if len(roles) == 0 {
		return r.DefaultDecision, nil
	}

	// Check if any role has the required permission
	requiredPerm := Permission{
		Action:   req.Action,
		Resource: req.Resource,
	}

	for _, role := range roles {
		permissions, ok := r.RolePermissions[role]
		if !ok {
			continue
		}

		if r.hasPermission(permissions, requiredPerm) {
			return DecisionAllow, nil
		}
	}

	return DecisionDeny, nil
}

// hasPermission checks if a permission list contains the required permission
func (r *RBACAuthorizer) hasPermission(permissions []Permission, required Permission) bool {
	for _, perm := range permissions {
		if r.matchesPermission(perm, required) {
			return true
		}
	}
	return false
}

// matchesPermission checks if a permission matches the required permission
// Supports wildcards: "*" matches anything
func (r *RBACAuthorizer) matchesPermission(perm, required Permission) bool {
	actionMatch := perm.Action == "*" || perm.Action == required.Action
	resourceMatch := perm.Resource == "*" || perm.Resource == required.Resource || r.matchesResourcePattern(perm.Resource, required.Resource)
	return actionMatch && resourceMatch
}

// matchesResourcePattern checks if a resource pattern matches a resource
// Supports patterns like "user:*", "file:/path/*", etc.
func (r *RBACAuthorizer) matchesResourcePattern(pattern, resource string) bool {
	if pattern == "*" {
		return true
	}

	// Simple prefix matching for now
	// In production, you might want more sophisticated pattern matching
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(resource) >= len(prefix) && resource[:len(prefix)] == prefix
	}

	return pattern == resource
}

// RequireRole creates a simple authorizer that requires a specific role
func RequireRole(role string) Authorizer {
	return &SimpleRoleAuthorizer{
		RequiredRole: role,
	}
}

// SimpleRoleAuthorizer is a simple authorizer that checks for a single role
type SimpleRoleAuthorizer struct {
	RequiredRole string
}

// Authorize checks if the principal has the required role
func (s *SimpleRoleAuthorizer) Authorize(ctx context.Context, req *Request) (Decision, error) {
	if req.Principal == nil {
		return DecisionDeny, fmt.Errorf("principal is required: %w", ErrInvalidRequest)
	}

	// Try to get roles from principal attributes
	rolesInterface, ok := req.Principal.GetAttribute("roles")
	if !ok {
		return DecisionDeny, nil
	}

	roles, ok := rolesInterface.([]string)
	if !ok {
		// Try to convert from []interface{}
		if rolesSlice, ok := rolesInterface.([]interface{}); ok {
			roles = make([]string, 0, len(rolesSlice))
			for _, r := range rolesSlice {
				if roleStr, ok := r.(string); ok {
					roles = append(roles, roleStr)
				}
			}
		} else {
			return DecisionDeny, nil
		}
	}

	for _, role := range roles {
		if role == s.RequiredRole {
			return DecisionAllow, nil
		}
	}

	return DecisionDeny, nil
}
