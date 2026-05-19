package quotas

import (
	"sync"
	"time"
)

// MemoryLimiter is an in-memory quota limiter: limit N units per key, refilled at refillInterval
// (or no refill for fixed total). Remaining is non-negative.
type MemoryLimiter struct {
	mu             sync.Mutex
	limit          int64
	refillInterval time.Duration
	usage          map[string]int64
	lastRefill     map[string]time.Time
}

// NewMemoryLimiter creates a limiter with limit per key. If refillInterval > 0,
// one unit is refilled per interval; otherwise usage never refills (fixed total).
func NewMemoryLimiter(limit int64, refillInterval time.Duration) *MemoryLimiter {
	return &MemoryLimiter{
		limit:          limit,
		refillInterval: refillInterval,
		usage:          make(map[string]int64),
		lastRefill:     make(map[string]time.Time),
	}
}

// Allow checks quota; if consume is true, increments usage. Refills based on elapsed time if refillInterval set.
func (m *MemoryLimiter) Allow(key string, consume bool) (allowed bool, remaining int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if m.refillInterval > 0 {
		if last, ok := m.lastRefill[key]; ok {
			elapsed := now.Sub(last)
			if elapsed >= m.refillInterval {
				refill := int64(elapsed / m.refillInterval)
				u := m.usage[key] - refill
				if u < 0 {
					u = 0
				}
				m.usage[key] = u
				m.lastRefill[key] = now
			}
		} else {
			m.lastRefill[key] = now
		}
	}

	used := m.usage[key]
	remaining = m.limit - used
	if remaining < 0 {
		remaining = 0
	}
	if used >= m.limit {
		return false, 0
	}
	if consume {
		m.usage[key] = used + 1
		remaining = m.limit - (used + 1)
		if remaining < 0 {
			remaining = 0
		}
	}
	return true, remaining
}

// Reset clears usage for key.
func (m *MemoryLimiter) Reset(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.usage, key)
	delete(m.lastRefill, key)
	return nil
}
