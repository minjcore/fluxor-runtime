// Package drain provides graceful draining functionality for runtime components.
//
// Draining allows components to stop accepting new work while allowing
// in-flight work to complete gracefully. This is essential for zero-downtime
// deployments and graceful shutdowns.
//
// Example usage:
//
//	// Create a drainer
//	drainer := drain.NewDrainer(drain.DefaultConfig())
//
//	// Register components that need to be drained
//	drainer.Register("eventbus", eventBus)
//	drainer.Register("executor", executor)
//	drainer.Register("server", httpServer)
//
//	// Drain all components during shutdown
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	if err := drainer.DrainAll(ctx); err != nil {
//		log.Printf("Drain completed with errors: %v", err)
//	}
//
// Components that implement the Drainable interface can be registered
// and drained. The drainer supports both sequential and parallel draining,
// with configurable timeouts and callbacks.
//
// Path: pkg/core/runtime/drain
package drain
