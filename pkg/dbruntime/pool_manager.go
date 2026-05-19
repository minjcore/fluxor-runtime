package dbruntime

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/runtime/state"
)

// PoolManager manages the database connection pool in a dedicated goroutine
// This separates pool lifecycle management from connection operations
type PoolManager struct {
	pool     *Pool
	config   PoolConfig
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
	stateManager state.Manager
	eventBus core.EventBus

	// Health check configuration
	healthCheckInterval time.Duration
	lastHealthCheck      time.Time
	healthCheckMu        sync.RWMutex

	// Reconnect: only one reconnect at a time
	reconnectMu sync.Mutex
}

// NewPoolManager creates a new pool manager
func NewPoolManager(config PoolConfig, eventBus core.EventBus) *PoolManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create state manager with callbacks for event publishing
	stateConfig := state.DefaultConfig()
	stateConfig.HistorySize = 20 // Keep last 20 state transitions
	stateConfig.OnStateChange = func(from, to state.State) {
		if eventBus != nil {
			_ = eventBus.Publish("pool.manager.state.changed", map[string]interface{}{
				"from": from.String(),
				"to":   to.String(),
			})
		}
	}
	
	pm := &PoolManager{
		config:              config,
		ctx:                  ctx,
		cancel:               cancel,
		healthCheckInterval: 30 * time.Second, // Default health check interval
		eventBus:            eventBus,
		stateManager:        state.NewManager(stateConfig),
	}
	
	return pm
}

// Start initializes the pool and starts the manager goroutine
func (pm *PoolManager) Start() error {
	// Use state manager for lifecycle management
	return pm.stateManager.Start(pm.ctx, func() error {
		// Create the pool
		pool, err := NewPool(pm.config)
		if err != nil {
			return fmt.Errorf("failed to create pool: %w", err)
		}

		pm.mu.Lock()
		pm.pool = pool
		pm.mu.Unlock()

		// Start the manager goroutine
		pm.wg.Add(1)
		go pm.run()

		// Publish event
		if pm.eventBus != nil {
			_ = pm.eventBus.Publish("pool.manager.started", map[string]interface{}{
				"max_open_conns": pm.config.MaxOpenConns,
				"max_idle_conns": pm.config.MaxIdleConns,
			})
		}

		return nil
	})
}

// Stop stops the pool manager and closes the pool
func (pm *PoolManager) Stop() error {
	// Use state manager for lifecycle management
	return pm.stateManager.Stop(func() error {
		// Signal shutdown
		pm.cancel()

		// Wait for goroutine to finish
		pm.wg.Wait()

		// Close the pool
		pm.mu.Lock()
		pool := pm.pool
		pm.pool = nil
		pm.mu.Unlock()

		if pool != nil {
			if err := pool.Close(); err != nil {
				return err
			}
		}

		// Publish event
		if pm.eventBus != nil {
			_ = pm.eventBus.Publish("pool.manager.stopped", nil)
		}

		return nil
	})
}

// run is the main loop for the pool manager goroutine
// It handles health checks, connection monitoring, and pool maintenance
func (pm *PoolManager) run() {
	defer pm.wg.Done()

	ticker := time.NewTicker(pm.healthCheckInterval)
	defer ticker.Stop()

	log.Printf("[PoolManager] Started pool management goroutine")

	for {
		select {
		case <-pm.ctx.Done():
			log.Printf("[PoolManager] Shutting down pool management goroutine")
			return
		case <-ticker.C:
			pm.performHealthCheck()
			pm.monitorPool()
		}
	}
}

// performHealthCheck performs a health check on the pool; on failure attempts reconnect
func (pm *PoolManager) performHealthCheck() {
	pm.mu.RLock()
	pool := pm.pool
	pm.mu.RUnlock()

	if pool == nil {
		// No pool yet (e.g. SkipInitialPing): try to create one
		pm.tryReconnect()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		log.Printf("[PoolManager] Health check failed: %v", err)
		if pm.eventBus != nil {
			_ = pm.eventBus.Publish("pool.health.check.failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
		pm.tryReconnect()
	} else {
		pm.healthCheckMu.Lock()
		pm.lastHealthCheck = time.Now()
		pm.healthCheckMu.Unlock()
	}
}

// tryReconnect closes the current pool (if any) and creates a new one; safe to call when pool is nil (e.g. lazy start)
func (pm *PoolManager) tryReconnect() {
	pm.reconnectMu.Lock()
	defer pm.reconnectMu.Unlock()

	// Create new pool (NewPool pings unless SkipInitialPing)
	cfg := pm.config
	cfg.SkipInitialPing = false // always verify when reconnecting
	newPool, err := NewPool(cfg)
	if err != nil {
		log.Printf("[PoolManager] Reconnect failed: %v", err)
		return
	}

	pm.mu.Lock()
	oldPool := pm.pool
	pm.pool = newPool
	pm.mu.Unlock()

	if oldPool != nil {
		if err := oldPool.Close(); err != nil {
			log.Printf("[PoolManager] Close old pool: %v", err)
		}
	}

	pm.healthCheckMu.Lock()
	pm.lastHealthCheck = time.Now()
	pm.healthCheckMu.Unlock()

	log.Printf("[PoolManager] Reconnected to database")
	if pm.eventBus != nil {
		_ = pm.eventBus.Publish("pool.reconnected", nil)
	}
}

// monitorPool monitors pool statistics and publishes metrics
func (pm *PoolManager) monitorPool() {
	pm.mu.RLock()
	pool := pm.pool
	pm.mu.RUnlock()

	if pool == nil {
		return
	}

	stats := pool.Stats()
	if pm.eventBus != nil {
		_ = pm.eventBus.Publish("pool.stats", map[string]interface{}{
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
			"wait_count":       stats.WaitCount,
			"wait_duration":    stats.WaitDuration.String(),
		})
	}
}

// GetPool returns the managed pool (thread-safe)
func (pm *PoolManager) GetPool() *Pool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.pool
}

// IsStarted returns whether the manager is started
func (pm *PoolManager) IsStarted() bool {
	currentState := pm.stateManager.Current()
	return currentState == state.StateRunning || currentState == state.StateStarting
}

// GetState returns the current state of the pool manager
func (pm *PoolManager) GetState() state.State {
	return pm.stateManager.Current()
}

// GetStateStats returns state transition statistics
func (pm *PoolManager) GetStateStats() state.Stats {
	return pm.stateManager.Stats()
}

// GetStateHistory returns the state transition history
func (pm *PoolManager) GetStateHistory() []state.Transition {
	return pm.stateManager.History()
}

// WaitForState waits for the pool manager to reach a specific state
func (pm *PoolManager) WaitForState(ctx context.Context, target state.State) error {
	return pm.stateManager.WaitForState(ctx, target)
}

// OnStateChange registers a callback for state changes
func (pm *PoolManager) OnStateChange(callback func(from, to state.State)) {
	pm.stateManager.OnStateChange(callback)
}

// GetLastHealthCheck returns the time of the last successful health check
func (pm *PoolManager) GetLastHealthCheck() time.Time {
	pm.healthCheckMu.RLock()
	defer pm.healthCheckMu.RUnlock()
	return pm.lastHealthCheck
}

// SetHealthCheckInterval sets the health check interval
func (pm *PoolManager) SetHealthCheckInterval(interval time.Duration) {
	pm.healthCheckInterval = interval
}
