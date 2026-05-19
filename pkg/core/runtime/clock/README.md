# Clock Package

The `clock` package provides an abstraction over time operations, enabling both system time and mockable time implementations. This is particularly useful for testing, where you need to control time advancement deterministically.

## Features

- **System Clock**: Production-ready implementation using the system time
- **Mock Clock**: Test-friendly implementation with controllable time
- **Timer Support**: Create and manage timers with `NewTimer()` and `AfterFunc()`
- **Ticker Support**: Create periodic tickers with `NewTicker()`
- **Time Utilities**: `Since()` and `Until()` helpers
- **Thread-Safe**: All implementations are safe for concurrent use

## Installation

```go
import "github.com/fluxorio/fluxor/pkg/core/runtime/clock"
```

## Basic Usage

### System Clock (Production)

```go
import (
    "github.com/fluxorio/fluxor/pkg/core/runtime/clock"
    "time"
)

// Create a system clock
clock := clock.NewSystemClock()

// Get current time
now := clock.Now()

// Sleep for a duration
clock.Sleep(1 * time.Second)

// Calculate elapsed time
start := clock.Now()
// ... do work ...
elapsed := clock.Since(start)

// Calculate time until a future point
deadline := clock.Now().Add(5 * time.Minute)
remaining := clock.Until(deadline)
```

### Mock Clock (Testing)

```go
import (
    "github.com/fluxorio/fluxor/pkg/core/runtime/clock"
    "time"
)

// Create a mock clock with a specific initial time
initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
mockClock := clock.NewMockClock(initialTime)

// Get current time
now := mockClock.Now() // Returns the initial time

// Advance time by a duration
mockClock.Advance(1 * time.Hour)
future := mockClock.Now() // Returns initial time + 1 hour

// Set time to a specific value
newTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
mockClock.SetTime(newTime)
now = mockClock.Now() // Returns newTime

// Sleep advances time immediately (non-blocking in mock)
mockClock.Sleep(30 * time.Minute)
```

## Timer Operations

### Using Timers

```go
clock := clock.NewSystemClock()

// Create a timer that fires after 5 seconds
timer := clock.NewTimer(5 * time.Second)

// Wait for timer to fire
select {
case <-timer.C():
    fmt.Println("Timer fired!")
case <-time.After(10 * time.Second):
    fmt.Println("Timeout waiting for timer")
}

// Stop a timer before it fires
timer := clock.NewTimer(5 * time.Second)
if timer.Stop() {
    fmt.Println("Timer stopped successfully")
}

// Reset a timer
timer := clock.NewSystemClock().NewTimer(10 * time.Second)
timer.Stop()
timer.Reset(5 * time.Second) // Reset to fire after 5 seconds instead
```

### Using After

```go
clock := clock.NewSystemClock()

// Wait for a duration and receive the time
ch := clock.After(2 * time.Second)
select {
case t := <-ch:
    fmt.Printf("Time after 2 seconds: %v\n", t)
}
```

### Using AfterFunc

```go
clock := clock.NewSystemClock()

// Execute a function after a duration
timer := clock.AfterFunc(3 * time.Second, func() {
    fmt.Println("Callback executed!")
})

// The callback will be executed in its own goroutine after 3 seconds
// You can stop it before execution
if timer.Stop() {
    fmt.Println("Callback cancelled")
}
```

## Ticker Operations

### Using Tickers

```go
clock := clock.NewSystemClock()

// Create a ticker that fires every second
ticker := clock.NewTicker(1 * time.Second)
defer ticker.Stop()

// Receive ticks
for i := 0; i < 5; i++ {
    select {
    case t := <-ticker.C():
        fmt.Printf("Tick %d at %v\n", i+1, t)
    }
}
```

## Testing with Mock Clock

The mock clock is particularly useful for testing time-dependent code:

```go
func TestTimeout(t *testing.T) {
    // Create mock clock starting at a known time
    mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
    
    // Create a timer
    timer := mockClock.NewTimer(1 * time.Hour)
    
    // Initially, timer should not fire
    select {
    case <-timer.C():
        t.Error("Timer should not fire yet")
    default:
        // Expected - timer hasn't fired
    }
    
    // Advance time by 1 hour
    mockClock.Advance(1 * time.Hour)
    
    // Now timer should fire
    select {
    case <-timer.C():
        // Success!
    case <-time.After(100 * time.Millisecond):
        t.Error("Timer should have fired after advancing time")
    }
}

func TestTicker(t *testing.T) {
    mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
    ticker := mockClock.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    
    // Advance by 3 hours - should get 3 ticks
    mockClock.Advance(3 * time.Hour)
    
    tickCount := 0
    for i := 0; i < 3; i++ {
        select {
        case <-ticker.C():
            tickCount++
        case <-time.After(100 * time.Millisecond):
            t.Errorf("Expected tick %d", i+1)
        }
    }
    
    if tickCount != 3 {
        t.Errorf("Expected 3 ticks, got %d", tickCount)
    }
}
```

### Testing Multiple Timers

```go
func TestMultipleTimers(t *testing.T) {
    mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
    
    // Create multiple timers with different durations
    timer1 := mockClock.NewTimer(1 * time.Hour)
    timer2 := mockClock.NewTimer(2 * time.Hour)
    timer3 := mockClock.NewTimer(30 * time.Minute)
    
    // Advance by 30 minutes - only timer3 should fire
    mockClock.Advance(30 * time.Minute)
    
    select {
    case <-timer3.C():
        // Success
    default:
        t.Error("Timer3 should fire after 30 minutes")
    }
    
    // Advance by another 30 minutes - timer1 should fire
    mockClock.Advance(30 * time.Minute)
    
    select {
    case <-timer1.C():
        // Success
    default:
        t.Error("Timer1 should fire after 1 hour")
    }
    
    // Advance by another hour - timer2 should fire
    mockClock.Advance(1 * time.Hour)
    
    select {
    case <-timer2.C():
        // Success
    default:
        t.Error("Timer2 should fire after 2 hours")
    }
}
```

## Advanced Patterns

### Dependency Injection

Inject a clock into your components for testability:

```go
type Service struct {
    clock clock.Clock
}

func NewService(c clock.Clock) *Service {
    return &Service{clock: c}
}

func (s *Service) ProcessWithTimeout(timeout time.Duration) error {
    deadline := s.clock.Now().Add(timeout)
    
    for s.clock.Now().Before(deadline) {
        // Do work
        if done {
            return nil
        }
        s.clock.Sleep(100 * time.Millisecond)
    }
    
    return errors.New("timeout")
}

// Production use
service := NewService(clock.NewSystemClock())

// Testing use
mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
testService := NewService(mockClock)
// Control time in tests
mockClock.Advance(5 * time.Second)
```

### Retry with Backoff

```go
func RetryWithBackoff(c clock.Clock, maxAttempts int, initialDelay time.Duration) error {
    delay := initialDelay
    
    for i := 0; i < maxAttempts; i++ {
        if err := doWork(); err == nil {
            return nil
        }
        
        if i < maxAttempts-1 {
            c.Sleep(delay)
            delay *= 2 // Exponential backoff
        }
    }
    
    return errors.New("max attempts exceeded")
}
```

### Rate Limiting

```go
type RateLimiter struct {
    clock    clock.Clock
    interval time.Duration
    lastTime time.Time
    mu       sync.Mutex
}

func (rl *RateLimiter) Allow() bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    now := rl.clock.Now()
    if now.Sub(rl.lastTime) >= rl.interval {
        rl.lastTime = now
        return true
    }
    
    return false
}
```

## Interface Reference

### Clock Interface

```go
type Clock interface {
    // Now returns the current time
    Now() time.Time
    
    // Sleep pauses the current goroutine for at least the duration d
    Sleep(d time.Duration)
    
    // After waits for the duration to elapse and sends the current time
    After(d time.Duration) <-chan time.Time
    
    // AfterFunc waits for the duration and calls f in its own goroutine
    AfterFunc(d time.Duration, f func()) Timer
    
    // NewTimer creates a new Timer that sends time after duration d
    NewTimer(d time.Duration) Timer
    
    // NewTicker returns a new Ticker that sends time at intervals
    NewTicker(d time.Duration) Ticker
    
    // Since returns the time elapsed since t
    Since(t time.Time) time.Duration
    
    // Until returns the duration until t
    Until(t time.Time) time.Duration
}
```

### Timer Interface

```go
type Timer interface {
    // C returns the channel on which the time is delivered
    C() <-chan time.Time
    
    // Stop prevents the Timer from firing
    Stop() bool
    
    // Reset changes the timer to expire after duration d
    Reset(d time.Duration) bool
}
```

### Ticker Interface

```go
type Ticker interface {
    // C returns the channel on which the ticks are delivered
    C() <-chan time.Time
    
    // Stop stops a ticker
    Stop()
}
```

## Mock Clock API

### NewMockClock

```go
func NewMockClock(initialTime time.Time) *MockClock
```

Creates a new mock clock with the specified initial time.

### Advance

```go
func (c *MockClock) Advance(d time.Duration)
```

Advances the clock by the given duration and triggers any timers or tickers that should fire.

### SetTime

```go
func (c *MockClock) SetTime(t time.Time)
```

Sets the current time to the given time.

## Best Practices

1. **Use Dependency Injection**: Pass a `Clock` interface to your components rather than using `time.Now()` directly
2. **Mock in Tests**: Always use `MockClock` in tests to control time deterministically
3. **System Clock in Production**: Use `NewSystemClock()` in production code
4. **Don't Mix Clocks**: Use the same clock instance throughout a component's lifetime
5. **Clean Up Resources**: Always call `Stop()` on timers and tickers when done

## Thread Safety

All clock implementations are thread-safe and can be used concurrently from multiple goroutines. The mock clock uses mutexes to protect its internal state.

## Performance Considerations

- **System Clock**: Uses Go's standard `time` package, which is highly optimized
- **Mock Clock**: Overhead is minimal; mutex contention only occurs during `Advance()` and `SetTime()`
- **Timers/Tickers**: Mock clock timers fire synchronously during `Advance()`, making tests fast and deterministic

## See Also

- [Go time package documentation](https://pkg.go.dev/time)
- [Testing best practices](../README.md#testing)
