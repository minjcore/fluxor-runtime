package concurrency

import (
	"sync"
	"time"
)

// ExecutorMetrics represents metrics for an executor
type ExecutorMetrics struct {
	ID                string  `json:"id"`
	QueuedTasks       int64   `json:"queuedTasks"`
	QueueUtilization  float64 `json:"queueUtilization"`
	Throughput        float64 `json:"throughput"` // Tasks per second (simplified - based on completed tasks)
	WorkerUtilization float64 `json:"workerUtilization"`
	CompletedTasks    int64   `json:"completedTasks"`
	RejectedTasks     int64   `json:"rejectedTasks"`
}

// WorkerPoolMetrics represents metrics for a worker pool
type WorkerPoolMetrics struct {
	ID                string  `json:"id"`
	QueuedTasks       int64   `json:"queuedTasks"`
	QueueUtilization  float64 `json:"queueUtilization"`
	Throughput        float64 `json:"throughput"`
	WorkerUtilization float64 `json:"workerUtilization"`
	CompletedTasks    int64   `json:"completedTasks"`
	RejectedTasks     int64   `json:"rejectedTasks"`
}

// DashboardMetrics represents all dashboard metrics
type DashboardMetrics struct {
	Executors   []ExecutorMetrics   `json:"executors"`
	WorkerPools []WorkerPoolMetrics `json:"workerPools"`
	Timestamp   time.Time           `json:"timestamp"`
}

// executorEntry represents a registered executor
type executorEntry struct {
	id                 string
	executor           Executor
	lastCompletedTasks int64
	lastTimestamp      time.Time
	lastThroughput     float64
}

// workerPoolEntry represents a registered worker pool
type workerPoolEntry struct {
	id                 string
	pool               WorkerPool
	lastCompletedTasks int64
	lastTimestamp      time.Time
	lastThroughput     float64
}

// dashboardRegistry is a global registry for tracking executors and worker pools
type dashboardRegistry struct {
	mu          sync.RWMutex
	executors   map[string]*executorEntry
	workerPools map[string]*workerPoolEntry
	startTime   time.Time // Track start time for throughput calculation
}

var (
	registry = &dashboardRegistry{
		executors:   make(map[string]*executorEntry),
		workerPools: make(map[string]*workerPoolEntry),
		startTime:   time.Now(),
	}
)

// RegisterExecutor registers an executor with the dashboard registry
func RegisterExecutor(id string, executor Executor) {
	if id == "" || executor == nil {
		return
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.executors[id] = &executorEntry{
		id:                 id,
		executor:           executor,
		lastCompletedTasks: 0,
		lastTimestamp:      time.Now(),
	}
}

// UnregisterExecutor removes an executor from the dashboard registry
func UnregisterExecutor(id string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	delete(registry.executors, id)
}

// RegisterWorkerPool registers a worker pool with the dashboard registry
func RegisterWorkerPool(id string, pool WorkerPool) {
	if id == "" || pool == nil {
		return
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.workerPools[id] = &workerPoolEntry{
		id:                 id,
		pool:               pool,
		lastCompletedTasks: 0,
		lastTimestamp:      time.Now(),
	}
}

// UnregisterWorkerPool removes a worker pool from the dashboard registry
func UnregisterWorkerPool(id string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	delete(registry.workerPools, id)
}

// GetDashboardMetrics collects metrics from all registered executors and worker pools
func GetDashboardMetrics() DashboardMetrics {
	registry.mu.Lock() // Use write lock since we're updating lastCompletedTasks and lastTimestamp
	defer registry.mu.Unlock()

	executorMetrics := make([]ExecutorMetrics, 0, len(registry.executors))
	workerPoolMetrics := make([]WorkerPoolMetrics, 0, len(registry.workerPools))

	// Collect executor metrics
	for _, entry := range registry.executors {
		stats := entry.executor.Stats()

		// Calculate throughput based on delta since last measurement
		now := time.Now()
		deltaTime := now.Sub(entry.lastTimestamp).Seconds()
		if deltaTime < 0.1 {
			deltaTime = 0.1 // Avoid division by very small numbers
		}

		deltaTasks := stats.CompletedTasks - entry.lastCompletedTasks
		instantThroughput := float64(deltaTasks) / deltaTime

		// Apply exponential smoothing: 70% new value, 30% old value
		// This prevents sharp drops to zero when no tasks complete
		var throughput float64
		if entry.lastThroughput == 0 {
			throughput = instantThroughput
		} else {
			throughput = 0.7*instantThroughput + 0.3*entry.lastThroughput
		}

		// Update last values for next calculation
		entry.lastCompletedTasks = stats.CompletedTasks
		entry.lastTimestamp = now
		entry.lastThroughput = throughput

		// Worker utilization: For executor, we assume all workers are active if queue is not empty
		// This is a simplification - real implementation would track active workers
		workerUtilization := 0.0
		if stats.QueuedTasks > 0 {
			workerUtilization = 100.0 // Simplified: assume workers are busy if queue has tasks
		} else if stats.QueueCapacity > 0 {
			workerUtilization = float64(stats.ActiveWorkers) / float64(stats.QueueCapacity) * 100.0
		}

		executorMetrics = append(executorMetrics, ExecutorMetrics{
			ID:                entry.id,
			QueuedTasks:       stats.QueuedTasks,
			QueueUtilization:  stats.QueueUtilization,
			Throughput:        throughput,
			WorkerUtilization: workerUtilization,
			CompletedTasks:    stats.CompletedTasks,
			RejectedTasks:     stats.RejectedTasks,
		})
	}

	// Collect worker pool metrics
	for _, entry := range registry.workerPools {
		stats := entry.pool.Stats()

		// Calculate throughput based on delta since last measurement
		now := time.Now()
		deltaTime := now.Sub(entry.lastTimestamp).Seconds()
		if deltaTime < 0.1 {
			deltaTime = 0.1 // Avoid division by very small numbers
		}

		deltaTasks := stats.CompletedTasks - entry.lastCompletedTasks
		instantThroughput := float64(deltaTasks) / deltaTime

		// Apply exponential smoothing: 70% new value, 30% old value
		// This prevents sharp drops to zero when no tasks complete
		var throughput float64
		if entry.lastThroughput == 0 {
			throughput = instantThroughput
		} else {
			throughput = 0.7*instantThroughput + 0.3*entry.lastThroughput
		}

		// Update last values for next calculation
		entry.lastCompletedTasks = stats.CompletedTasks
		entry.lastTimestamp = now
		entry.lastThroughput = throughput

		// Worker utilization: similar calculation for worker pools
		workerUtilization := 0.0
		if stats.QueuedTasks > 0 {
			workerUtilization = 100.0 // Workers are busy if queue has tasks
		} else if stats.CompletedTasks > 0 {
			workerUtilization = 50.0 // Workers are partially utilized
		}

		workerPoolMetrics = append(workerPoolMetrics, WorkerPoolMetrics{
			ID:                entry.id,
			QueuedTasks:       stats.QueuedTasks,
			QueueUtilization:  stats.QueueUtilization,
			Throughput:        throughput,
			WorkerUtilization: workerUtilization,
			CompletedTasks:    stats.CompletedTasks,
			RejectedTasks:     stats.RejectedTasks,
		})
	}

	return DashboardMetrics{
		Executors:   executorMetrics,
		WorkerPools: workerPoolMetrics,
		Timestamp:   time.Now(),
	}
}
