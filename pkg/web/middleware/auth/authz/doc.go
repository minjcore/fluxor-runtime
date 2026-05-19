// Package authz provides authorization functionality for the Fluxor framework.
//
// Package Independence:
//
// This package is completely independent from the web layer and can be used
// in any Go application (CLI tools, gRPC services, workers, etc.). It only
// depends on Go standard library packages (context, errors, fmt) and the
// authn package (for Principal type). It has no dependencies on pkg/web or
// any framework-specific code. The package location is for organizational
// purposes only.
//
// This package defines core authorization interfaces and types that can be
// used with various authorization models. It includes:
//
//   - Decision: Authorization decision (Allow, Deny, Abstain)
//   - Request: Authorization request with principal, action, and resource
//   - Authorizer: Interface for authorization providers
//   - Policy: Authorization policy definition
//
// Sub-packages:
//
//   - rbac: Role-Based Access Control
//   - abac: Attribute-Based Access Control
//   - permissions: Fine-grained permission management
//   - scopes: OAuth2-style scope validation
//
// Example usage:
//
//	// Using RBAC
//	import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz"
//
//	rbac := authz.NewRBACAuthorizer(func(p *authn.Principal) []string {
//		roles, _ := p.GetAttribute("roles")
//		return roles.([]string)
//	})
//	rbac.AssignRole("admin", []authz.Permission{
//		{Action: "*", Resource: "*"},
//	})
//
//	decision, err := rbac.Authorize(ctx, &authz.Request{
//		Principal: principal,
//		Action:    "delete",
//		Resource:  "user:123",
//	})
//
//	// Using ABAC
//	import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz/abac"
//
//	engine := abac.NewEngine()
//	engine.AddPolicy(&abac.Policy{
//		ID:     "policy1",
//		Effect: "allow",
//		PrincipalConditions: map[string]interface{}{
//			"role": "admin",
//		},
//	})
//	authorizer := engine.Authorizer()
//
//	// Using Permissions
//	import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz/permissions"
//
//	manager := permissions.NewManager()
//	manager.DefinePermission(&permissions.PermissionDefinition{
//		Name:    "users.read",
//		Actions: []string{"read"},
//		Resources: []string{"user:*"},
//	})
//	manager.AssignRolePermission("user", "users.read")
//
//	// Using Scopes
//	import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz/scopes"
//
//	validator := scopes.NewValidator(func(p *authn.Principal) []string {
//		scopes, _ := p.GetAttribute("scopes")
//		return scopes.([]string)
//	})
//	validator.DefineScope(&scopes.ScopeDefinition{
//		Name:    "read",
//		Actions: []string{"read"},
//	})
package authz
