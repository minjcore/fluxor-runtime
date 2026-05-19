// Package fallback provides resilience through alternative execution paths.
//
// The fallback pattern provides an alternative execution path when the primary
// operation fails. This ensures system resilience by allowing graceful degradation
// instead of complete failure.
//
// The fallback package implements configurable fallback strategies with support for:
//   - Multiple fallback functions executed in sequence
//   - Conditional fallback execution based on error predicates
//   - Comprehensive lifecycle callbacks
//   - Detailed statistics and metrics
//
// Basic Usage:
//
//	import (
//	    "context"
//	    "github.com/fluxorio/fluxor/pkg/core/resilience/fallback"
//	)
//
//	// Create manager with default config
//	manager := fallback.NewManager()
//
//	// Execute with fallback
//	err := manager.Execute(ctx, func(ctx context.Context) error {
//	    // Primary operation
//	    return primaryOperation(ctx)
//	}, func(ctx context.Context) error {
//	    // Fallback operation
//	    return fallbackOperation(ctx)
//	})
//
// Advanced Usage with Multiple Fallbacks:
//
//	config := fallback.Config{
//	    Fallbacks: []fallback.Executable{
//	        fallbackFunction1,
//	        fallbackFunction2,
//	        fallbackFunction3,
//	    },
//	    Predicate: fallback.FallbackOnTimeout, // Only fallback on timeout errors
//	    OnPrimaryError: func(ctx context.Context, err error) {
//	        log.Printf("Primary operation failed: %v", err)
//	    },
//	    OnFallbackSucceeded: func(ctx context.Context, index int, result error) {
//	        log.Printf("Fallback %d succeeded", index)
//	    },
//	    OnAllFallbacksExhausted: func(ctx context.Context, primaryErr error, fallbackErrors []error) {
//	        log.Printf("All fallbacks exhausted: primary=%v, fallbacks=%v", primaryErr, fallbackErrors)
//	    },
//	}
//
//	manager := fallback.NewManager()
//	err := manager.ExecuteWithConfig(ctx, primaryFunction, config)
//
// Fallback Predicates:
//
// Predicates allow you to control when fallback should be executed:
//
//   - AlwaysFallback: Always execute fallback on any error (default)
//   - NeverFallback: Never execute fallback
//   - FallbackOnTimeout: Only fallback on timeout/deadline errors
//   - FallbackOnTemporary: Only fallback on temporary errors
//   - FallbackOnNetworkError: Only fallback on network errors
//   - FallbackOnErrorType: Fallback if error is of specific type
//   - FallbackOnErrorMessage: Fallback if error message contains substring
//   - FallbackOnAny: Fallback if any predicate matches (OR)
//   - FallbackOnAll: Fallback only if all predicates match (AND)
//   - FallbackOnNot: Fallback if predicate doesn't match (NOT)
//
// Example with Predicate:
//
//	config := fallback.Config{
//	    Fallbacks: []fallback.Executable{fallbackFunction},
//	    Predicate: fallback.FallbackOnAny(
//	        fallback.FallbackOnTimeout,
//	        fallback.FallbackOnNetworkError,
//	    ), // Fallback only on timeout or network errors
//	}
//
// Execution Flow:
//
// 1. Primary function is executed
// 2. If primary succeeds, return success (no fallback)
// 3. If primary fails:
//    a. Check predicate (if configured) - if false, return primary error
//    b. Try fallbacks in order until one succeeds or all fail
//    c. If all fallbacks fail, return ErrCodeAllFallbacksExhausted
//
// Thread Safety:
//
// The Manager is safe for concurrent use by multiple goroutines.
//
// Example: Database with Cache Fallback
//
//	manager := fallback.NewManager()
//
//	err := manager.ExecuteWithFallback(ctx,
//	    // Primary: Database query
//	    func(ctx context.Context) error {
//	        result, err := db.Query(ctx, query)
//	        if err != nil {
//	            return err
//	        }
//	        // Process result...
//	        return nil
//	    },
//	    // Fallback: Cache lookup
//	    func(ctx context.Context) error {
//	        result, err := cache.Get(ctx, cacheKey)
//	        if err != nil {
//	            return err
//	        }
//	        // Process cached result...
//	        return nil
//	    },
//	)
package fallback
