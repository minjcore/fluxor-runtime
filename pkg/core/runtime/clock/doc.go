// Package clock provides an abstraction over time operations.
//
// The clock package provides a Clock interface that abstracts time operations,
// allowing for both system time and mockable time implementations. This is
// particularly useful for testing, where you need to control time advancement.
//
// Usage:
//
//	// Use system clock (default)
//	clock := clock.NewSystemClock()
//	now := clock.Now()
//	clock.Sleep(1 * time.Second)
//
//	// Use mock clock for testing
//	mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
//	mockClock.Advance(1 * time.Hour)
//	future := mockClock.Now()
//
// The Clock interface provides:
//   - Now() - Get current time
//   - Sleep() - Sleep for a duration
//   - After() - Get a channel that fires after a duration
//   - AfterFunc() - Execute a function after a duration
//   - NewTimer() - Create a new timer
//   - NewTicker() - Create a new ticker
//   - Since() - Get elapsed time since a point
//   - Until() - Get time until a point
//
// All implementations are thread-safe and can be used concurrently.
package clock
