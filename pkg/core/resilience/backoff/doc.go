// Package backoff provides backoff strategies for delay calculations.
//
// The backoff package implements various backoff strategies for calculating
// delays between retries, operations, or any other repeated attempts. This is
// useful for implementing exponential backoff, linear backoff, or fixed delays
// in retry logic, rate limiting, or other resilience patterns.
//
// The backoff package implements configurable backoff strategies with support for:
//   - Fixed backoff (constant delay)
//   - Exponential backoff (delay increases exponentially)
//   - Linear backoff (delay increases linearly)
//   - Jitter support (random variation to prevent thundering herd)
//   - Configurable max delay and max attempts
//   - Context-aware waiting
//
// Basic Usage:
//
//	import (
//	    "context"
//	    "github.com/fluxorio/fluxor/pkg/core/resilience/backoff"
//	)
//
//	// Create manager with fixed backoff (1 second)
//	manager := backoff.NewManager()
//
//	// Wait for backoff delay
//	err := manager.Wait(ctx, attempt)
//
// Advanced Usage with Exponential Backoff:
//
//	strategy := backoff.NewExponentialBackoff(
//	    100*time.Millisecond, // Initial delay
//	    10*time.Second,       // Max delay
//	    2.0,                  // Multiplier
//	    true,                 // With jitter
//	)
//
//	manager := backoff.NewManagerWithStrategy(strategy)
//
//	// Wait for backoff delay
//	err := manager.Wait(ctx, 3) // Wait for attempt 3
//
// Custom Config:
//
//	config := backoff.Config{
//	    Strategy: backoff.NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 2.0, true),
//	    MaxAttempts: 10,
//	    MaxDelay: 5*time.Second,
//	    OnBackoff: func(ctx context.Context, attempt int, delay time.Duration) {
//	        log.Printf("Backing off: attempt %d, delay %v", attempt, delay)
//	    },
//	}
//
//	err := manager.WaitWithConfig(ctx, attempt, config)
//
// Backoff Strategies:
//
// 1. FixedBackoff: Constant delay for all attempts
//    - Use when you want a consistent delay
//    - Example: Wait 1 second between all attempts
//
// 2. ExponentialBackoff: Delay increases exponentially
//    - Use for retries where you want increasing delays
//    - Formula: initial * multiplier^attempt
//    - Example: 100ms, 200ms, 400ms, 800ms...
//
// 3. LinearBackoff: Delay increases linearly
//    - Use when you want gradual increases
//    - Formula: initial + (attempt * increment)
//    - Example: 100ms, 200ms, 300ms, 400ms...
//
// Jitter:
//
// Exponential backoff supports jitter to prevent thundering herd problems.
// When enabled, random variation (±25%) is added to the delay to spread out
// retry attempts across multiple clients.
//
// Non-Blocking Delay Calculation:
//
//	// Calculate delay without waiting
//	delay := manager.Delay(attempt)
//
//	// Use in custom logic
//	select {
//	case <-ctx.Done():
//	    return ctx.Err()
//	case <-time.After(delay):
//	    // Continue
//	}
//
// Thread Safety:
//
// Strategies are safe for concurrent use by multiple goroutines.
//
// Example: Exponential Backoff for Retries
//
//	manager := backoff.NewManagerWithStrategy(
//	    backoff.NewExponentialBackoff(
//	        100*time.Millisecond,
//	        10*time.Second,
//	        2.0,
//		true,
//	    ),
//	)
//
//	for attempt := 0; attempt < maxRetries; attempt++ {
//	    err := operation()
//	    if err == nil {
//	        break
//	    }
//
//	    // Wait before next retry
//	    if err := manager.Wait(ctx, attempt); err != nil {
//	        return err
//	    }
//	}
package backoff
