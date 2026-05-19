package health

import (
	"context"
	"sync"
	"time"
)

// Checker is a health check function
type Checker func(ctx context.Context) error

// NamedChecker is a health check with a name
type NamedChecker struct {
	Name    string
	Checker Checker
	Timeout time.Duration
}

// Registry manages health checks
type Registry struct {
	mu       sync.RWMutex
	checkers map[string]*NamedChecker
}

// NewRegistry creates a new health check registry
func NewRegistry() *Registry {
	return &Registry{
		checkers: make(map[string]*NamedChecker),
	}
}

// Register registers a health check
func (r *Registry) Register(name string, checker Checker) {
	r.RegisterWithTimeout(name, checker, 5*time.Second)
}

// RegisterWithTimeout registers a health check with a timeout
func (r *Registry) RegisterWithTimeout(name string, checker Checker, timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.checkers[name] = &NamedChecker{
		Name:    name,
		Checker: checker,
		Timeout: timeout,
	}
}

// Unregister removes a health check
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.checkers, name)
}

// Check runs all health checks and returns results
func (r *Registry) Check(ctx context.Context) map[string]CheckResult {
	r.mu.RLock()
	checkers := make(map[string]*NamedChecker)
	for k, v := range r.checkers {
		checkers[k] = v
	}
	r.mu.RUnlock()

	results := make(map[string]CheckResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, checker := range checkers {
		wg.Add(1)
		go func(name string, checker *NamedChecker) {
			defer wg.Done()

			result := r.runCheck(ctx, checker)
			mu.Lock()
			results[name] = result
			mu.Unlock()
		}(name, checker)
	}

	wg.Wait()
	return results
}

// runCheck runs a single health check with timeout
func (r *Registry) runCheck(ctx context.Context, checker *NamedChecker) CheckResult {
	timeout := checker.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	err := checker.Checker(checkCtx)
	duration := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:   StatusDown,
			Message:  err.Error(),
			Duration: duration,
		}
	}

	return CheckResult{
		Status:   StatusUp,
		Message:  "OK",
		Duration: duration,
	}
}

// CheckResult represents the result of a health check
type CheckResult struct {
	Status   Status        `json:"status"`
	Message  string        `json:"message,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
}

// Status represents health check status
type Status string

const (
	StatusUp   Status = "UP"
	StatusDown Status = "DOWN"
)

// Global registry
var globalRegistry = NewRegistry()

// Register registers a health check in the global registry
func Register(name string, checker Checker) {
	globalRegistry.Register(name, checker)
}

// RegisterWithTimeout registers a health check with timeout in the global registry
func RegisterWithTimeout(name string, checker Checker, timeout time.Duration) {
	globalRegistry.RegisterWithTimeout(name, checker, timeout)
}

// Unregister removes a health check from the global registry
func Unregister(name string) {
	globalRegistry.Unregister(name)
}

// Check runs all health checks in the global registry
func Check(ctx context.Context) map[string]CheckResult {
	return globalRegistry.Check(ctx)
}
