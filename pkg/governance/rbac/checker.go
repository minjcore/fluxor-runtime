package rbac

import (
	"strings"
	"sync"
)

// RoleChecker is an in-memory RBAC checker: subject -> roles, role -> permissions.
type RoleChecker struct {
	mu               sync.RWMutex
	subjectRoles     map[string][]string       // subjectID -> roles
	rolePermissions  map[string][]Permission   // role -> permissions
}

// NewRoleChecker creates an empty checker.
func NewRoleChecker() *RoleChecker {
	return &RoleChecker{
		subjectRoles:    make(map[string][]string),
		rolePermissions: make(map[string][]Permission),
	}
}

// AssignRole assigns a role to a subject.
func (r *RoleChecker) AssignRole(subjectID, role string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	roles := r.subjectRoles[subjectID]
	for _, existing := range roles {
		if existing == role {
			return
		}
	}
	r.subjectRoles[subjectID] = append(r.subjectRoles[subjectID], role)
}

// AssignRoles assigns multiple roles to a subject.
func (r *RoleChecker) AssignRoles(subjectID string, roles []string) {
	for _, role := range roles {
		r.AssignRole(subjectID, role)
	}
}

// SetRolePermissions sets the full permission list for a role (replaces existing).
func (r *RoleChecker) SetRolePermissions(role string, perms []Permission) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rolePermissions[role] = append([]Permission(nil), perms...)
}

// Allow returns true if the subject has a role that grants the action on resource.
// Supports "*" for action or resource; resource supports suffix "*" (prefix match).
func (r *RoleChecker) Allow(subjectID, action, resource string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles := r.subjectRoles[subjectID]
	for _, role := range roles {
		perms := r.rolePermissions[role]
		for _, p := range perms {
			if matchAction(p.Action, action) && matchResource(p.Resource, resource) {
				return true, nil
			}
		}
	}
	return false, nil
}

func matchAction(pattern, value string) bool {
	return pattern == "*" || pattern == value
}

func matchResource(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}
	return pattern == value
}
