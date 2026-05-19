// Package scopes provides OAuth2-style scope validation.
//
// Scopes are used to limit what an access token can do. This package allows
// you to define scopes and validate them against principals.
//
// Example usage:
//
//	// Create a scope validator
//	validator := scopes.NewValidator(func(principal *authn.Principal) []string {
//		scopesInterface, _ := principal.GetAttribute("scopes")
//		if scopes, ok := scopesInterface.([]string); ok {
//			return scopes
//		}
//		return []string{}
//	})
//
//	// Define a scope
//	validator.DefineScope(&scopes.ScopeDefinition{
//		Name:        "read",
//		Description: "Read access",
//		Actions:     []string{"read", "get"},
//		Resources:   []string{"*"},
//	})
//
//	// Check scope
//	hasScope := validator.HasScope(principal, "read")
//
//	// Create an authorizer
//	authorizer := validator.Authorizer("read", "write")
package scopes
