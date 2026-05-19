package compute

// BackpressurePolicy defines how to handle queue overflow
type BackpressurePolicy int

const (
	// Block blocks the caller until queue has space (safe, default)
	Block BackpressurePolicy = iota

	// DropNewest drops the newest job when queue is full
	DropNewest

	// DropOldest drops the oldest job when queue is full
	DropOldest

	// CoalesceByKey coalesces jobs with the same key (keep newest, drop older)
	// Useful for LLM streaming, image processing where only latest state matters
	CoalesceByKey
)

// String returns the string representation of the policy
func (p BackpressurePolicy) String() string {
	switch p {
	case Block:
		return "Block"
	case DropNewest:
		return "DropNewest"
	case DropOldest:
		return "DropOldest"
	case CoalesceByKey:
		return "CoalesceByKey"
	default:
		return "Unknown"
	}
}

// RoutingPolicy defines how jobs are routed to workers
type RoutingPolicy int

const (
	// RoundRobin distributes jobs evenly across workers
	RoundRobin RoutingPolicy = iota

	// HashByKey routes jobs with same key to same worker (locality)
	// Critical for LLM KV cache, session affinity
	HashByKey

	// LeastBusy routes to worker with shortest queue
	LeastBusy
)

// String returns the string representation of the routing policy
func (r RoutingPolicy) String() string {
	switch r {
	case RoundRobin:
		return "RoundRobin"
	case HashByKey:
		return "HashByKey"
	case LeastBusy:
		return "LeastBusy"
	default:
		return "Unknown"
	}
}
