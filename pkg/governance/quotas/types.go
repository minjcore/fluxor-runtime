package quotas

// Limiter checks and optionally consumes quota (e.g. rate or usage limit per key).
type Limiter interface {
	// Allow checks if the key is within quota. If consume is true, consumes one unit.
	Allow(key string, consume bool) (allowed bool, remaining int64)
	// Reset resets usage for key (if supported).
	Reset(key string) error
}
