// Package signals provides signal handling for graceful shutdown.
//
// The signals package provides utilities for handling OS signals (SIGINT, SIGTERM, etc.)
// to enable graceful shutdown of applications. It supports both simple signal waiting
// and advanced signal handling with callbacks, timeouts, and signal history.
//
// Basic Usage:
//
//	// Simple signal wait
//	ctx := context.Background()
//	sig, err := signals.WaitForSignal(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Received signal: %v", sig)
//
// Advanced Usage with Callback:
//
//	// Create handler with callback
//	config := signals.Config{
//	    Signals: []os.Signal{os.Interrupt, syscall.SIGTERM},
//	    OnSignal: func(sig os.Signal) {
//	        log.Printf("Received signal: %v", sig)
//	        // Perform cleanup
//	    },
//	}
//
//	handler := signals.NewHandler(config)
//	ctx := context.Background()
//	if err := handler.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Wait for signal
//	sig, err := handler.Wait(ctx)
//
// Async Callback Usage:
//
//	// Callback executed asynchronously in a separate goroutine
//	config := signals.Config{
//	    OnSignalAsync: func(sig os.Signal) {
//	        // Non-blocking processing
//	        log.Printf("Async signal received: %v", sig)
//	    },
//	}
//
//	handler := signals.NewHandler(config)
//	handler.Start(ctx)
//
// Timeout Usage:
//
//	// Wait for signal with timeout
//	ctx := context.Background()
//	sig, err := signals.WaitForSignalWithTimeout(ctx, 30*time.Second)
//	if err != nil {
//	    if err == context.DeadlineExceeded {
//	        log.Println("Timeout waiting for signal")
//	    }
//	}
//
// Graceful Shutdown with Options:
//
//	// Setup graceful shutdown with timeout
//	ctx := context.Background()
//	err := signals.GracefulShutdown(ctx, func() {
//	    log.Println("Shutting down gracefully...")
//	    // Perform cleanup
//	}, signals.WithShutdownTimeout(30*time.Second))
//
// Signal Statistics:
//
//	// Get signal statistics
//	handler := signals.NewHandler(signals.DefaultConfig())
//	handler.Start(ctx)
//	defer handler.Stop()
//
//	stats := handler.Stats()
//	log.Printf("Signals received: %d", stats.SignalCount)
//	log.Printf("Last signal: %v", stats.LastSignal)
//	log.Printf("Last signal time: %v", stats.LastSignalTime)
//
// Signal History:
//
//	// Enable signal history tracking
//	config := signals.DefaultConfig()
//	config.SignalHistory = true
//	handler := signals.NewHandler(config)
//
//	// Get signal history
//	history := handler.(*signalHandler).GetHistory()
//
// Continue On Signal:
//
//	// Continue listening after receiving a signal
//	config := signals.DefaultConfig()
//	config.ContinueOnSignal = true
//	handler := signals.NewHandler(config)
//
// The Handler interface provides:
//   - Start() - Begin listening for signals
//   - Stop() - Stop listening for signals (with optional timeout)
//   - Wait() - Wait for a signal to be received
//   - Channel() - Get the signal channel directly
//   - Stats() - Get statistics about signals received
//
// All methods are thread-safe and can be used concurrently.
package signals
