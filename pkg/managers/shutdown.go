package managers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ShutdownHook represents a function to be called during shutdown
type ShutdownHook func(ctx context.Context) error

// ShutdownPhase represents the phase of shutdown
type ShutdownPhase int

const (
	// ShutdownPhasePreStop is called before stopping services
	ShutdownPhasePreStop ShutdownPhase = iota
	// ShutdownPhaseStop is the main stop phase
	ShutdownPhaseStop
	// ShutdownPhasePostStop is called after services are stopped
	ShutdownPhasePostStop
)

// ShutdownCoordinator manages graceful shutdown of all services
type ShutdownCoordinator struct {
	hooks map[ShutdownPhase][]ShutdownHook
	mu    sync.RWMutex
}

// newShutdownCoordinator creates a new shutdown coordinator
func newShutdownCoordinator() *ShutdownCoordinator {
	return &ShutdownCoordinator{
		hooks: make(map[ShutdownPhase][]ShutdownHook),
	}
}

// RegisterHook registers a shutdown hook for a specific phase
func (sc *ShutdownCoordinator) RegisterHook(phase ShutdownPhase, hook ShutdownHook) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.hooks[phase] = append(sc.hooks[phase], hook)
}

// Shutdown executes all registered shutdown hooks in phases
func (sc *ShutdownCoordinator) Shutdown(ctx context.Context) error {
	phases := []ShutdownPhase{
		ShutdownPhasePreStop,
		ShutdownPhaseStop,
		ShutdownPhasePostStop,
	}

	var collectedErrors []error
	for _, phase := range phases {
		if err := sc.executePhase(ctx, phase); err != nil {
			collectedErrors = append(collectedErrors, fmt.Errorf("phase %d: %w", phase, err))
			// Continue to next phase even if current phase has errors
		}
	}

	if len(collectedErrors) > 0 {
		return fmt.Errorf("shutdown errors: %v", collectedErrors)
	}

	return nil
}

// executePhase executes all hooks for a specific phase in parallel
func (sc *ShutdownCoordinator) executePhase(ctx context.Context, phase ShutdownPhase) error {
	sc.mu.RLock()
	hooks := sc.hooks[phase]
	sc.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	// Execute hooks in parallel with WaitGroup
	var wg sync.WaitGroup
	errChan := make(chan error, len(hooks))

	for i, hook := range hooks {
		wg.Add(1)
		go func(idx int, h ShutdownHook) {
			defer wg.Done()
			if err := h(ctx); err != nil {
				errChan <- fmt.Errorf("hook %d: %w", idx, err)
			}
		}(i, hook)
	}

	// Wait for all hooks to complete or context cancellation
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		close(errChan)
	case <-ctx.Done():
		// Context cancelled, collect whatever errors we have
		close(errChan)
		return fmt.Errorf("phase cancelled: %w", ctx.Err())
	}

	// Collect all errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("phase errors: %v", errors)
	}

	return nil
}

// Managers methods for shutdown coordination

// RegisterShutdownHook registers a shutdown hook with Managers
func (m *Managers) RegisterShutdownHook(phase ShutdownPhase, hook ShutdownHook) {
	if m.shutdownCoordinator == nil {
		m.mu.Lock()
		if m.shutdownCoordinator == nil {
			m.shutdownCoordinator = newShutdownCoordinator()
		}
		m.mu.Unlock()
	}
	m.shutdownCoordinator.RegisterHook(phase, hook)
}

// Shutdown performs graceful shutdown of all Managers-managed components
func (m *Managers) Shutdown(ctx context.Context) error {
	startTime := time.Now()

	// Set timeout if not already set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	if m.logger != nil {
		m.logger.Info("Starting Managers shutdown")
	}

	// Stop heartbeat first
	m.StopHeartbeat()

	// Execute shutdown hooks if registered
	if m.shutdownCoordinator != nil {
		if err := m.shutdownCoordinator.Shutdown(ctx); err != nil {
			if m.logger != nil {
				m.logger.Error("Shutdown hooks failed", "error", err)
			}
			// Continue with remaining shutdown steps
		}
	}

	// Stop HTTP server
	if m.httpServer != nil {
		if err := m.httpServer.Stop(); err != nil {
			if m.logger != nil {
				m.logger.Error("HTTP server stop failed", "error", err)
			}
			// Continue with remaining shutdown
		}
	}

	// Close cache
	if m.cache != nil {
		if err := m.cache.Clear(ctx); err != nil {
			if m.logger != nil {
				m.logger.Error("Cache clear failed", "error", err)
			}
		}
	}

	duration := time.Since(startTime)
	if m.logger != nil {
		m.logger.Info("Managers shutdown complete", "duration", duration)
	}

	return nil
}

// DefaultShutdownHooks registers default shutdown hooks for common patterns
func (m *Managers) DefaultShutdownHooks() {
	// PreStop: Mark services as stopping
	m.RegisterShutdownHook(ShutdownPhasePreStop, func(ctx context.Context) error {
		if m.serviceRegistry != nil {
			for _, service := range m.serviceRegistry.List() {
				m.serviceRegistry.UpdateStatus(service.Name, ServiceStatusStopping)
			}
		}
		if m.logger != nil {
			m.logger.Info("PreStop: Marking services as stopping")
		}
		return nil
	})

	// Stop: Actual service shutdown
	m.RegisterShutdownHook(ShutdownPhaseStop, func(ctx context.Context) error {
		if m.logger != nil {
			m.logger.Info("Stop: Shutting down services")
		}
		// Services are stopped by their own Stop() methods
		return nil
	})

	// PostStop: Cleanup and finalization
	m.RegisterShutdownHook(ShutdownPhasePostStop, func(ctx context.Context) error {
		if m.serviceRegistry != nil {
			for _, service := range m.serviceRegistry.List() {
				m.serviceRegistry.UpdateStatus(service.Name, ServiceStatusStopped)
			}
		}
		if m.logger != nil {
			m.logger.Info("PostStop: Services marked as stopped")
		}
		return nil
	})
}
