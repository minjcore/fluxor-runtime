package permissions

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz"
)

func TestManager_DefinePermission(t *testing.T) {
	manager := NewManager()

	perm := &PermissionDefinition{
		Name:        "users.read",
		Description: "Read users",
		Actions:     []string{"read", "get"},
		Resources:   []string{"user:*"},
	}

	manager.DefinePermission(perm)

	if manager.PermissionDefinitions["users.read"] == nil {
		t.Error("DefinePermission() should add permission")
	}
}

func TestManager_AssignPermission(t *testing.T) {
	manager := NewManager()

	manager.AssignPermission("user1", "users.read")
	manager.AssignPermission("user1", "users.write")

	perms := manager.PrincipalPermissions["user1"]
	if len(perms) != 2 {
		t.Errorf("AssignPermission() len = %v, want %v", len(perms), 2)
	}

	// Test duplicate assignment
	manager.AssignPermission("user1", "users.read")
	if len(manager.PrincipalPermissions["user1"]) != 2 {
		t.Error("AssignPermission() should not add duplicate")
	}
}

func TestManager_AssignRolePermission(t *testing.T) {
	manager := NewManager()

	manager.AssignRolePermission("admin", "users.read")
	manager.AssignRolePermission("admin", "users.write")

	perms := manager.RolePermissions["admin"]
	if len(perms) != 2 {
		t.Errorf("AssignRolePermission() len = %v, want %v", len(perms), 2)
	}
}

func TestManager_GetPermissions(t *testing.T) {
	manager := NewManager()

	// Assign direct permission
	manager.AssignPermission("user1", "users.read")

	// Assign role permission
	manager.AssignRolePermission("admin", "users.write")

	principal := &authn.Principal{
		ID: "user1",
		Attributes: map[string]interface{}{
			"roles": []string{"admin"},
		},
	}

	perms := manager.GetPermissions(principal)
	if len(perms) != 2 {
		t.Errorf("GetPermissions() len = %v, want %v", len(perms), 2)
	}
}

func TestManager_HasPermission(t *testing.T) {
	manager := NewManager()
	manager.AssignPermission("user1", "users.read")
	manager.AssignPermission("user1", "users.*")

	principal := &authn.Principal{
		ID: "user1",
	}

	if !manager.HasPermission(principal, "users.read") {
		t.Error("HasPermission() should return true for exact match")
	}

	if !manager.HasPermission(principal, "users.write") {
		t.Error("HasPermission() should return true for wildcard match")
	}

	if manager.HasPermission(principal, "files.read") {
		t.Error("HasPermission() should return false for non-matching permission")
	}
}

func TestManager_CheckPermission(t *testing.T) {
	manager := NewManager()

	manager.DefinePermission(&PermissionDefinition{
		Name:    "users.read",
		Actions: []string{"read", "get"},
		Resources: []string{"user:*"},
	})

	manager.AssignPermission("user1", "users.read")

	principal := &authn.Principal{
		ID: "user1",
	}

	if !manager.CheckPermission(principal, "read", "user:123") {
		t.Error("CheckPermission() should return true for allowed action")
	}

	if manager.CheckPermission(principal, "delete", "user:123") {
		t.Error("CheckPermission() should return false for disallowed action")
	}
}

func TestManager_Authorizer(t *testing.T) {
	manager := NewManager()
	manager.DefinePermission(&PermissionDefinition{
		Name:    "users.read",
		Actions: []string{"read"},
		Resources: []string{"user:*"},
	})
	manager.AssignPermission("user1", "users.read")

	authorizer := manager.Authorizer()
	if authorizer == nil {
		t.Error("Authorizer() should return non-nil authorizer")
	}

	ctx := context.Background()
	req := &authz.Request{
		Principal: &authn.Principal{
			ID: "user1",
		},
		Action:   "read",
		Resource: "user:123",
	}

	decision, err := authorizer.Authorize(ctx, req)
	if err != nil {
		t.Errorf("Authorizer.Authorize() error = %v", err)
	}
	if decision != authz.DecisionAllow {
		t.Errorf("Authorizer.Authorize() = %v, want %v", decision, authz.DecisionAllow)
	}
}
