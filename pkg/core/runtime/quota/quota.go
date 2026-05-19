package quota

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// QuotaType represents the type of quota being tracked.
type QuotaType string

const (
	// QuotaTypeRequests tracks request-based quotas.
	QuotaTypeRequests QuotaType = "requests"

	// QuotaTypeMemory tracks memory-based quotas.
	QuotaTypeMemory QuotaType = "memory"

	// QuotaTypeCPU tracks CPU-based quotas.
	QuotaTypeCPU QuotaType = "cpu"

	// QuotaTypeCustom tracks custom quotas.
	QuotaTypeCustom QuotaType = "custom"
)

// String returns the string representation of the quota type.
func (qt QuotaType) String() string {
	return string(qt)
}

// Manager provides quota management and enforcement.
type Manager interface {
	// Register registers a quota with the given name, type, and limit.
	// Options can be provided to configure the quota.
	Register(name string, quotaType QuotaType, limit int64, window time.Duration, opts ...Option) error

	// Unregister removes a quota.
	Unregister(name string) error

	// Acquire attempts to acquire quota for the given name.
	// Returns true if quota is available, false if exceeded.
	// Also returns the current usage and remaining quota.
	Acquire(ctx context.Context, name string, amount int64) (bool, int64, int64, error)

	// Release releases quota for the given name.
	Release(name string, amount int64) error

	// GetUsage returns the current usage for a quota.
	GetUsage(name string) (int64, error)

	// GetRemaining returns the remaining quota for a quota.
	GetRemaining(name string) (int64, error)

	// Reset resets the usage for a quota.
	Reset(name string) error

	// ResetAll resets all quotas.
	ResetAll() error

	// Clear removes all quotas.
	Clear()

	// Stats returns statistics about quota usage.
	Stats() Stats

	// GetQuota returns information about a registered quota.
	GetQuota(name string) (*QuotaInfo, error)

	// QuotaStats returns statistics for a specific quota.
	QuotaStats(name string) (*QuotaStat, error)

	// ListQuotas returns a list of all registered quotas, sorted by name.
	ListQuotas() []QuotaInfo
}

// Config configures quota behavior.
type Config struct {
	// OnQuotaExceeded is called when a quota is exceeded.
	OnQuotaExceeded func(name string, quotaType QuotaType, limit, usage int64)

	// OnQuotaExceededAsync is called asynchronously when a quota is exceeded.
	OnQuotaExceededAsync func(name string, quotaType QuotaType, limit, usage int64)

	// OnQuotaReset is called when a quota is reset.
	OnQuotaReset func(name string)

	// OnQuotaResetAsync is called asynchronously when a quota is reset.
	OnQuotaResetAsync func(name string)

	// HistorySize is the maximum number of quota events to keep in history.
	// Zero means no history. Defaults to 0.
	HistorySize int

	// AutoResetInterval is the interval at which quotas are automatically reset.
	// Zero means no auto-reset. Defaults to 0.
	AutoResetInterval time.Duration

	// EnableMetrics enables detailed metrics collection.
	EnableMetrics bool
}

// DefaultConfig returns the default quota configuration.
func DefaultConfig() Config {
	return Config{
		HistorySize:       0,
		AutoResetInterval: 0,
		EnableMetrics:     false,
	}
}

// Quota represents a quota with its configuration and current state.
type Quota struct {
	Name       string
	Type       QuotaType
	Limit      int64
	Window     time.Duration
	Usage      int64 // atomic
	LastReset  time.Time
	LastUpdate time.Time
	RegisteredAt time.Time
	mu         sync.RWMutex
	
	// Per-quota statistics (only tracked when EnableMetrics is true)
	TotalAcquired int64 // atomic
	TotalReleased int64 // atomic
	TotalExceeded int64 // atomic
}

// QuotaInfo contains metadata about a registered quota.
type QuotaInfo struct {
	// Name is the unique identifier for the quota.
	Name string

	// Type is the type of quota.
	Type QuotaType

	// Limit is the quota limit. Zero means unlimited.
	Limit int64

	// Window is the time window for automatic resets.
	Window time.Duration

	// Usage is the current usage.
	Usage int64

	// Remaining is the remaining quota. -1 means unlimited.
	Remaining int64

	// RegisteredAt is when the quota was registered.
	RegisteredAt time.Time

	// LastReset is when the quota was last reset.
	LastReset time.Time

	// LastUpdate is when the quota was last updated.
	LastUpdate time.Time
}

// Option configures quota registration.
type Option func(*Quota)

// QuotaStat contains statistics for a specific quota.
type QuotaStat struct {
	Name        string
	Type        QuotaType
	Limit       int64
	Usage       int64
	Remaining   int64
	Window      time.Duration
	LastReset   time.Time
	LastUpdate  time.Time
	Exceeded    int64
	TotalAcquired int64
	TotalReleased int64
}

// Event represents a quota event (exceeded, reset, etc.).
type Event struct {
	Name      string
	Type      QuotaType
	EventType string
	Limit     int64
	Usage     int64
	Timestamp time.Time
}

// Stats contains statistics about quota operations.
type Stats struct {
	// TotalQuotas is the total number of registered quotas.
	TotalQuotas int64

	// TotalAcquired is the total number of successful quota acquisitions.
	TotalAcquired int64

	// TotalExceeded is the total number of quota exceeded events.
	TotalExceeded int64

	// TotalReleased is the total number of quota releases.
	TotalReleased int64

	// TotalReset is the total number of quota resets.
	TotalReset int64

	// LastEventTime is when the last quota event occurred.
	LastEventTime time.Time
}

// quotaManager implements the Manager interface.
type quotaManager struct {
	config      Config
	quotas      map[string]*Quota
	mu          sync.RWMutex
	stats       Stats
	history     []Event
	historyMu   sync.RWMutex
	stopChan    chan struct{}
	doneChan    chan struct{}
	running     int32 // atomic: 1 if auto-reset is running, 0 otherwise
}

// NewManager creates a new quota manager with the given configuration.
func NewManager(config Config) Manager {
	if config.HistorySize < 0 {
		config.HistorySize = 0
	}

	manager := &quotaManager{
		config:    config,
		quotas:    make(map[string]*Quota),
		history:   make([]Event, 0, config.HistorySize),
		stopChan:  make(chan struct{}),
		doneChan:  make(chan struct{}),
	}

	// Start auto-reset if configured
	if config.AutoResetInterval > 0 {
		atomic.StoreInt32(&manager.running, 1)
		go manager.autoResetLoop()
	} else {
		close(manager.doneChan)
	}

	return manager
}

// Register registers a quota with the given name, type, and limit.
func (m *quotaManager) Register(name string, quotaType QuotaType, limit int64, window time.Duration, opts ...Option) error {
	if name == "" {
		return NewError(ErrCodeInvalidQuota, "quota name cannot be empty")
	}
	if limit < 0 {
		return NewError(ErrCodeInvalidQuota, "quota limit cannot be negative")
	}
	if window < 0 {
		return NewError(ErrCodeInvalidQuota, "quota window cannot be negative")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.quotas[name]; exists {
		return NewError(ErrCodeQuotaExists, fmt.Sprintf("quota %s already exists", name))
	}

	now := time.Now()
	quota := &Quota{
		Name:         name,
		Type:         quotaType,
		Limit:        limit,
		Window:       window,
		Usage:        0,
		LastReset:    now,
		LastUpdate:   now,
		RegisteredAt: now,
	}

	// Apply options
	for _, opt := range opts {
		opt(quota)
	}

	m.quotas[name] = quota
	atomic.AddInt64(&m.stats.TotalQuotas, 1)

	return nil
}

// Unregister removes a quota.
func (m *quotaManager) Unregister(name string) error {
	if name == "" {
		return NewError(ErrCodeInvalidQuota, "quota name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.quotas[name]; !exists {
		return NewError(ErrCodeQuotaNotFound, fmt.Sprintf("quota %s not found", name))
	}

	delete(m.quotas, name)
	atomic.AddInt64(&m.stats.TotalQuotas, -1)

	return nil
}

// Acquire attempts to acquire quota for the given name.
func (m *quotaManager) Acquire(ctx context.Context, name string, amount int64) (bool, int64, int64, error) {
	if ctx == nil {
		return false, 0, 0, NewError(ErrCodeNilContext, "context cannot be nil")
	}
	if name == "" {
		return false, 0, 0, NewError(ErrCodeInvalidQuota, "quota name cannot be empty")
	}
	if amount < 0 {
		return false, 0, 0, NewError(ErrCodeInvalidQuota, "amount cannot be negative")
	}

	m.mu.RLock()
	quota, exists := m.quotas[name]
	m.mu.RUnlock()

	if !exists {
		return false, 0, 0, NewError(ErrCodeQuotaNotFound, fmt.Sprintf("quota %s not found", name))
	}

	quota.mu.Lock()
	defer quota.mu.Unlock()

	// Check if window has expired and reset if needed
	now := time.Now()
	if quota.Window > 0 && now.Sub(quota.LastReset) >= quota.Window {
		atomic.StoreInt64(&quota.Usage, 0)
		quota.LastReset = now
		m.recordEvent(Event{
			Name:      name,
			Type:      quota.Type,
			EventType: "reset",
			Limit:     quota.Limit,
			Usage:     0,
			Timestamp: now,
		})
		m.onQuotaReset(name)
	}

	// Check current usage
	currentUsage := atomic.LoadInt64(&quota.Usage)
	newUsage := currentUsage + amount

	// Check if quota would be exceeded
	if quota.Limit > 0 && newUsage > quota.Limit {
		atomic.AddInt64(&m.stats.TotalExceeded, 1)
		if m.config.EnableMetrics {
			atomic.AddInt64(&quota.TotalExceeded, 1)
		}
		quota.LastUpdate = now
		m.recordEvent(Event{
			Name:      name,
			Type:      quota.Type,
			EventType: "exceeded",
			Limit:     quota.Limit,
			Usage:     currentUsage,
			Timestamp: now,
		})
		m.onQuotaExceeded(name, quota.Type, quota.Limit, currentUsage)
		return false, currentUsage, quota.Limit - currentUsage, nil
	}

	// Acquire quota
	atomic.StoreInt64(&quota.Usage, newUsage)
	atomic.AddInt64(&m.stats.TotalAcquired, 1)
	if m.config.EnableMetrics {
		atomic.AddInt64(&quota.TotalAcquired, 1)
	}
	quota.LastUpdate = now

	remaining := int64(-1) // -1 indicates unlimited
	if quota.Limit > 0 {
		remaining = quota.Limit - newUsage
		if remaining < 0 {
			remaining = 0
		}
	}

	return true, newUsage, remaining, nil
}

// Release releases quota for the given name.
func (m *quotaManager) Release(name string, amount int64) error {
	if name == "" {
		return NewError(ErrCodeInvalidQuota, "quota name cannot be empty")
	}
	if amount < 0 {
		return NewError(ErrCodeInvalidQuota, "amount cannot be negative")
	}

	m.mu.RLock()
	quota, exists := m.quotas[name]
	m.mu.RUnlock()

	if !exists {
		return NewError(ErrCodeQuotaNotFound, fmt.Sprintf("quota %s not found", name))
	}

	quota.mu.Lock()
	defer quota.mu.Unlock()

	currentUsage := atomic.LoadInt64(&quota.Usage)
	newUsage := currentUsage - amount
	if newUsage < 0 {
		newUsage = 0
	}

	atomic.StoreInt64(&quota.Usage, newUsage)
	atomic.AddInt64(&m.stats.TotalReleased, 1)
	if m.config.EnableMetrics {
		atomic.AddInt64(&quota.TotalReleased, 1)
	}
	quota.LastUpdate = time.Now()

	return nil
}

// GetUsage returns the current usage for a quota.
func (m *quotaManager) GetUsage(name string) (int64, error) {
	if name == "" {
		return 0, NewError(ErrCodeInvalidQuota, "quota name cannot be empty")
	}

	m.mu.RLock()
	quota, exists := m.quotas[name]
	m.mu.RUnlock()

	if !exists {
		return 0, NewError(ErrCodeQuotaNotFound, fmt.Sprintf("quota %s not found", name))
	}

	return atomic.LoadInt64(&quota.Usage), nil
}

// GetRemaining returns the remaining quota for a quota.
func (m *quotaManager) GetRemaining(name string) (int64, error) {
	if name == "" {
		return 0, NewError(ErrCodeInvalidQuota, "quota name cannot be empty")
	}

	m.mu.RLock()
	quota, exists := m.quotas[name]
	m.mu.RUnlock()

	if !exists {
		return 0, NewError(ErrCodeQuotaNotFound, fmt.Sprintf("quota %s not found", name))
	}

	usage := atomic.LoadInt64(&quota.Usage)
	if quota.Limit <= 0 {
		return -1, nil // unlimited
	}

	remaining := quota.Limit - usage
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// Reset resets the usage for a quota.
func (m *quotaManager) Reset(name string) error {
	if name == "" {
		return NewError(ErrCodeInvalidQuota, "quota name cannot be empty")
	}

	m.mu.RLock()
	quota, exists := m.quotas[name]
	m.mu.RUnlock()

	if !exists {
		return NewError(ErrCodeQuotaNotFound, fmt.Sprintf("quota %s not found", name))
	}

	quota.mu.Lock()
	defer quota.mu.Unlock()

	now := time.Now()
	atomic.StoreInt64(&quota.Usage, 0)
	quota.LastReset = now
	quota.LastUpdate = now
	atomic.AddInt64(&m.stats.TotalReset, 1)

	m.recordEvent(Event{
		Name:      name,
		Type:      quota.Type,
		EventType: "reset",
		Limit:     quota.Limit,
		Usage:     0,
		Timestamp: now,
	})
	m.onQuotaReset(name)

	return nil
}

// ResetAll resets all quotas.
func (m *quotaManager) ResetAll() error {
	m.mu.RLock()
	names := make([]string, 0, len(m.quotas))
	for name := range m.quotas {
		names = append(names, name)
	}
	m.mu.RUnlock()

	var errors []error
	for _, name := range names {
		if err := m.Reset(name); err != nil {
			errors = append(errors, fmt.Errorf("failed to reset %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("reset completed with %d error(s): %v", len(errors), errors)
	}

	return nil
}

// Clear removes all quotas.
func (m *quotaManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.quotas = make(map[string]*Quota)
	atomic.StoreInt64(&m.stats.TotalQuotas, 0)
}

// GetQuota returns information about a registered quota.
func (m *quotaManager) GetQuota(name string) (*QuotaInfo, error) {
	if name == "" {
		return nil, NewError(ErrCodeInvalidQuota, "quota name cannot be empty")
	}

	m.mu.RLock()
	quota, exists := m.quotas[name]
	m.mu.RUnlock()

	if !exists {
		return nil, NewError(ErrCodeQuotaNotFound, fmt.Sprintf("quota %s not found", name))
	}

	quota.mu.RLock()
	defer quota.mu.RUnlock()

	usage := atomic.LoadInt64(&quota.Usage)
	remaining := int64(-1) // -1 indicates unlimited
	if quota.Limit > 0 {
		remaining = quota.Limit - usage
		if remaining < 0 {
			remaining = 0
		}
	}

	return &QuotaInfo{
		Name:         quota.Name,
		Type:         quota.Type,
		Limit:        quota.Limit,
		Window:       quota.Window,
		Usage:        usage,
		Remaining:    remaining,
		RegisteredAt: quota.RegisteredAt,
		LastReset:    quota.LastReset,
		LastUpdate:   quota.LastUpdate,
	}, nil
}

// Stats returns statistics about quota operations.
func (m *quotaManager) Stats() Stats {
	return Stats{
		TotalQuotas:    atomic.LoadInt64(&m.stats.TotalQuotas),
		TotalAcquired:  atomic.LoadInt64(&m.stats.TotalAcquired),
		TotalExceeded:  atomic.LoadInt64(&m.stats.TotalExceeded),
		TotalReleased:  atomic.LoadInt64(&m.stats.TotalReleased),
		TotalReset:     atomic.LoadInt64(&m.stats.TotalReset),
		LastEventTime:  m.stats.LastEventTime,
	}
}

// QuotaStats returns statistics for a specific quota.
func (m *quotaManager) QuotaStats(name string) (*QuotaStat, error) {
	if name == "" {
		return nil, NewError(ErrCodeInvalidQuota, "quota name cannot be empty")
	}

	m.mu.RLock()
	quota, exists := m.quotas[name]
	m.mu.RUnlock()

	if !exists {
		return nil, NewError(ErrCodeQuotaNotFound, fmt.Sprintf("quota %s not found", name))
	}

	quota.mu.RLock()
	defer quota.mu.RUnlock()

	usage := atomic.LoadInt64(&quota.Usage)
	remaining := int64(-1) // -1 indicates unlimited
	if quota.Limit > 0 {
		remaining = quota.Limit - usage
		if remaining < 0 {
			remaining = 0
		}
	}

	// Get per-quota statistics
	exceeded := int64(0)
	totalAcquired := int64(0)
	totalReleased := int64(0)
	if m.config.EnableMetrics {
		exceeded = atomic.LoadInt64(&quota.TotalExceeded)
		totalAcquired = atomic.LoadInt64(&quota.TotalAcquired)
		totalReleased = atomic.LoadInt64(&quota.TotalReleased)
	} else {
		// Fallback to counting from history if metrics not enabled but history exists
		m.historyMu.RLock()
		for _, event := range m.history {
			if event.Name == name && event.EventType == "exceeded" {
				exceeded++
			}
		}
		m.historyMu.RUnlock()
	}

	return &QuotaStat{
		Name:          quota.Name,
		Type:          quota.Type,
		Limit:         quota.Limit,
		Usage:         usage,
		Remaining:     remaining,
		Window:        quota.Window,
		LastReset:     quota.LastReset,
		LastUpdate:    quota.LastUpdate,
		Exceeded:      exceeded,
		TotalAcquired: totalAcquired,
		TotalReleased: totalReleased,
	}, nil
}

// ListQuotas returns a list of all registered quotas, sorted by name.
func (m *quotaManager) ListQuotas() []QuotaInfo {
	m.mu.RLock()
	quotas := make([]*Quota, 0, len(m.quotas))
	for _, quota := range m.quotas {
		quotas = append(quotas, quota)
	}
	m.mu.RUnlock()

	// Sort by name
	sort.Slice(quotas, func(i, j int) bool {
		return quotas[i].Name < quotas[j].Name
	})

	infos := make([]QuotaInfo, 0, len(quotas))
	for _, quota := range quotas {
		quota.mu.RLock()
		usage := atomic.LoadInt64(&quota.Usage)
		remaining := int64(-1) // -1 indicates unlimited
		if quota.Limit > 0 {
			remaining = quota.Limit - usage
			if remaining < 0 {
				remaining = 0
			}
		}

		infos = append(infos, QuotaInfo{
			Name:         quota.Name,
			Type:         quota.Type,
			Limit:        quota.Limit,
			Window:       quota.Window,
			Usage:        usage,
			Remaining:    remaining,
			RegisteredAt: quota.RegisteredAt,
			LastReset:    quota.LastReset,
			LastUpdate:   quota.LastUpdate,
		})
		quota.mu.RUnlock()
	}

	return infos
}

// autoResetLoop runs the auto-reset loop.
func (m *quotaManager) autoResetLoop() {
	ticker := time.NewTicker(m.config.AutoResetInterval)
	defer ticker.Stop()
	defer close(m.doneChan)

	for {
		select {
		case <-ticker.C:
			m.ResetAll()
		case <-m.stopChan:
			return
		}
	}
}

// Stop stops the quota manager (stops auto-reset if running).
func (m *quotaManager) Stop() error {
	if !atomic.CompareAndSwapInt32(&m.running, 1, 0) {
		return nil // Already stopped or never started
	}

	// Safely close the channel - only one goroutine can reach here
	select {
	case <-m.stopChan:
		// Already closed
	default:
		close(m.stopChan)
	}

	<-m.doneChan

	return nil
}

// recordEvent records an event in history.
func (m *quotaManager) recordEvent(event Event) {
	if m.config.HistorySize <= 0 {
		return
	}

	m.historyMu.Lock()
	defer m.historyMu.Unlock()

	m.history = append(m.history, event)
	if len(m.history) > m.config.HistorySize {
		m.history = m.history[1:]
	}

	m.stats.LastEventTime = event.Timestamp
}

// onQuotaExceeded invokes quota exceeded callbacks.
func (m *quotaManager) onQuotaExceeded(name string, quotaType QuotaType, limit, usage int64) {
	if m.config.OnQuotaExceeded != nil {
		m.config.OnQuotaExceeded(name, quotaType, limit, usage)
	}

	if m.config.OnQuotaExceededAsync != nil {
		go m.config.OnQuotaExceededAsync(name, quotaType, limit, usage)
	}
}

// onQuotaReset invokes quota reset callbacks.
func (m *quotaManager) onQuotaReset(name string) {
	if m.config.OnQuotaReset != nil {
		m.config.OnQuotaReset(name)
	}

	if m.config.OnQuotaResetAsync != nil {
		go m.config.OnQuotaResetAsync(name)
	}
}
