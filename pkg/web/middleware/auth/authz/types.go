package authz

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
)

// Decision represents an authorization decision
type Decision int

const (
	// DecisionDeny explicitly denies access
	DecisionDeny Decision = iota
	// DecisionAllow explicitly allows access
	DecisionAllow
	// DecisionAbstain means the authorizer doesn't have an opinion
	DecisionAbstain
)

// String returns the string representation of the decision
func (d Decision) String() string {
	switch d {
	case DecisionDeny:
		return "deny"
	case DecisionAllow:
		return "allow"
	case DecisionAbstain:
		return "abstain"
	default:
		return "unknown"
	}
}

// Request represents an authorization request
type Request struct {
	// Principal is the authenticated entity requesting access
	Principal *authn.Principal

	// Action is the action being requested (e.g., "read", "write", "delete")
	Action string

	// Resource is the resource being accessed (e.g., "user:123", "file:/path/to/file")
	Resource string

	// Context contains additional context for the authorization decision
	Context map[string]interface{}
}

// Authorizer is the interface for authorization providers
type Authorizer interface {
	// Authorize determines if a principal is authorized to perform an action on a resource
	Authorize(ctx context.Context, req *Request) (Decision, error)
}

// Policy represents an authorization policy
type Policy struct {
	// ID is the unique identifier of the policy
	ID string

	// Name is a human-readable name for the policy
	Name string

	// Description describes what the policy does
	Description string

	// Effect is the effect of the policy (allow or deny)
	Effect string

	// Conditions are the conditions that must be met for the policy to apply
	Conditions map[string]interface{}

	// Actions are the actions this policy applies to
	Actions []string

	// Resources are the resources this policy applies to
	Resources []string
}

// PolicyEngine evaluates policies to make authorization decisions
type PolicyEngine interface {
	// Evaluate evaluates a request against policies and returns a decision
	Evaluate(ctx context.Context, req *Request, policies []*Policy) (Decision, error)

	// AddPolicy adds a policy to the engine
	AddPolicy(policy *Policy) error

	// RemovePolicy removes a policy from the engine
	RemovePolicy(policyID string) error

	// GetPolicies returns all policies
	GetPolicies() []*Policy
}
