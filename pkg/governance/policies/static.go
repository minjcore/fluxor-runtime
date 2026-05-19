package policies

// StaticPolicy is a policy that allows or denies fixed subject/action/resource patterns.
// Empty string matches any value for that field.
type StaticPolicy struct {
	name     string
	effect   Effect
	subject  string
	action   string
	resource string
}

// NewStaticPolicy creates a policy: if subject/action/resource match (empty = any), return effect.
func NewStaticPolicy(name string, effect Effect, subject, action, resource string) *StaticPolicy {
	return &StaticPolicy{
		name:     name,
		effect:   effect,
		subject:  subject,
		action:   action,
		resource: resource,
	}
}

// Name returns the policy name.
func (s *StaticPolicy) Name() string {
	return s.name
}

// Evaluate returns (result, true) if this policy matches the request.
func (s *StaticPolicy) Evaluate(req Request) (Result, bool) {
	if !match(s.subject, req.Subject) || !match(s.action, req.Action) || !match(s.resource, req.Resource) {
		return Result{}, false
	}
	return Result{
		Allowed: s.effect == EffectAllow,
		Effect:  s.effect,
		Reason:  s.name,
	}, true
}

func match(pattern, value string) bool {
	if pattern == "" {
		return true
	}
	return pattern == value
}
