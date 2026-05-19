package policies

// Engine evaluates a request against a set of policies (e.g. first deny wins).
type Engine interface {
	Evaluate(req Request) Result
	AddPolicy(p Policy)
	RemovePolicy(name string)
}

// DefaultEngine evaluates policies in order; first explicit deny wins, else first allow, else deny.
type DefaultEngine struct {
	policies []Policy
}

// NewDefaultEngine creates an engine with no policies.
func NewDefaultEngine() *DefaultEngine {
	return &DefaultEngine{policies: make([]Policy, 0)}
}

// Evaluate runs all policies; first deny wins, else first allow, else deny.
func (e *DefaultEngine) Evaluate(req Request) Result {
	var allowResult Result
	for _, p := range e.policies {
		res, applied := p.Evaluate(req)
		if !applied {
			continue
		}
		if res.Effect == EffectDeny {
			return Result{Allowed: false, Effect: EffectDeny, Reason: res.Reason}
		}
		// remember first allow in case no deny
		if allowResult.Reason == "" {
			allowResult = res
		}
	}
	if allowResult.Reason != "" {
		return Result{Allowed: true, Effect: EffectAllow, Reason: allowResult.Reason}
	}
	return Result{Allowed: false, Effect: EffectDeny, Reason: "no matching policy"}
}

// AddPolicy appends a policy.
func (e *DefaultEngine) AddPolicy(p Policy) {
	if p == nil {
		return
	}
	e.policies = append(e.policies, p)
}

// RemovePolicy removes the first policy with the given name.
func (e *DefaultEngine) RemovePolicy(name string) {
	for i, p := range e.policies {
		if p.Name() == name {
			e.policies = append(e.policies[:i], e.policies[i+1:]...)
			return
		}
	}
}
