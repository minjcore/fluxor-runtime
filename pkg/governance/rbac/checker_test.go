package rbac

import (
	"testing"
)

func TestRoleChecker_Allow_no_roles_denied(t *testing.T) {
	c := NewRoleChecker()
	ok, err := c.Allow("user1", "read", "documents")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected deny when subject has no roles")
	}
}

func TestRoleChecker_Allow_role_with_permission(t *testing.T) {
	c := NewRoleChecker()
	c.AssignRole("user1", "viewer")
	c.SetRolePermissions("viewer", []Permission{{Action: "read", Resource: "documents"}})
	ok, err := c.Allow("user1", "read", "documents")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected allow when role has permission")
	}
}

func TestRoleChecker_Allow_wildcard_action(t *testing.T) {
	c := NewRoleChecker()
	c.AssignRole("admin", "admin")
	c.SetRolePermissions("admin", []Permission{{Action: "*", Resource: "users"}})
	ok, err := c.Allow("admin", "delete", "users")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected allow for * action")
	}
}

func TestRoleChecker_Allow_resource_prefix(t *testing.T) {
	c := NewRoleChecker()
	c.AssignRole("u", "editor")
	c.SetRolePermissions("editor", []Permission{{Action: "write", Resource: "doc:*"}})
	ok, err := c.Allow("u", "write", "doc:123")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected allow for resource prefix match")
	}
}

func TestRoleChecker_AssignRoles(t *testing.T) {
	c := NewRoleChecker()
	c.AssignRoles("u", []string{"a", "b"})
	c.SetRolePermissions("a", []Permission{{Action: "read", Resource: "x"}})
	ok, _ := c.Allow("u", "read", "x")
	if !ok {
		t.Error("expected allow after AssignRoles")
	}
}
