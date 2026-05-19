package permissions

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz"
)

// Manager manages permissions for principals
type Manager struct {
	// PrincipalPermissions maps principal IDs to their permissions
	PrincipalPermissions map[string][]string

	// RolePermissions maps roles to permissions
	RolePermissions map[string][]string

	// PermissionDefinitions defines what each permission allows
	PermissionDefinitions map[string]*PermissionDefinition
}

// PermissionDefinition defines what a permission allows
type PermissionDefinition struct {
	// Name is the permission name (e.g., "users.read", "files.write")
	Name string

	// Description describes what the permission allows
	Description string

	// Actions are the actions this permission allows
	Actions []string

	// Resources are the resource patterns this permission applies to
	Resources []string
}

// NewManager creates a new permissions manager
func NewManager() *Manager {
	return &Manager{
		PrincipalPermissions:  make(map[string][]string),
		RolePermissions:       make(map[string][]string),
		PermissionDefinitions: make(map[string]*PermissionDefinition),
	}
}

// DefinePermission defines a permission
func (m *Manager) DefinePermission(perm *PermissionDefinition) {
	m.PermissionDefinitions[perm.Name] = perm
}

// AssignPermission assigns a permission directly to a principal
func (m *Manager) AssignPermission(principalID, permission string) {
	if m.PrincipalPermissions == nil {
		m.PrincipalPermissions = make(map[string][]string)
	}
	perms := m.PrincipalPermissions[principalID]
	for _, p := range perms {
		if p == permission {
			return // Already assigned
		}
	}
	m.PrincipalPermissions[principalID] = append(perms, permission)
}

// AssignRolePermission assigns a permission to a role
func (m *Manager) AssignRolePermission(role, permission string) {
	if m.RolePermissions == nil {
		m.RolePermissions = make(map[string][]string)
	}
	perms := m.RolePermissions[role]
	for _, p := range perms {
		if p == permission {
			return // Already assigned
		}
	}
	m.RolePermissions[role] = append(perms, permission)
}

// GetPermissions returns all permissions for a principal
func (m *Manager) GetPermissions(principal *authn.Principal) []string {
	perms := make(map[string]bool)

	// Get direct permissions
	if directPerms, ok := m.PrincipalPermissions[principal.ID]; ok {
		for _, p := range directPerms {
			perms[p] = true
		}
	}

	// Get role-based permissions
	rolesInterface, ok := principal.GetAttribute("roles")
	if ok {
		var roles []string
		if rolesSlice, ok := rolesInterface.([]string); ok {
			roles = rolesSlice
		} else if rolesSlice, ok := rolesInterface.([]interface{}); ok {
			roles = make([]string, 0, len(rolesSlice))
			for _, r := range rolesSlice {
				if roleStr, ok := r.(string); ok {
					roles = append(roles, roleStr)
				}
			}
		}

		for _, role := range roles {
			if rolePerms, ok := m.RolePermissions[role]; ok {
				for _, p := range rolePerms {
					perms[p] = true
				}
			}
		}
	}

	// Convert to slice
	result := make([]string, 0, len(perms))
	for p := range perms {
		result = append(result, p)
	}

	return result
}

// HasPermission checks if a principal has a specific permission
func (m *Manager) HasPermission(principal *authn.Principal, permission string) bool {
	perms := m.GetPermissions(principal)
	for _, p := range perms {
		if p == permission {
			return true
		}
		// Support wildcard permissions (e.g., "users.*" matches "users.read")
		if strings.HasSuffix(p, ".*") {
			prefix := strings.TrimSuffix(p, ".*")
			if strings.HasPrefix(permission, prefix+".") {
				return true
			}
		}
	}
	return false
}

// CheckPermission checks if a principal has permission for an action on a resource
func (m *Manager) CheckPermission(principal *authn.Principal, action, resource string) bool {
	perms := m.GetPermissions(principal)

	for _, permName := range perms {
		def, ok := m.PermissionDefinitions[permName]
		if !ok {
			// If no definition, check if permission name matches action.resource pattern
			if m.matchesPermissionPattern(permName, action, resource) {
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
				if r == "*" || r == resource || m.matchesResourcePattern(r, resource) {
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

// matchesPermissionPattern checks if a permission name matches action.resource
func (m *Manager) matchesPermissionPattern(permName, action, resource string) bool {
	parts := strings.Split(permName, ".")
	if len(parts) != 2 {
		return false
	}
	return (parts[0] == "*" || parts[0] == action) && (parts[1] == "*" || parts[1] == resource || m.matchesResourcePattern(parts[1], resource))
}

// matchesResourcePattern checks if a resource pattern matches a resource
func (m *Manager) matchesResourcePattern(pattern, resource string) bool {
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

// Authorizer creates an Authorizer from the manager
func (m *Manager) Authorizer() authz.Authorizer {
	return &permissionAuthorizer{manager: m}
}

type permissionAuthorizer struct {
	manager *Manager
}

func (p *permissionAuthorizer) Authorize(ctx context.Context, req *authz.Request) (authz.Decision, error) {
	if req.Principal == nil {
		return authz.DecisionDeny, fmt.Errorf("principal is required: %w", authz.ErrInvalidRequest)
	}

	allowed := p.manager.CheckPermission(req.Principal, req.Action, req.Resource)
	if allowed {
		return authz.DecisionAllow, nil
	}

	return authz.DecisionDeny, nil
}
