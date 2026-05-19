// Package rate provides rate calculation and tracking.
//
// The rate package implements sliding window rate calculation to track
// the rate of events over time. This is useful for monitoring throughput,
// request rates, and other metrics that need to be calculated over time windows.
//
// The rate package implements configurable rate tracking with support for:
//   - Sliding window algorithm
//   - Configurable window size and granularity
//   - Real-time rate calculations
//   - Comprehensive statistics and metrics
//
// Basic Usage:
//
//	import (
//	    "context"
//	    "github.com/fluxorio/fluxor/pkg/core/resilience/rate"
//	)
//
//	// Create manager with default config (1 second window)
//	manager := rate.NewManager()
//
//	// Record events
//	manager.Record(ctx)
//	manager.Record(ctx)
//	manager.Record(ctx)
//
//	// Get current rate (events per second)
//	currentRate := manager.Rate()
//
// Advanced Usage with Custom Config:
//
//	config := rate.Config{
//	    Window:      time.Minute,  // 1 minute window
//	    Granularity: 60,           // 60 buckets (1 per second)
//	    OnRateCalculated: func(ctx context.Context, rate float64, window time.Duration) {
//	        log.Printf("Current rate: %.2f events per %v", rate, window)
//	    },
//	}
//
//	manager := rate.NewManagerWithConfig(config)
//
//	// Record events
//	manager.RecordN(ctx, 10) // Record 10 events at once
//
//	// Get current rate (events per minute)
//	currentRate := manager.Rate()
//
//	// Get rate for different window
//	ratePerSecond := manager.RateWithWindow(time.Second)
//
// Sliding Window Algorithm:
//
// The rate manager uses a sliding window algorithm with configurable granularity:
//   - Events are tracked in buckets (time slices)
//   - Window is divided into buckets based on granularity
//   - Old buckets are automatically cleared as time advances
//   - Rate is calculated by summing events in buckets within the window
//
// Window and Granularity:
//
//   - Window: The time window for rate calculation (e.g., 1 second, 1 minute)
//   - Granularity: Number of buckets used (higher = more accurate but more memory)
//   - Example: Window=1min, Granularity=60 → 60 buckets of 1 second each
//
// Thread Safety:
//
// The Manager is safe for concurrent use by multiple goroutines.
//
// Example: Request Rate Tracking
//
//	manager := rate.NewManagerWithConfig(rate.Config{
//	    Window:      time.Second,
//	    Granularity: 10, // 10 buckets of 100ms each
//	})
//
//	// Record each request
//	manager.Record(ctx)
//
//	// Get current requests per second
//	rps := manager.Rate()
//
// Example: Throughput Monitoring
//
//	manager := rate.NewManagerWithConfig(rate.Config{
//	    Window:      time.Minute,
//	    Granularity: 60, // 60 buckets of 1 second each
//	})
//
//	// Record bytes processed
//	manager.RecordN(ctx, bytesProcessed)
//
//	// Get bytes per minute
//	throughput := manager.Rate()
//
//	// Get bytes per second
//	throughputPerSecond := manager.RateWithWindow(time.Second)
package rate
