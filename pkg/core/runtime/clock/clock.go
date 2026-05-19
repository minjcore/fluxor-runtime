package clock

import (
	"time"
)

// Clock provides an abstraction over time operations.
// This allows for testing with mockable time and controlled time advancement.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// Sleep pauses the current goroutine for at least the duration d.
	// A negative or zero duration causes Sleep to return immediately.
	Sleep(d time.Duration)

	// After waits for the duration to elapse and then sends the current time
	// on the returned channel. It is equivalent to NewTimer(d).C.
	After(d time.Duration) <-chan time.Time

	// AfterFunc waits for the duration to elapse and then calls f in its own goroutine.
	// It returns a Timer that can be used to cancel the call using its Stop method.
	AfterFunc(d time.Duration, f func()) Timer

	// NewTimer creates a new Timer that will send the current time on its channel
	// after at least duration d.
	NewTimer(d time.Duration) Timer

	// NewTicker returns a new Ticker containing a channel that will send
	// the time with a period specified by the duration argument.
	// It adjusts the intervals or drops ticks to make up for slow receivers.
	NewTicker(d time.Duration) Ticker

	// Since returns the time elapsed since t.
	Since(t time.Time) time.Duration

	// Until returns the duration until t.
	Until(t time.Time) time.Duration
}

// Timer represents a single event.
// When the Timer expires, the current time will be sent on C, unless the Timer
// was created by AfterFunc. A Timer must be created with NewTimer or AfterFunc.
type Timer interface {
	// C returns the channel on which the time is delivered.
	C() <-chan time.Time

	// Stop prevents the Timer from firing. It returns true if the call stops
	// the timer, false if the timer has already expired or been stopped.
	Stop() bool

	// Reset changes the timer to expire after duration d.
	// It returns true if the timer had been active, false if the timer had
	// expired or been stopped.
	Reset(d time.Duration) bool
}

// Ticker holds a channel that delivers "ticks" of a clock at intervals.
type Ticker interface {
	// C returns the channel on which the ticks are delivered.
	C() <-chan time.Time

	// Stop stops a ticker. After Stop, no more ticks will be sent.
	// Stop does not close the channel, to prevent a concurrent goroutine
	// reading from the channel from seeing an erroneous "tick".
	Stop()
}

// SystemClock is the default clock implementation that uses the system time.
type SystemClock struct{}

// NewSystemClock creates a new system clock.
func NewSystemClock() Clock {
	return &SystemClock{}
}

// Now returns the current system time.
func (c *SystemClock) Now() time.Time {
	return time.Now()
}

// Sleep pauses the current goroutine for at least the duration d.
func (c *SystemClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// After waits for the duration to elapse and then sends the current time
// on the returned channel.
func (c *SystemClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// AfterFunc waits for the duration to elapse and then calls f in its own goroutine.
func (c *SystemClock) AfterFunc(d time.Duration, f func()) Timer {
	return &systemTimer{
		timer: time.AfterFunc(d, f),
	}
}

// NewTimer creates a new Timer that will send the current time on its channel
// after at least duration d.
func (c *SystemClock) NewTimer(d time.Duration) Timer {
	return &systemTimer{
		timer: time.NewTimer(d),
	}
}

// NewTicker returns a new Ticker containing a channel that will send
// the time with a period specified by the duration argument.
func (c *SystemClock) NewTicker(d time.Duration) Ticker {
	return &systemTicker{
		ticker: time.NewTicker(d),
	}
}

// Since returns the time elapsed since t.
func (c *SystemClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Until returns the duration until t.
func (c *SystemClock) Until(t time.Time) time.Duration {
	return time.Until(t)
}

// systemTimer wraps time.Timer to implement the Timer interface.
type systemTimer struct {
	timer *time.Timer
}

func (t *systemTimer) C() <-chan time.Time {
	if t.timer == nil {
		return nil
	}
	return t.timer.C
}

func (t *systemTimer) Stop() bool {
	if t.timer == nil {
		return false
	}
	return t.timer.Stop()
}

func (t *systemTimer) Reset(d time.Duration) bool {
	if t.timer == nil {
		return false
	}
	return t.timer.Reset(d)
}

// systemTicker wraps time.Ticker to implement the Ticker interface.
type systemTicker struct {
	ticker *time.Ticker
}

func (t *systemTicker) C() <-chan time.Time {
	if t.ticker == nil {
		return nil
	}
	return t.ticker.C
}

func (t *systemTicker) Stop() {
	if t.ticker != nil {
		t.ticker.Stop()
	}
}
