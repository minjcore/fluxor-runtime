package clock

import (
	"sync"
	"time"
)

// MockClock is a mock clock implementation for testing.
// It allows controlling time and advancing it programmatically.
type MockClock struct {
	mu        sync.RWMutex
	now       time.Time
	timers    []*mockTimer
	tickers   []*mockTicker
	callbacks []func()
}

// NewMockClock creates a new mock clock with the given initial time.
func NewMockClock(initialTime time.Time) *MockClock {
	return &MockClock{
		now:     initialTime,
		timers:  make([]*mockTimer, 0),
		tickers: make([]*mockTicker, 0),
	}
}

// Now returns the current mock time.
func (c *MockClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

// Sleep advances the clock by the given duration.
// In a mock clock, this immediately advances time rather than blocking.
func (c *MockClock) Sleep(d time.Duration) {
	c.Advance(d)
}

// Advance advances the clock by the given duration and triggers any
// timers or tickers that should fire.
func (c *MockClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if d <= 0 {
		return
	}

	c.now = c.now.Add(d)

	// Check and fire timers
	for i := 0; i < len(c.timers); i++ {
		timer := c.timers[i]
		if timer != nil && !timer.stopped && !timer.fired && c.now.After(timer.deadline) || c.now.Equal(timer.deadline) {
			timer.fired = true
			if timer.callback != nil {
				// Call callback in a goroutine to simulate AfterFunc behavior
				go timer.callback()
			} else if timer.ch != nil {
				// Send to channel (non-blocking)
				select {
				case timer.ch <- c.now:
				default:
				}
			}
		}
	}

	// Check and fire tickers
	for _, ticker := range c.tickers {
		if ticker != nil && !ticker.stopped {
			// Calculate how many ticks should have fired since lastTick
			elapsed := c.now.Sub(ticker.lastTick)
			if elapsed >= ticker.interval {
				ticks := int(elapsed / ticker.interval)
				for i := 0; i < ticks; i++ {
					// Send to channel (non-blocking)
					select {
					case ticker.ch <- ticker.nextTick:
					default:
					}
					ticker.lastTick = ticker.nextTick
					ticker.nextTick = ticker.nextTick.Add(ticker.interval)
				}
			}
		}
	}
}

// SetTime sets the current time to the given time.
func (c *MockClock) SetTime(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

// After waits for the duration to elapse and then sends the current time
// on the returned channel.
func (c *MockClock) After(d time.Duration) <-chan time.Time {
	timer := c.NewTimer(d)
	return timer.C()
}

// AfterFunc waits for the duration to elapse and then calls f in its own goroutine.
func (c *MockClock) AfterFunc(d time.Duration, f func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()

	timer := &mockTimer{
		clock:    c,
		deadline: c.now.Add(d),
		callback: f,
	}

	c.timers = append(c.timers, timer)

	// If the timer should fire immediately, fire it
	if c.now.After(timer.deadline) || c.now.Equal(timer.deadline) {
		timer.fired = true
		go f()
	}

	return timer
}

// NewTimer creates a new Timer that will send the current time on its channel
// after at least duration d.
func (c *MockClock) NewTimer(d time.Duration) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()

	timer := &mockTimer{
		clock:    c,
		deadline: c.now.Add(d),
		ch:       make(chan time.Time, 1),
	}

	c.timers = append(c.timers, timer)

	// If the timer should fire immediately, fire it
	if c.now.After(timer.deadline) || c.now.Equal(timer.deadline) {
		timer.fired = true
		select {
		case timer.ch <- c.now:
		default:
		}
	}

	return timer
}

// NewTicker returns a new Ticker containing a channel that will send
// the time with a period specified by the duration argument.
func (c *MockClock) NewTicker(d time.Duration) Ticker {
	c.mu.Lock()
	defer c.mu.Unlock()

	ticker := &mockTicker{
		clock:     c,
		interval:  d,
		ch:        make(chan time.Time, 100), // Buffer to accommodate multiple ticks
		lastTick:  c.now,
		nextTick:  c.now.Add(d),
	}

	c.tickers = append(c.tickers, ticker)

	return ticker
}

// Since returns the time elapsed since t.
func (c *MockClock) Since(t time.Time) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now.Sub(t)
}

// Until returns the duration until t.
func (c *MockClock) Until(t time.Time) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return t.Sub(c.now)
}

// removeTimer removes a timer from the clock's timer list.
func (c *MockClock) removeTimer(timer *mockTimer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, t := range c.timers {
		if t == timer {
			c.timers = append(c.timers[:i], c.timers[i+1:]...)
			return
		}
	}
}

// removeTicker removes a ticker from the clock's ticker list.
func (c *MockClock) removeTicker(ticker *mockTicker) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, t := range c.tickers {
		if t == ticker {
			c.tickers = append(c.tickers[:i], c.tickers[i+1:]...)
			return
		}
	}
}

// mockTimer implements Timer for MockClock.
type mockTimer struct {
	clock    *MockClock
	deadline time.Time
	ch       chan time.Time
	callback func()
	stopped  bool
	fired    bool
	mu       sync.Mutex
}

func (t *mockTimer) C() <-chan time.Time {
	return t.ch
}

func (t *mockTimer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped || t.fired {
		return false
	}

	t.stopped = true
	t.clock.removeTimer(t)
	return true
}

func (t *mockTimer) Reset(d time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	wasActive := !t.stopped && !t.fired

	t.stopped = false
	t.fired = false
	t.deadline = t.clock.Now().Add(d)

	if wasActive {
		t.clock.removeTimer(t)
	}

	t.clock.mu.Lock()
	t.clock.timers = append(t.clock.timers, t)
	t.clock.mu.Unlock()

	// If the timer should fire immediately, fire it
	if t.clock.now.After(t.deadline) || t.clock.now.Equal(t.deadline) {
		t.fired = true
		if t.callback != nil {
			go t.callback()
		} else if t.ch != nil {
			select {
			case t.ch <- t.clock.now:
			default:
			}
		}
	}

	return wasActive
}

// mockTicker implements Ticker for MockClock.
type mockTicker struct {
	clock    *MockClock
	interval time.Duration
	ch       chan time.Time
	lastTick time.Time
	nextTick time.Time
	stopped  bool
	mu       sync.Mutex
}

func (t *mockTicker) C() <-chan time.Time {
	return t.ch
}

func (t *mockTicker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return
	}

	t.stopped = true
	t.clock.removeTicker(t)
}
