// Package abac provides Attribute-Based Access Control (ABAC) functionality.
//
// ABAC is a flexible authorization model that uses attributes (characteristics)
// of principals, resources, actions, and environment to make authorization decisions.
//
// Example usage:
//
//	// Create an ABAC engine
//	engine := abac.NewEngine()
//
//	// Add a policy
//	policy := &abac.Policy{
//		ID:     "policy1",
//		Name:   "Allow admins to delete any resource",
//		Effect: "allow",
//		PrincipalConditions: map[string]interface{}{
//			"role": "admin",
//		},
//		ActionConditions: map[string]interface{}{
//			"action": "delete",
//		},
//	}
//	engine.AddPolicy(policy)
//
//	// Create an authorizer
//	authorizer := engine.Authorizer()
//
//	// Evaluate a request
//	decision, err := authorizer.Authorize(ctx, &authz.Request{
//		Principal: principal,
//		Action:    "delete",
//		Resource:  "user:123",
//	})
package abac
