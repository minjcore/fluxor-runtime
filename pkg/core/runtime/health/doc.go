// Package health provides health checking for runtime components.
//
// The health package allows you to:
//   - Perform health checks on runtime components
//   - Register custom health checkers
//   - Monitor runtime metrics (memory, goroutines, GC)
//   - Set thresholds for runtime metrics
//   - Track health check statistics
//
// Example usage:
//
//	config := health.DefaultConfig()
//	manager := health.NewManager(config)
//
//	// Perform a health check
//	result, err := manager.Check(context.Background())
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if result.Healthy {
//		fmt.Println("System is healthy")
//	} else {
//		fmt.Printf("System is unhealthy: %s\n", result.Message)
//	}
//
//	// Register a custom health checker
//	manager.RegisterChecker("database", func(ctx context.Context) error {
//		// Check database connection
//		return db.PingContext(ctx)
//	})
//
//	// Register a threshold for goroutines
//	manager.RegisterThreshold("goroutines", health.Threshold{
//		MaxGoroutines: 1000,
//	})
//
//	// Register a threshold for memory
//	manager.RegisterThreshold("memory", health.Threshold{
//		MaxMemoryAlloc: 100 * 1024 * 1024, // 100MB
//	})
//
//	// Check if system is healthy
//	healthy, err := manager.IsHealthy(context.Background())
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if !healthy {
//		log.Fatal("System is unhealthy")
//	}
//
// Package health is part of the runtime package suite, providing health checking
// capabilities alongside state management, signal handling, drain operations, and debugging.
package health
