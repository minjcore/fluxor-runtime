# Signals Package

The `signals` package provides signal handling for graceful shutdown of applications. It supports handling OS signals (SIGINT, SIGTERM, etc.) with features like callbacks, timeouts, signal statistics, and signal history tracking.

## Features

- **Signal Handling**: Listen for OS signals (SIGINT, SIGTERM, SIGINT, etc.)
- **Callbacks**: Synchronous and asynchronous callbacks on signal receipt
- **Graceful Shutdown**: Built-in support for graceful shutdown patterns
- **Timeout Support**: Configurable timeouts for shutdown operations
- **Signal Statistics**: Track signal count, last signal, and timestamps
- **Signal History**: Optional history tracking of received signals
- **Signal Queuing**: Configurable queue to prevent signal loss
- **Continue Mode**: Option to continue listening after receiving signals
- **Thread-Safe**: All operations are safe for concurrent use
- **Context Support**: Full integration with Go contexts

## Installation

```go
import "github.com/fluxorio/fluxor/pkg/core/runtime/signals"
```

## Basic Usage

### Simple Signal Wait

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core/runtime/signals"
    "log"
    "os"
)

func main() {
    ctx := context.Background()
    
    // Wait for a signal (blocks until signal received or context cancelled)
    sig, err := signals.WaitForSignal(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Received signal: %v", sig)
    // Perform cleanup...
}
```

### Wait with Timeout

```go
ctx := context.Background()

// Wait for signal with 30 second timeout
sig, err := signals.WaitForSignalWithTimeout(ctx, 30*time.Second)
if err != nil {
    if err == context.DeadlineExceeded {
        log.Println("Timeout waiting for signal")
    } else {
        log.Fatalf("Error: %v", err)
    }
} else {
    log.Printf("Received signal: %v", sig)
}
```

### Manual Handler Management

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core/runtime/signals"
    "log"
    "os"
    "syscall"
)

func main() {
    // Create handler with default configuration
    config := signals.DefaultConfig()
    handler := signals.NewHandler(config)
    
    ctx := context.Background()
    
    // Start listening for signals
    if err := handler.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer handler.Stop()
    
    // Wait for signal
    sig, err := handler.Wait(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Received signal: %v", sig)
}
```

## Advanced Usage

### Callback-Based Signal Handling

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core/runtime/signals"
    "log"
    "os"
    "syscall"
)

func main() {
    // Configure handler with callback
    config := signals.Config{
        Signals: []os.Signal{os.Interrupt, syscall.SIGTERM},
        OnSignal: func(sig os.Signal) {
            log.Printf("Received signal: %v", sig)
            // Perform cleanup synchronously
            cleanup()
        },
    }
    
    handler := signals.NewHandler(config)
    ctx := context.Background()
    
    if err := handler.Start(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Handler will call OnSignal when signal is received
    // Then wait for it to complete
    sig, err := handler.Wait(ctx)
    log.Printf("Signal handling complete: %v", sig)
}

func cleanup() {
    // Your cleanup logic here
}
```

### Asynchronous Callbacks

```go
config := signals.Config{
    OnSignalAsync: func(sig os.Signal) {
        // This callback runs in a separate goroutine
        // Non-blocking - doesn't delay signal handling
        log.Printf("Async processing signal: %v", sig)
        
        // Perform async cleanup
        go performAsyncCleanup()
    },
}

handler := signals.NewHandler(config)
handler.Start(ctx)
```

### Listen Helper Function

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core/runtime/signals"
    "os"
)

ctx := context.Background()

// Simple listen with callback
err := signals.Listen(ctx, signals.DefaultConfig(), func(sig os.Signal) {
    log.Printf("Signal received: %v", sig)
    // Handle signal
})

// Listen with async callback
err = signals.ListenAsync(ctx, signals.DefaultConfig(), func(sig os.Signal) {
    log.Printf("Async signal: %v", sig)
})
```

## Graceful Shutdown

### Basic Graceful Shutdown

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core/runtime/signals"
    "log"
)

func main() {
    ctx := context.Background()
    
    // Setup graceful shutdown
    err := signals.GracefulShutdown(ctx, func() {
        log.Println("Shutting down gracefully...")
        
        // Close database connections
        db.Close()
        
        // Flush logs
        log.Flush()
        
        // Save state
        saveState()
        
        log.Println("Shutdown complete")
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Application continues running...
    // When SIGINT/SIGTERM is received, the callback is executed
}
```

### Graceful Shutdown with Timeout

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core/runtime/signals"
    "log"
    "time"
)

func main() {
    ctx := context.Background()
    
    // Setup graceful shutdown with 30 second timeout
    err := signals.GracefulShutdown(ctx, func() {
        log.Println("Starting graceful shutdown...")
        
        // Perform cleanup that might take time
        cleanup()
        
        log.Println("Shutdown complete")
    },
        signals.WithShutdownTimeout(30*time.Second),
    )
    
    if err != nil {
        log.Fatal(err)
    }
}
```

### Graceful Shutdown with Custom Signals

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core/runtime/signals"
    "os"
    "syscall"
)

ctx := context.Background()

err := signals.GracefulShutdown(ctx, func() {
    cleanup()
},
    signals.WithSignals(os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1),
    signals.WithShutdownTimeout(10*time.Second),
)
```

## Signal Statistics

### Get Signal Statistics

```go
handler := signals.NewHandler(signals.DefaultConfig())
handler.Start(ctx)
defer handler.Stop()

// ... application runs ...

// Get statistics
stats := handler.Stats()

log.Printf("Signals received: %d", stats.SignalCount)
log.Printf("Last signal: %v", stats.LastSignal)
log.Printf("Last signal time: %v", stats.LastSignalTime)
log.Printf("Handler started: %v", stats.IsStarted)
log.Printf("Handler stopped: %v", stats.IsStopped)
```

### Monitor Signal Statistics

```go
func monitorSignals(handler signals.Handler, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for range ticker.C {
        stats := handler.Stats()
        
        if stats.SignalCount > 0 {
            log.Printf("Total signals: %d, Last: %v at %v",
                stats.SignalCount,
                stats.LastSignal,
                stats.LastSignalTime,
            )
        }
    }
}
```

## Signal History

### Enable Signal History Tracking

```go
config := signals.DefaultConfig()
config.SignalHistory = true  // Enable history tracking
config.QueueSize = 50        // Store up to 50 signals

handler := signals.NewHandler(config)
handler.Start(ctx)
defer handler.Stop()

// ... receive signals ...

// Get signal history
handlerImpl := handler.(*signals.signalHandler)
history := handlerImpl.GetHistory()

for i, sig := range history {
    log.Printf("Signal %d: %v", i, sig)
}
```

## Continue On Signal

### Multiple Signal Handling

```go
config := signals.DefaultConfig()
config.ContinueOnSignal = true  // Continue listening after each signal

handler := signals.NewHandler(config)
handler.Start(ctx)
defer handler.Stop()

// Handle multiple signals
for {
    sig, err := handler.Wait(ctx)
    if err != nil {
        log.Printf("Error: %v", err)
        break
    }
    
    log.Printf("Received signal: %v", sig)
    
    // Process signal without stopping the handler
    processSignal(sig)
    
    // Handler continues to listen for more signals
}
```

## Signal Queuing

### Configure Signal Queue Size

```go
config := signals.DefaultConfig()
config.QueueSize = 100  // Buffer up to 100 signals

handler := signals.NewHandler(config)
handler.Start(ctx)
```

If the queue is full, additional signals will be dropped. Increase `QueueSize` if you expect high signal frequency.

## Error Handling

### Custom Error Types

```go
import "github.com/fluxorio/fluxor/pkg/core/runtime/signals"

handler := signals.NewHandler(signals.DefaultConfig())
err := handler.Start(nil)

if err != nil {
    if sigErr, ok := err.(*signals.Error); ok {
        switch sigErr.Code {
        case signals.ErrCodeNilContext:
            log.Println("Context cannot be nil")
        case signals.ErrCodeAlreadyStarted:
            log.Println("Handler already started")
        case signals.ErrCodeAlreadyStopped:
            log.Println("Handler already stopped")
        case signals.ErrCodeShutdownTimeout:
            log.Printf("Shutdown timeout: %v", sigErr.Message)
        default:
            log.Printf("Signal error: %v", sigErr)
        }
    }
}
```

### Error Codes

- `ErrCodeAlreadyStarted`: Handler is already started
- `ErrCodeAlreadyStopped`: Handler is already stopped
- `ErrCodeNilContext`: Context cannot be nil
- `ErrCodeNilCallback`: Callback cannot be nil
- `ErrCodeChannelClosed`: Signal channel closed
- `ErrCodeHandlerStopped`: Handler stopped unexpectedly
- `ErrCodeShutdownTimeout`: Shutdown operation timed out

## Testing

### Testing Signal Handling

```go
import (
    "context"
    "os"
    "syscall"
    "testing"
    "github.com/fluxorio/fluxor/pkg/core/runtime/signals"
)

func TestSignalHandling(t *testing.T) {
    config := signals.DefaultConfig()
    handler := signals.NewHandler(config)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Start handler
    if err := handler.Start(ctx); err != nil {
        t.Fatalf("Start() error = %v", err)
    }
    defer handler.Stop()
    
    // Send signal (in real test, this would be done by the OS)
    // Note: This is a simplified example - in practice, you'd use
    // process signals or a test helper
    go func() {
        time.Sleep(10 * time.Millisecond)
        // Simulate signal by cancelling context
        cancel()
    }()
    
    // Wait for signal or context cancellation
    sig, err := handler.Wait(ctx)
    
    if err != nil {
        // Context was cancelled, which is expected in this test
        t.Logf("Context cancelled: %v", err)
    } else {
        t.Logf("Signal received: %v", sig)
    }
}
```

### Testing Callbacks

```go
func TestSignalCallback(t *testing.T) {
    called := make(chan os.Signal, 1)
    
    config := signals.Config{
        Signals: []os.Signal{os.Interrupt},
        OnSignal: func(sig os.Signal) {
            called <- sig
        },
    }
    
    handler := signals.NewHandler(config)
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    handler.Start(ctx)
    defer handler.Stop()
    
    // In a real scenario, you'd trigger the signal
    // For testing, you might need to use process signals
    // or mock the signal channel
}
```

## Integration Patterns

### With HTTP Server

```go
import (
    "context"
    "net/http"
    "github.com/fluxorio/fluxor/pkg/core/runtime/signals"
)

func main() {
    server := &http.Server{Addr: ":8080"}
    
    // Start server
    go func() {
        log.Println("Starting server on :8080")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()
    
    // Setup graceful shutdown
    ctx := context.Background()
    signals.GracefulShutdown(ctx, func() {
        log.Println("Shutting down server...")
        
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        
        if err := server.Shutdown(shutdownCtx); err != nil {
            log.Printf("Server shutdown error: %v", err)
        }
        
        log.Println("Server shutdown complete")
    })
}
```

### With Application State

```go
type App struct {
    handler signals.Handler
    state   *AppState
    mu      sync.Mutex
}

func (a *App) Start(ctx context.Context) error {
    config := signals.Config{
        OnSignal: func(sig os.Signal) {
            a.handleShutdown(sig)
        },
    }
    
    a.handler = signals.NewHandler(config)
    return a.handler.Start(ctx)
}

func (a *App) handleShutdown(sig os.Signal) {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    log.Printf("Shutdown signal received: %v", sig)
    
    // Save application state
    a.state.Save()
    
    // Close resources
    a.state.Close()
}
```

### With Multiple Components

```go
type ComponentManager struct {
    components []Shutdownable
    handler    signals.Handler
}

type Shutdownable interface {
    Shutdown(ctx context.Context) error
}

func (cm *ComponentManager) Start(ctx context.Context) error {
    config := signals.Config{
        OnSignal: func(sig os.Signal) {
            cm.shutdownAll(sig)
        },
        ShutdownTimeout: 30 * time.Second,
    }
    
    cm.handler = signals.NewHandler(config)
    return cm.handler.Start(ctx)
}

func (cm *ComponentManager) shutdownAll(sig os.Signal) {
    log.Printf("Shutting down %d components...", len(cm.components))
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    var wg sync.WaitGroup
    for _, comp := range cm.components {
        wg.Add(1)
        go func(c Shutdownable) {
            defer wg.Done()
            if err := c.Shutdown(ctx); err != nil {
                log.Printf("Component shutdown error: %v", err)
            }
        }(comp)
    }
    
    wg.Wait()
    log.Println("All components shut down")
}
```

## Configuration Options

### Full Configuration Example

```go
config := signals.Config{
    // Signals to listen for
    Signals: []os.Signal{
        os.Interrupt,
        syscall.SIGTERM,
        syscall.SIGINT,
        syscall.SIGUSR1,  // Custom signal
    },
    
    // Synchronous callback
    OnSignal: func(sig os.Signal) {
        log.Printf("Signal: %v", sig)
    },
    
    // Asynchronous callback
    OnSignalAsync: func(sig os.Signal) {
        go processSignalAsync(sig)
    },
    
    // Shutdown timeout
    ShutdownTimeout: 30 * time.Second,
    
    // Signal queue size
    QueueSize: 50,
    
    // Continue listening after signal
    ContinueOnSignal: false,
    
    // Track signal history
    SignalHistory: true,
}

handler := signals.NewHandler(config)
```

## Best Practices

1. **Always Use Context**: Pass a context to `Start()` and `Wait()` for cancellation support
2. **Cleanup on Stop**: Always call `Stop()` to clean up resources
3. **Handle Errors**: Check errors from `Start()` and `Wait()`
4. **Use Timeouts**: Configure `ShutdownTimeout` for cleanup operations
5. **Async Callbacks**: Use `OnSignalAsync` for non-blocking signal processing
6. **Signal Statistics**: Monitor `Stats()` in production for observability
7. **Signal History**: Enable history for debugging signal-related issues
8. **Queue Size**: Configure appropriate `QueueSize` based on signal frequency
9. **Graceful Shutdown**: Use `GracefulShutdown()` helper for common patterns
10. **Test with Context**: Use context cancellation in tests instead of actual signals

## Thread Safety

All methods in the `Handler` interface are thread-safe and can be called concurrently from multiple goroutines. The implementation uses mutexes and atomic operations to ensure safe concurrent access.

## Performance Considerations

- **Signal Processing**: Callbacks are executed synchronously by default; use `OnSignalAsync` for non-blocking processing
- **Queue Size**: Larger queue sizes use more memory but prevent signal loss
- **Signal History**: Enabling history tracking has minimal overhead but uses additional memory
- **Continue Mode**: `ContinueOnSignal` mode processes each signal sequentially

## Common Signals

### Unix/Linux Signals

- `os.Interrupt` (SIGINT): Interrupt signal (Ctrl+C)
- `syscall.SIGTERM`: Termination signal (default for kill)
- `syscall.SIGINT`: Interrupt signal (same as os.Interrupt)
- `syscall.SIGUSR1`: User-defined signal 1
- `syscall.SIGUSR2`: User-defined signal 2
- `syscall.SIGHUP`: Hangup signal

### Windows Signals

- `os.Interrupt`: Interrupt signal (Ctrl+C)
- `syscall.SIGTERM`: Termination signal

Note: On Windows, only `os.Interrupt` and `syscall.SIGTERM` are typically available.

## Interface Reference

### Handler Interface

```go
type Handler interface {
    // Start begins listening for signals
    Start(ctx context.Context) error
    
    // Stop stops listening for signals
    Stop() error
    
    // Wait waits for a shutdown signal
    Wait(ctx context.Context) (os.Signal, error)
    
    // Channel returns the signal channel
    Channel() <-chan os.Signal
    
    // Stats returns statistics about signals
    Stats() Stats
}
```

### Stats Structure

```go
type Stats struct {
    SignalCount    int64         // Total signals received
    LastSignal     os.Signal     // Last signal received
    LastSignalTime time.Time     // When last signal was received
    IsStarted      bool          // Handler is started
    IsStopped      bool          // Handler is stopped
}
```

### Config Structure

```go
type Config struct {
    Signals          []os.Signal
    OnSignal         func(os.Signal)
    OnSignalAsync    func(os.Signal)
    ShutdownTimeout  time.Duration
    QueueSize        int
    ContinueOnSignal bool
    SignalHistory    bool
}
```

## See Also

- [Go signal package documentation](https://pkg.go.dev/os/signal)
- [Context package documentation](https://pkg.go.dev/context)
- [Graceful shutdown patterns](../README.md#graceful-shutdown)
