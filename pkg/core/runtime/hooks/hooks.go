package hooks

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Hook represents a function that can be registered and executed.
// The context parameter allows hooks to be cancelled or have timeouts.
type Hook func(ctx context.Context) error

// HookInfo contains metadata about a registered hook.
type HookInfo struct {
	// Name is the unique identifier for the hook.
	Name string

	// Priority determines execution order. Lower values execute first.
	// Hooks with the same priority are executed in registration order.
	Priority int

	// Hook is the function to execute.
	Hook Hook

	// Async determines if the hook should be executed asynchronously.
	Async bool

	// RegisteredAt is when the hook was registered.
	RegisteredAt time.Time

	// ExecutionCount is the number of times this hook has been executed.
	ExecutionCount int64

	// LastExecutionTime is when this hook was last executed.
	LastExecutionTime time.Time

	// LastError is the last error returned by this hook (if any).
	LastError error
}

// Config configures hook registry behavior.
type Config struct {
	// DefaultTimeout is the default timeout for hook execution.
	// If zero, hooks execute without timeout.
	DefaultTimeout time.Duration

	// OnHookStart is called when a hook execution starts.
	OnHookStart func(name string)

	// OnHookComplete is called when a hook execution completes.
	OnHookComplete func(name string, err error)

	// OnHookError is called when a hook execution encounters an error.
	OnHookError func(name string, err error)

	// StopOnError determines whether to stop executing remaining hooks
	// when a hook returns an error. Defaults to false.
	StopOnError bool

	// Parallel controls whether hooks are executed in parallel or sequentially.
	// Defaults to false (sequential).
	Parallel bool

	// MaxConcurrency limits the number of hooks that can execute concurrently
	// when Parallel is true. Zero means unlimited.
	MaxConcurrency int
}

// DefaultConfig returns the default hook registry configuration.
func DefaultConfig() Config {
	return Config{
		StopOnError: false,
		Parallel:    false,
	}
}

// Stats contains statistics about hook execution.
type Stats struct {
	// TotalHooks is the total number of registered hooks.
	TotalHooks int64

	// TotalExecutions is the total number of hook executions.
	TotalExecutions int64

	// TotalSuccesses is the total number of successful hook executions.
	TotalSuccesses int64

	// TotalFailures is the total number of failed hook executions.
	TotalFailures int64

	// LastExecutionTime is when hooks were last executed.
	LastExecutionTime time.Time

	// IsExecuting indicates if hooks are currently being executed.
	IsExecuting bool
}

// Registry provides a system for registering and executing hooks.
type Registry interface {
	// Register registers a hook with the given name and priority.
	// If a hook with the same name already exists, an error is returned.
	Register(name string, hook Hook, opts ...Option) error

	// Unregister removes a hook from the registry.
	Unregister(name string) error

	// Execute executes all registered hooks in priority order.
	// If context has a deadline, it's used as the timeout for each hook.
	Execute(ctx context.Context) error

	// ExecuteByName executes a specific hook by name.
	ExecuteByName(ctx context.Context, name string) error

	// ExecuteByPriority executes hooks with a specific priority.
	ExecuteByPriority(ctx context.Context, priority int) error

	// GetHook returns information about a registered hook.
	GetHook(name string) (*HookInfo, error)

	// ListHooks returns a list of all registered hooks, sorted by priority.
	ListHooks() []HookInfo

	// Stats returns statistics about hook execution.
	Stats() Stats

	// Clear removes all registered hooks.
	Clear()
}

// Option configures hook registration.
type Option func(*HookInfo)

// WithPriority sets the hook priority.
func WithPriority(priority int) Option {
	return func(info *HookInfo) {
		info.Priority = priority
	}
}

// WithAsync makes the hook execute asynchronously.
func WithAsync(async bool) Option {
	return func(info *HookInfo) {
		info.Async = async
	}
}

// hookRegistry implements the Registry interface.
type hookRegistry struct {
	config Config
	hooks  map[string]*HookInfo
	mu     sync.RWMutex

	// Statistics
	stats Stats
	isExecuting int32 // atomic: 1 if executing, 0 otherwise
	lastExecutionTime atomic.Value // stores time.Time
}

// NewRegistry creates a new hook registry with the given configuration.
func NewRegistry(config Config) Registry {
	return &hookRegistry{
		config: config,
		hooks:  make(map[string]*HookInfo),
	}
}

// Register registers a hook with the given name and options.
func (r *hookRegistry) Register(name string, hook Hook, opts ...Option) error {
	if name == "" {
		return NewError(ErrCodeNilHook, "hook name cannot be empty")
	}
	if hook == nil {
		return NewError(ErrCodeNilHook, "hook cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if hook already exists
	if _, exists := r.hooks[name]; exists {
		return NewError(ErrCodeHookExists, fmt.Sprintf("hook %s already exists", name))
	}

	// Create hook info
	info := &HookInfo{
		Name:         name,
		Priority:     0, // Default priority
		Hook:         hook,
		Async:        false,
		RegisteredAt: time.Now(),
	}

	// Apply options
	for _, opt := range opts {
		opt(info)
	}

	r.hooks[name] = info
	atomic.AddInt64(&r.stats.TotalHooks, 1)

	return nil
}

// Unregister removes a hook from the registry.
func (r *hookRegistry) Unregister(name string) error {
	if name == "" {
		return NewError(ErrCodeHookNotFound, "hook name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.hooks[name]; !exists {
		return NewError(ErrCodeHookNotFound, fmt.Sprintf("hook %s not found", name))
	}

	delete(r.hooks, name)
	atomic.AddInt64(&r.stats.TotalHooks, -1)

	return nil
}

// Execute executes all registered hooks in priority order.
func (r *hookRegistry) Execute(ctx context.Context) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	// Check if already executing
	if !atomic.CompareAndSwapInt32(&r.isExecuting, 0, 1) {
		return NewError(ErrCodeHookFailed, "hook execution already in progress")
	}
	defer atomic.StoreInt32(&r.isExecuting, 0)

	// Get hooks sorted by priority
	hooks := r.getHooksSorted()

	if len(hooks) == 0 {
		return nil
	}

	r.mu.RLock()
	config := r.config
	r.mu.RUnlock()

	// Note: LastExecutionTime is updated in Stats() method

	if config.Parallel {
		return r.executeParallel(ctx, hooks)
	}

	return r.executeSequential(ctx, hooks)
}

// executeSequential executes hooks sequentially in priority order.
func (r *hookRegistry) executeSequential(ctx context.Context, hooks []*HookInfo) error {
	r.mu.RLock()
	config := r.config
	r.mu.RUnlock()

	for _, info := range hooks {
		if r.config.OnHookStart != nil {
			r.config.OnHookStart(info.Name)
		}

		err := r.executeHook(ctx, info)

		if err != nil {
			atomic.AddInt64(&r.stats.TotalFailures, 1)
			atomic.AddInt64(&r.stats.TotalExecutions, 1)

			if config.OnHookError != nil {
				config.OnHookError(info.Name, err)
			}

			if config.StopOnError {
				return fmt.Errorf("hook %s failed: %w", info.Name, err)
			}
		} else {
			atomic.AddInt64(&r.stats.TotalSuccesses, 1)
			atomic.AddInt64(&r.stats.TotalExecutions, 1)
		}

		if config.OnHookComplete != nil {
			config.OnHookComplete(info.Name, err)
		}
	}

	return nil
}

// executeParallel executes hooks in parallel.
func (r *hookRegistry) executeParallel(ctx context.Context, hooks []*HookInfo) error {
	r.mu.RLock()
	config := r.config
	r.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	// Create semaphore for concurrency control
	var semaphore chan struct{}
	if config.MaxConcurrency > 0 {
		semaphore = make(chan struct{}, config.MaxConcurrency)
	}

	for _, info := range hooks {
		info := info // capture loop variable

		if config.OnHookStart != nil {
			config.OnHookStart(info.Name)
		}

		// Only add to wait group for non-async hooks
		if !info.Async {
			wg.Add(1)
		}

		go func() {
			// Only decrement wait group for non-async hooks
			if !info.Async {
				defer wg.Done()
			}

			// Acquire semaphore if concurrency limit is set
			if semaphore != nil {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
			}

			err := r.executeHook(ctx, info)

			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("hook %s: %w", info.Name, err))
				atomic.AddInt64(&r.stats.TotalFailures, 1)
			} else {
				atomic.AddInt64(&r.stats.TotalSuccesses, 1)
			}
			atomic.AddInt64(&r.stats.TotalExecutions, 1)
			mu.Unlock()

			if err != nil && config.OnHookError != nil {
				config.OnHookError(info.Name, err)
			}

			if config.OnHookComplete != nil {
				config.OnHookComplete(info.Name, err)
			}
		}()
	}

	wg.Wait()

	if len(errors) > 0 && config.StopOnError {
		return fmt.Errorf("hook execution failed with %d error(s): %v", len(errors), errors)
	}

	return nil
}

// executeHook executes a single hook with timeout support.
func (r *hookRegistry) executeHook(ctx context.Context, info *HookInfo) error {
	// Determine timeout
	r.mu.RLock()
	defaultTimeout := r.config.DefaultTimeout
	r.mu.RUnlock()

	timeout := defaultTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			return NewError(ErrCodeExecutionTimeout, "context deadline exceeded")
		}
	}

	// Create execution context with timeout if needed
	execCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Execute hook
	var err error
	if info.Async {
		// For async hooks, execute in goroutine and don't wait
		go func() {
			hookErr := info.Hook(execCtx)
			atomic.AddInt64(&info.ExecutionCount, 1)
			
			// Protect field writes with mutex to prevent race conditions
			r.mu.Lock()
			info.LastExecutionTime = time.Now()
			info.LastError = hookErr
			r.mu.Unlock()
		}()
		return nil // Return immediately for async hooks
	}

	err = info.Hook(execCtx)
	atomic.AddInt64(&info.ExecutionCount, 1)
	
	// Protect field writes with mutex to prevent race conditions
	r.mu.Lock()
	info.LastExecutionTime = time.Now()
	info.LastError = err
	r.mu.Unlock()

	if err != nil {
		return fmt.Errorf("hook %s failed: %w", info.Name, err)
	}

	return nil
}

// ExecuteByName executes a specific hook by name.
func (r *hookRegistry) ExecuteByName(ctx context.Context, name string) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	r.mu.RLock()
	info, exists := r.hooks[name]
	r.mu.RUnlock()

	if !exists {
		return NewError(ErrCodeHookNotFound, fmt.Sprintf("hook %s not found", name))
	}

	r.mu.RLock()
	config := r.config
	r.mu.RUnlock()

	if config.OnHookStart != nil {
		config.OnHookStart(info.Name)
	}

	err := r.executeHook(ctx, info)

	if err != nil {
		atomic.AddInt64(&r.stats.TotalFailures, 1)
		if config.OnHookError != nil {
			config.OnHookError(info.Name, err)
		}
	} else {
		atomic.AddInt64(&r.stats.TotalSuccesses, 1)
	}

	atomic.AddInt64(&r.stats.TotalExecutions, 1)

	if config.OnHookComplete != nil {
		config.OnHookComplete(info.Name, err)
	}

	return err
}

// ExecuteByPriority executes hooks with a specific priority.
func (r *hookRegistry) ExecuteByPriority(ctx context.Context, priority int) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	r.mu.RLock()
	hooks := make([]*HookInfo, 0)
	for _, info := range r.hooks {
		if info.Priority == priority {
			hooks = append(hooks, info)
		}
	}
	r.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	// Sort by registration order (RegisteredAt)
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].RegisteredAt.Before(hooks[j].RegisteredAt)
	})

	r.mu.RLock()
	config := r.config
	r.mu.RUnlock()

	if config.Parallel {
		return r.executeParallel(ctx, hooks)
	}

	return r.executeSequential(ctx, hooks)
}

// GetHook returns information about a registered hook.
func (r *hookRegistry) GetHook(name string) (*HookInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.hooks[name]
	if !exists {
		return nil, NewError(ErrCodeHookNotFound, fmt.Sprintf("hook %s not found", name))
	}

	// Return a copy to prevent external modification
	copy := *info
	return &copy, nil
}

// ListHooks returns a list of all registered hooks, sorted by priority.
func (r *hookRegistry) ListHooks() []HookInfo {
	hooks := r.getHooksSorted()

	result := make([]HookInfo, len(hooks))
	for i, info := range hooks {
		// Return copies to prevent external modification
		result[i] = *info
	}

	return result
}

// getHooksSorted returns hooks sorted by priority and registration time.
func (r *hookRegistry) getHooksSorted() []*HookInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hooks := make([]*HookInfo, 0, len(r.hooks))
	for _, info := range r.hooks {
		hooks = append(hooks, info)
	}

	// Sort by priority (ascending), then by registration time (ascending)
	sort.Slice(hooks, func(i, j int) bool {
		if hooks[i].Priority != hooks[j].Priority {
			return hooks[i].Priority < hooks[j].Priority
		}
		return hooks[i].RegisteredAt.Before(hooks[j].RegisteredAt)
	})

	return hooks
}

// Stats returns statistics about hook execution.
func (r *hookRegistry) Stats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := r.stats
	stats.TotalHooks = atomic.LoadInt64(&r.stats.TotalHooks)
	stats.TotalExecutions = atomic.LoadInt64(&r.stats.TotalExecutions)
	stats.TotalSuccesses = atomic.LoadInt64(&r.stats.TotalSuccesses)
	stats.TotalFailures = atomic.LoadInt64(&r.stats.TotalFailures)
	stats.IsExecuting = atomic.LoadInt32(&r.isExecuting) == 1

	if v := r.lastExecutionTime.Load(); v != nil {
		stats.LastExecutionTime = v.(time.Time)
	}

	return stats
}

// Clear removes all registered hooks.
func (r *hookRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.hooks = make(map[string]*HookInfo)
	atomic.StoreInt64(&r.stats.TotalHooks, 0)
}
