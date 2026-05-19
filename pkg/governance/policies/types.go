package policies

// Request is a policy check request (subject, action, resource).
type Request struct {
	Subject  string
	Action   string
	Resource string
	Context  map[string]interface{}
}

// Effect is allow or deny.
type Effect int

const (
	EffectDeny  Effect = iota
	EffectAllow
)

// Result is the outcome of policy evaluation.
type Result struct {
	Allowed bool
	Effect  Effect
	Reason  string
}

// Policy evaluates a request and returns a result. Multiple policies can be
// combined by an Engine (e.g. first deny wins).
type Policy interface {
	Evaluate(req Request) (Result, bool) // Result and true if this policy applied
	Name() string
}
