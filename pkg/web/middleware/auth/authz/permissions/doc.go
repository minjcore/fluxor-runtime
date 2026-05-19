// Package permissions provides fine-grained permission management.
//
// This package allows you to define permissions and assign them to principals
// or roles. Permissions can be checked for specific actions on resources.
//
// Example usage:
//
//	// Create a permissions manager
//	manager := permissions.NewManager()
//
//	// Define a permission
//	manager.DefinePermission(&permissions.PermissionDefinition{
//		Name:        "users.read",
//		Description: "Read user information",
//		Actions:     []string{"read", "get"},
//		Resources:   []string{"user:*"},
//	})
//
//	// Assign permission to a role
//	manager.AssignRolePermission("user", "users.read")
//
//	// Check permission
//	allowed := manager.CheckPermission(principal, "read", "user:123")
//
//	// Create an authorizer
//	authorizer := manager.Authorizer()
package permissions
