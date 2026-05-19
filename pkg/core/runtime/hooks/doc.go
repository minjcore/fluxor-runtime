// Package hooks provides a hook registry system for runtime lifecycle management.
//
// The hooks package provides a flexible system for registering and executing hooks
// (functions) that can be called at various points in an application's lifecycle.
// It supports priority-based execution ordering, synchronous and asynchronous hooks,
// parallel execution, and comprehensive error handling.
//
// Basic Usage:
//
//	// Create a hook registry
//	registry := hooks.NewRegistry(hooks.DefaultConfig())
//
//	// Register a hook
//	err := registry.Register("startup", func(ctx context.Context) error {
//	    log.Println("Application starting...")
//	    return nil
//	})
//
//	// Execute all hooks
//	ctx := context.Background()
//	err = registry.Execute(ctx)
//
// Priority-Based Execution:
//
//	// Hooks with lower priority execute first
//	registry.Register("hook1", hook1, hooks.WithPriority(10))
//	registry.Register("hook2", hook2, hooks.WithPriority(5))  // Executes first
//	registry.Register("hook3", hook3, hooks.WithPriority(15)) // Executes last
//
//	registry.Execute(ctx) // Executes: hook2, hook1, hook3
//
// Sequential vs Parallel Execution:
//
//	// Sequential execution (default)
//	config := hooks.DefaultConfig()
//	config.Parallel = false
//	registry := hooks.NewRegistry(config)
//
//	// Parallel execution
//	config.Parallel = true
//	config.MaxConcurrency = 5 // Limit concurrent executions
//	registry := hooks.NewRegistry(config)
//
// Async Hooks:
//
//	// Register an async hook that executes in background
//	registry.Register("async-hook", func(ctx context.Context) error {
//	    // This runs in a separate goroutine
//	    time.Sleep(1 * time.Second)
//	    return nil
//	}, hooks.WithAsync(true))
//
//	// Execute returns immediately, hook runs in background
//	registry.Execute(ctx)
//
// Error Handling:
//
//	// Stop execution on first error
//	config := hooks.DefaultConfig()
//	config.StopOnError = true
//	registry := hooks.NewRegistry(config)
//
//	// Continue execution even if hooks fail (default)
//	config.StopOnError = false
//	registry := hooks.NewRegistry(config)
//
// Execute Specific Hooks:
//
//	// Execute a single hook by name
//	err := registry.ExecuteByName(ctx, "startup-hook")
//
//	// Execute hooks with a specific priority
//	err := registry.ExecuteByPriority(ctx, 10)
//
// Callbacks:
//
//	config := hooks.Config{
//	    OnHookStart: func(name string) {
//	        log.Printf("Starting hook: %s", name)
//	    },
//	    OnHookComplete: func(name string, err error) {
//	        if err != nil {
//	            log.Printf("Hook %s failed: %v", name, err)
//	        }
//	    },
//	    OnHookError: func(name string, err error) {
//	        log.Printf("Error in hook %s: %v", name, err)
//	    },
//	}
//	registry := hooks.NewRegistry(config)
//
// Timeout Support:
//
//	// Set default timeout for all hooks
//	config := hooks.DefaultConfig()
//	config.DefaultTimeout = 30 * time.Second
//	registry := hooks.NewRegistry(config)
//
//	// Or use context timeout
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	registry.Execute(ctx)
//
// Statistics:
//
//	// Get execution statistics
//	stats := registry.Stats()
//	log.Printf("Total hooks: %d", stats.TotalHooks)
//	log.Printf("Total executions: %d", stats.TotalExecutions)
//	log.Printf("Successes: %d", stats.TotalSuccesses)
//	log.Printf("Failures: %d", stats.TotalFailures)
//
// Hook Information:
//
//	// Get information about a specific hook
//	info, err := registry.GetHook("startup-hook")
//	if err == nil {
//	    log.Printf("Hook: %s, Priority: %d, Executions: %d",
//	        info.Name, info.Priority, info.ExecutionCount)
//	}
//
//	// List all hooks
//	hooks := registry.ListHooks()
//	for _, hook := range hooks {
//	    log.Printf("Hook: %s (priority: %d)", hook.Name, hook.Priority)
//	}
//
// Lifecycle Management:
//
//	// Register hooks for different lifecycle stages
//	registry.Register("pre-start", preStartHook, hooks.WithPriority(1))
//	registry.Register("start", startHook, hooks.WithPriority(10))
//	registry.Register("post-start", postStartHook, hooks.WithPriority(20))
//
//	// Execute in order
//	registry.Execute(ctx)
//
//	// Cleanup
//	registry.Unregister("pre-start")
//	registry.Clear() // Remove all hooks
//
// The Registry interface provides:
//   - Register() - Register a hook with optional priority and async settings
//   - Unregister() - Remove a hook from the registry
//   - Execute() - Execute all hooks in priority order
//   - ExecuteByName() - Execute a specific hook by name
//   - ExecuteByPriority() - Execute hooks with a specific priority
//   - GetHook() - Get information about a registered hook
//   - ListHooks() - List all registered hooks
//   - Stats() - Get execution statistics
//   - Clear() - Remove all hooks
//
// All methods are thread-safe and can be used concurrently.
package hooks
