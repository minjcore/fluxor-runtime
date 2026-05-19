// Package quota provides quota management and enforcement for runtime resources.
//
// The quota package provides utilities for tracking and enforcing resource quotas
// such as requests, memory, CPU, or custom quotas. It supports quota registration,
// acquisition/release, window-based resets, and comprehensive statistics tracking.
// All operations are thread-safe and designed for concurrent use.
//
// Quota Types:
//
// The package supports different quota types:
//   - QuotaTypeRequests: Request-based quotas
//   - QuotaTypeMemory: Memory-based quotas
//   - QuotaTypeCPU: CPU-based quotas
//   - QuotaTypeCustom: Custom quotas
//
// Basic Usage:
//
//	// Create a quota manager with default configuration
//	manager := quota.NewManager(quota.DefaultConfig())
//
//	// Register a quota
//	err := manager.Register("api-requests", quota.QuotaTypeRequests, 1000, time.Minute)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Register with options (options are optional)
//	err = manager.Register("custom", quota.QuotaTypeCustom, 500, time.Minute)
//
//	// Acquire quota
//	ctx := context.Background()
//	acquired, usage, remaining, err := manager.Acquire(ctx, "api-requests", 10)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if !acquired {
//	    // Quota exceeded
//	    log.Println("Quota exceeded")
//	    return
//	}
//
//	log.Printf("Quota acquired: usage=%d, remaining=%d", usage, remaining)
//
//	// Release quota when done
//	err = manager.Release("api-requests", 10)
//
// Advanced Usage with Callbacks:
//
//	// Create manager with callbacks
//	config := quota.Config{
//	    OnQuotaExceeded: func(name string, quotaType quota.QuotaType, limit, usage int64) {
//	        log.Printf("Quota %s exceeded: usage=%d/%d", name, usage, limit)
//	    },
//	    OnQuotaExceededAsync: func(name string, quotaType quota.QuotaType, limit, usage int64) {
//	        // Non-blocking async callback
//	        metrics.RecordQuotaExceeded(name, usage, limit)
//	    },
//	    OnQuotaReset: func(name string) {
//	        log.Printf("Quota %s reset", name)
//	    },
//	}
//
//	manager := quota.NewManager(config)
//
// Window-Based Resets:
//
//	// Register quota with time window (resets automatically)
//	err := manager.Register("requests", quota.QuotaTypeRequests, 100, time.Minute)
//
//	// Quota automatically resets when window expires
//	// on next Acquire() call after window has passed
//
//	ctx := context.Background()
//	manager.Acquire(ctx, "requests", 50)
//
//	// Wait for window to expire
//	time.Sleep(time.Minute)
//
//	// Next acquire resets the quota
//	acquired, usage, _, _ := manager.Acquire(ctx, "requests", 10)
//	// usage will be 10, not 60 (quota was reset)
//
// Auto-Reset Configuration:
//
//	// Configure automatic reset of all quotas
//	config := quota.DefaultConfig()
//	config.AutoResetInterval = 5 * time.Minute
//	manager := quota.NewManager(config)
//
//	// All quotas will be automatically reset every 5 minutes
//
// Quota Statistics:
//
//	// Get overall statistics
//	stats := manager.Stats()
//	log.Printf("Total quotas: %d", stats.TotalQuotas)
//	log.Printf("Total acquired: %d", stats.TotalAcquired)
//	log.Printf("Total exceeded: %d", stats.TotalExceeded)
//	log.Printf("Total released: %d", stats.TotalReleased)
//
//	// Get statistics for a specific quota
//	quotaStats, err := manager.QuotaStats("api-requests")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	log.Printf("Quota: %s", quotaStats.Name)
//	log.Printf("Usage: %d/%d", quotaStats.Usage, quotaStats.Limit)
//	log.Printf("Remaining: %d", quotaStats.Remaining)
//
// Manual Reset:
//
//	// Reset a specific quota
//	err := manager.Reset("api-requests")
//
//	// Reset all quotas
//	err = manager.ResetAll()
//
// Unlimited Quotas:
//
//	// Register an unlimited quota (limit = 0)
//	err := manager.Register("unlimited", quota.QuotaTypeRequests, 0, time.Minute)
//
//	// Acquire will always succeed
//	acquired, _, remaining, _ := manager.Acquire(ctx, "unlimited", 1000)
//	// acquired = true, remaining = -1 (indicates unlimited)
//
// Multiple Quota Types:
//
//	// Register different types of quotas
//	manager.Register("requests", quota.QuotaTypeRequests, 1000, time.Minute)
//	manager.Register("memory", quota.QuotaTypeMemory, 1024*1024*1024, time.Hour)
//	manager.Register("cpu", quota.QuotaTypeCPU, 80, time.Minute)
//
//	// Manage each independently
//	manager.Acquire(ctx, "requests", 10)
//	manager.Acquire(ctx, "memory", 1024*1024)
//	manager.Acquire(ctx, "cpu", 5)
//
// Usage Tracking:
//
//	// Get current usage
//	usage, err := manager.GetUsage("api-requests")
//
//	// Get remaining quota
//	remaining, err := manager.GetRemaining("api-requests")
//
//	// List all registered quotas (returns QuotaInfo slice, sorted by name)
//	quotas := manager.ListQuotas()
//	for _, q := range quotas {
//	    log.Printf("Quota: %s, Usage: %d/%d, Remaining: %d",
//	        q.Name, q.Usage, q.Limit, q.Remaining)
//	}
//
// Get Quota Information:
//
//	// Get detailed information about a quota
//	info, err := manager.GetQuota("api-requests")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Quota: %s, Type: %s, Limit: %d, Usage: %d",
//	    info.Name, info.Type, info.Limit, info.Usage)
//
// Clear All Quotas:
//
//	// Remove all quotas at once
//	manager.Clear()
//
// Thread Safety:
//
// All Manager methods are thread-safe and can be called concurrently.
// Quota operations use atomic operations and mutex locks to ensure consistency
// under concurrent access.
//
// The Manager interface provides:
//   - Register() - Register a new quota (with optional options)
//   - Unregister() - Remove a quota
//   - Acquire() - Attempt to acquire quota (returns success, usage, remaining)
//   - Release() - Release quota
//   - GetUsage() - Get current usage for a quota
//   - GetRemaining() - Get remaining quota
//   - Reset() - Reset a quota
//   - ResetAll() - Reset all quotas
//   - Clear() - Remove all quotas
//   - Stats() - Get overall statistics
//   - GetQuota() - Get information about a specific quota
//   - QuotaStats() - Get statistics for a specific quota
//   - ListQuotas() - List all registered quotas (returns QuotaInfo slice)
package quota
