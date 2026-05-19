package abac

import (
	"context"
	"fmt"
	"reflect"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz"
)

// Engine implements Attribute-Based Access Control
type Engine struct {
	// Policies are the ABAC policies
	Policies []*Policy
}

// Policy represents an ABAC policy
type Policy struct {
	// ID is the unique identifier
	ID string

	// Name is a human-readable name
	Name string

	// Effect is "allow" or "deny"
	Effect string

	// PrincipalConditions are conditions on the principal
	PrincipalConditions map[string]interface{}

	// ResourceConditions are conditions on the resource
	ResourceConditions map[string]interface{}

	// ActionConditions are conditions on the action
	ActionConditions map[string]interface{}

	// EnvironmentConditions are conditions on the environment/context
	EnvironmentConditions map[string]interface{}
}

// NewEngine creates a new ABAC engine
func NewEngine() *Engine {
	return &Engine{
		Policies: make([]*Policy, 0),
	}
}

// AddPolicy adds a policy to the engine
func (e *Engine) AddPolicy(policy *Policy) {
	e.Policies = append(e.Policies, policy)
}

// RemovePolicy removes a policy by ID
func (e *Engine) RemovePolicy(policyID string) {
	policies := make([]*Policy, 0, len(e.Policies))
	for _, p := range e.Policies {
		if p.ID != policyID {
			policies = append(policies, p)
		}
	}
	e.Policies = policies
}

// Evaluate evaluates a request against all policies
func (e *Engine) Evaluate(ctx context.Context, req *authz.Request) (authz.Decision, error) {
	if req.Principal == nil {
		return authz.DecisionDeny, fmt.Errorf("principal is required: %w", authz.ErrInvalidRequest)
	}

	// Evaluate policies in order
	// Deny always wins, otherwise first matching allow wins
	var allowFound bool

	for _, policy := range e.Policies {
		if e.matchesPolicy(policy, req) {
			if policy.Effect == "allow" {
				allowFound = true
			} else if policy.Effect == "deny" {
				// Deny always wins
				return authz.DecisionDeny, nil
			}
		}
	}

	if allowFound {
		return authz.DecisionAllow, nil
	}

	return authz.DecisionDeny, nil
}

// matchesPolicy checks if a policy matches a request
func (e *Engine) matchesPolicy(policy *Policy, req *authz.Request) bool {
	// Check principal conditions
	if !e.matchesConditions(policy.PrincipalConditions, req.Principal.Attributes) {
		return false
	}

	// Check resource conditions
	if req.Resource != "" {
		resourceAttrs := map[string]interface{}{
			"resource": req.Resource,
		}
		if !e.matchesConditions(policy.ResourceConditions, resourceAttrs) {
			return false
		}
	}

	// Check action conditions
	if req.Action != "" {
		actionAttrs := map[string]interface{}{
			"action": req.Action,
		}
		if !e.matchesConditions(policy.ActionConditions, actionAttrs) {
			return false
		}
	}

	// Check environment conditions
	if !e.matchesConditions(policy.EnvironmentConditions, req.Context) {
		return false
	}

	return true
}

// matchesConditions checks if attributes match conditions
func (e *Engine) matchesConditions(conditions, attributes map[string]interface{}) bool {
	if len(conditions) == 0 {
		return true
	}

	if attributes == nil {
		return false
	}

	for key, conditionValue := range conditions {
		attrValue, ok := attributes[key]
		if !ok {
			return false
		}

		// Support various comparison operators
		if !e.compareValues(attrValue, conditionValue) {
			return false
		}
	}

	return true
}

// compareValues compares two values with support for operators
func (e *Engine) compareValues(attrValue, conditionValue interface{}) bool {
	// If condition is a map, it might contain operators
	if condMap, ok := conditionValue.(map[string]interface{}); ok {
		return e.compareWithOperators(attrValue, condMap)
	}

	// Direct equality comparison
	return reflect.DeepEqual(attrValue, conditionValue)
}

// compareWithOperators compares values using operators like $eq, $ne, $in, etc.
func (e *Engine) compareWithOperators(attrValue interface{}, operators map[string]interface{}) bool {
	// $eq - equals
	if eq, ok := operators["$eq"]; ok {
		if !reflect.DeepEqual(attrValue, eq) {
			return false
		}
	}

	// $ne - not equals
	if ne, ok := operators["$ne"]; ok {
		if reflect.DeepEqual(attrValue, ne) {
			return false
		}
	}

	// $in - in array
	if in, ok := operators["$in"]; ok {
		if inSlice, ok := in.([]interface{}); ok {
			found := false
			for _, v := range inSlice {
				if reflect.DeepEqual(attrValue, v) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// $gt, $gte, $lt, $lte - numeric comparisons
	if attrNum, ok := attrValue.(float64); ok {
		if gt, ok := operators["$gt"].(float64); ok {
			if attrNum <= gt {
				return false
			}
		}
		if gte, ok := operators["$gte"].(float64); ok {
			if attrNum < gte {
				return false
			}
		}
		if lt, ok := operators["$lt"].(float64); ok {
			if attrNum >= lt {
				return false
			}
		}
		if lte, ok := operators["$lte"].(float64); ok {
			if attrNum > lte {
				return false
			}
		}
	}

	return true
}

// Authorizer creates an Authorizer from the engine
func (e *Engine) Authorizer() authz.Authorizer {
	return &abacAuthorizer{engine: e}
}

type abacAuthorizer struct {
	engine *Engine
}

func (a *abacAuthorizer) Authorize(ctx context.Context, req *authz.Request) (authz.Decision, error) {
	return a.engine.Evaluate(ctx, req)
}
