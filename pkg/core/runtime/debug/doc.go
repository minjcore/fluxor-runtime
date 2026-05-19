// Package debug provides debug information collection and management for runtime components.
//
// The debug package allows you to:
//   - Enable/disable debug mode
//   - Collect stack traces and goroutine dumps
//   - Gather memory and garbage collection statistics
//   - Register custom debug data collectors
//   - Track debug operation statistics
//   - Collect data in parallel or sequentially
//   - Maintain collection history
//   - Filter goroutine dumps by count
//
// Example usage:
//
//	config := debug.DefaultConfig()
//	config.Enabled = true
//	config.ParallelCollect = true  // Collect custom data in parallel
//	config.HistorySize = 10        // Keep last 10 collections
//	config.MaxGoroutines = 100     // Limit goroutine dump to 100 goroutines
//	manager := debug.NewManager(config)
//
//	// Enable debug mode
//	manager.Enable()
//
//	// Collect debug information
//	info, err := manager.Collect(context.Background())
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Access collected information
//	fmt.Printf("Goroutines: %d\n", info.NumGoroutines)
//	fmt.Printf("Memory Alloc: %d bytes\n", info.MemoryStats.Alloc)
//	fmt.Printf("Collection took: %v\n", info.Duration)
//
//	// Register a custom collector
//	manager.RegisterCollector("custom", func(ctx context.Context) (interface{}, error) {
//		return map[string]interface{}{
//			"custom_metric": 42,
//		}, nil
//	})
//
//	// Access collection history
//	history := manager.History()
//	if len(history) > 0 {
//		fmt.Printf("Last collection: %v\n", history[len(history)-1].Timestamp)
//	}
//
//	// Get statistics
//	stats := manager.Stats()
//	fmt.Printf("Total collections: %d\n", stats.TotalCollections)
//	fmt.Printf("Average duration: %v\n", stats.AverageCollectionDuration)
//
// Package debug is part of the runtime package suite, providing debugging
// capabilities alongside state management, signal handling, drain operations, and health checks.
package debug
