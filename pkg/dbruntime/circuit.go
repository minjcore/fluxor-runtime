package dbruntime

import (
	"errors"
	"log/slog"
	"sync"
	"time"
)

// CircuitState represents the circuit breaker state.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // normal — requests pass through
	CircuitOpen                         // tripped — requests fail fast
	CircuitHalfOpen                     // testing — one probe request allowed
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit is open and requests are rejected.
var ErrCircuitOpen = errors.New("dbruntime: circuit breaker open — database unavailable")

// CircuitConfig configures the circuit breaker.
type CircuitConfig struct {
	// Threshold: number of consecutive ping failures before tripping. Default 3.
	Threshold int

	// OpenTimeout: how long to stay open before probing. Default 30s.
	OpenTimeout time.Duration

	// OnTrip is called when the circuit trips open. err is the failure that caused it.
	// Use this to fire an alert, push a Slack message, increment a counter, etc.
	OnTrip func(err error, failures int)

	// OnReset is called when the circuit resets to closed after a successful probe.
	OnReset func()

	// OnProbe is called each time a half-open probe is attempted.
	OnProbe func()
}

func (c *CircuitConfig) setDefaults() {
	if c.Threshold <= 0 {
		c.Threshold = 3
	}
	if c.OpenTimeout <= 0 {
		c.OpenTimeout = 30 * time.Second
	}
}

// Circuit is a circuit breaker for database health checks.
// It wraps the health-check loop in DB and adds trip/reset logic with notifications.
//
// The circuit does NOT wrap individual queries — database/sql handles per-query
// retries internally. The circuit only controls whether the health-check considers
// the pool healthy enough to accept new connections.
type Circuit struct {
	cfg      CircuitConfig
	mu       sync.Mutex
	state    CircuitState
	failures int
	trippedAt time.Time
}

// NewCircuit creates a circuit breaker with the given config.
func NewCircuit(cfg CircuitConfig) *Circuit {
	cfg.setDefaults()
	return &Circuit{cfg: cfg}
}

// State returns the current circuit state (safe for concurrent use).
func (c *Circuit) State() CircuitState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resolveState()
}

// resolveState returns the current state, transitioning Open→HalfOpen if timeout elapsed.
// Must be called with c.mu held.
func (c *Circuit) resolveState() CircuitState {
	if c.state == CircuitOpen && time.Since(c.trippedAt) >= c.cfg.OpenTimeout {
		c.state = CircuitHalfOpen
		slog.Info("dbruntime: circuit half-open — probing database",
			"open_duration", time.Since(c.trippedAt).Round(time.Second))
		if c.cfg.OnProbe != nil {
			go c.cfg.OnProbe()
		}
	}
	return c.state
}

// Allow reports whether a health-check ping should proceed.
// Returns false when the circuit is open (fail fast).
func (c *Circuit) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resolveState() != CircuitOpen
}

// RecordSuccess records a successful ping.
// If the circuit was half-open it resets to closed and fires OnReset.
func (c *Circuit) RecordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	wasHalfOpen := c.state == CircuitHalfOpen
	c.failures = 0
	c.state = CircuitClosed
	if wasHalfOpen {
		slog.Info("dbruntime: circuit reset — database recovered")
		if c.cfg.OnReset != nil {
			go c.cfg.OnReset()
		}
	}
}

// RecordFailure records a failed ping.
// Trips the circuit to open when consecutive failures exceed the threshold.
func (c *Circuit) RecordFailure(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures++
	slog.Warn("dbruntime: health check failure",
		"failures", c.failures,
		"threshold", c.cfg.Threshold,
		"circuit", c.resolveState().String(),
		"err", err)

	if c.state != CircuitOpen && c.failures >= c.cfg.Threshold {
		c.state = CircuitOpen
		c.trippedAt = time.Now()
		slog.Error("dbruntime: circuit tripped — database unreachable, failing fast",
			"failures", c.failures,
			"open_timeout", c.cfg.OpenTimeout)
		if c.cfg.OnTrip != nil {
			go c.cfg.OnTrip(err, c.failures)
		}
	}
}

// Stats returns a snapshot of circuit breaker state for metrics/observability.
func (c *Circuit) Stats() CircuitStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CircuitStats{
		State:     c.resolveState(),
		Failures:  c.failures,
		TrippedAt: c.trippedAt,
	}
}

// CircuitStats is a point-in-time snapshot for Prometheus or logging.
type CircuitStats struct {
	State     CircuitState
	Failures  int
	TrippedAt time.Time // zero if never tripped
}
