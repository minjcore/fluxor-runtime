package rate

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Config configures rate tracking behavior.
type Config struct {
	// Window is the time window for calculating rates.
	// Must be greater than 0.
	// Defaults to 1 second.
	Window time.Duration

	// Granularity is the granularity for rate calculations (number of buckets).
	// Higher values provide more accurate rates but use more memory.
	// Must be greater than 0. Defaults to 10.
	Granularity int

	// OnRateCalculated is called when rate is calculated.
	OnRateCalculated func(ctx context.Context, rate float64, window time.Duration)

	// OnRateCalculatedAsync is called asynchronously when rate is calculated.
	OnRateCalculatedAsync func(ctx context.Context, rate float64, window time.Duration)

	// EnableMetrics enables detailed metrics collection.
	EnableMetrics bool
}

// DefaultConfig returns the default rate tracking configuration.
func DefaultConfig() Config {
	return Config{
		Window:      time.Second,
		Granularity: 10,
	}
}

// Manager provides rate calculation and tracking functionality.
type Manager interface {
	// Record records an event for rate calculation.
	Record(ctx context.Context)

	// RecordN records N events for rate calculation.
	RecordN(ctx context.Context, n int64)

	// Rate returns the current rate (events per window).
	Rate() float64

	// RateWithWindow returns the current rate for the specified window.
	RateWithWindow(window time.Duration) float64

	// Stats returns statistics about rate tracking.
	Stats() Stats
}

// Stats contains statistics about rate tracking.
type Stats struct {
	// TotalEvents is the total number of events recorded.
	TotalEvents int64

	// CurrentRate is the current rate (events per window).
	CurrentRate float64

	// Window is the current rate calculation window.
	Window time.Duration

	// Granularity is the number of buckets used for rate calculation.
	Granularity int

	// LastEventTime is when the last event was recorded.
	LastEventTime time.Time

	// FirstEventTime is when the first event was recorded.
	FirstEventTime time.Time
}

// rateManager implements the Manager interface using a sliding window algorithm.
type rateManager struct {
	config Config

	// Sliding window buckets
	buckets     []int64       // Event counts per bucket
	currentIdx  int           // Current bucket index
	bucketSize  time.Duration // Size of each bucket
	lastUpdate  time.Time     // Last time buckets were updated
	bucketsMu   sync.RWMutex  // Protects buckets and currentIdx

	// Statistics
	totalEvents int64
	firstEvent  time.Time
	lastEvent   time.Time
	statsMu     sync.RWMutex
}

// NewManager creates a new rate manager with default config.
func NewManager() Manager {
	return NewManagerWithConfig(DefaultConfig())
}

// NewManagerWithConfig creates a new rate manager with the specified config.
func NewManagerWithConfig(config Config) Manager {
	if config.Window <= 0 {
		config.Window = DefaultConfig().Window
	}
	if config.Granularity <= 0 {
		config.Granularity = DefaultConfig().Granularity
	}

	bucketSize := config.Window / time.Duration(config.Granularity)
	if bucketSize <= 0 {
		bucketSize = time.Millisecond
	}

	return &rateManager{
		config:     config,
		buckets:    make([]int64, config.Granularity),
		bucketSize: bucketSize,
		lastUpdate: time.Now(),
	}
}

// Record records an event for rate calculation.
func (m *rateManager) Record(ctx context.Context) {
	m.RecordN(ctx, 1)
}

// RecordN records N events for rate calculation.
func (m *rateManager) RecordN(ctx context.Context, n int64) {
	if ctx == nil {
		return // Silently ignore if context is nil (optional validation)
	}
	if n <= 0 {
		return
	}

	now := time.Now()

	atomic.AddInt64(&m.totalEvents, n)

	m.statsMu.Lock()
	if m.firstEvent.IsZero() {
		m.firstEvent = now
	}
	m.lastEvent = now
	m.statsMu.Unlock()

	m.bucketsMu.Lock()
	defer m.bucketsMu.Unlock()

	// Update buckets based on elapsed time
	m.updateBuckets(now)

	// Record event in current bucket
	m.buckets[m.currentIdx] += n
}

// updateBuckets updates buckets based on elapsed time.
func (m *rateManager) updateBuckets(now time.Time) {
	elapsed := now.Sub(m.lastUpdate)
	if elapsed < m.bucketSize {
		return // No buckets to advance
	}

	bucketsToAdvance := int(elapsed / m.bucketSize)
	if bucketsToAdvance >= m.config.Granularity {
		// More than a full window elapsed - clear all buckets
		for i := range m.buckets {
			m.buckets[i] = 0
		}
		m.currentIdx = 0
	} else {
		// Advance buckets and clear expired ones
		for i := 0; i < bucketsToAdvance; i++ {
			m.currentIdx = (m.currentIdx + 1) % m.config.Granularity
			m.buckets[m.currentIdx] = 0
		}
	}

	m.lastUpdate = now
}

// Rate returns the current rate (events per window).
func (m *rateManager) Rate() float64 {
	return m.RateWithWindow(m.config.Window)
}

// RateWithWindow returns the current rate for the specified window.
func (m *rateManager) RateWithWindow(window time.Duration) float64 {
	m.bucketsMu.Lock()
	defer m.bucketsMu.Unlock()

	now := time.Now()
	m.updateBuckets(now)

	// Calculate rate based on available window
	if window <= 0 {
		window = m.config.Window
	}

	bucketsInWindow := int(window / m.bucketSize)
	if bucketsInWindow > m.config.Granularity {
		bucketsInWindow = m.config.Granularity
	}
	if bucketsInWindow <= 0 {
		bucketsInWindow = 1
	}

	// Sum events in the relevant buckets
	var total int64
	if bucketsInWindow >= m.config.Granularity {
		// Use all buckets
		for _, count := range m.buckets {
			total += count
		}
	} else {
		// Use recent buckets (wrapping around if needed)
		for i := 0; i < bucketsInWindow; i++ {
			idx := (m.currentIdx - bucketsInWindow + 1 + i + m.config.Granularity) % m.config.Granularity
			total += m.buckets[idx]
		}
	}

	// Calculate rate per window
	actualWindow := time.Duration(bucketsInWindow) * m.bucketSize
	if actualWindow <= 0 {
		actualWindow = m.config.Window
	}

	rate := float64(total) / actualWindow.Seconds()

	// Invoke callback if configured
	if m.config.OnRateCalculated != nil {
		m.config.OnRateCalculated(context.Background(), rate, window)
	}

	if m.config.OnRateCalculatedAsync != nil {
		go m.config.OnRateCalculatedAsync(context.Background(), rate, window)
	}

	return rate
}

// Stats returns statistics about rate tracking.
func (m *rateManager) Stats() Stats {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	currentRate := m.Rate()

	return Stats{
		TotalEvents:    atomic.LoadInt64(&m.totalEvents),
		CurrentRate:    currentRate,
		Window:         m.config.Window,
		Granularity:    m.config.Granularity,
		LastEventTime:  m.lastEvent,
		FirstEventTime: m.firstEvent,
	}
}
