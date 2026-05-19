package signals

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Handler provides signal handling for graceful shutdown.
type Handler interface {
	// Start begins listening for signals. It blocks until a signal is received
	// or the context is cancelled.
	Start(ctx context.Context) error

	// Stop stops listening for signals.
	Stop() error

	// Wait waits for a shutdown signal. Returns the signal that was received.
	Wait(ctx context.Context) (os.Signal, error)

	// Channel returns a channel that receives signals.
	Channel() <-chan os.Signal

	// Stats returns statistics about signals received.
	Stats() Stats
}

// Stats contains statistics about signal handling.
type Stats struct {
	// SignalCount is the total number of signals received.
	SignalCount int64

	// LastSignal is the last signal that was received.
	LastSignal os.Signal

	// LastSignalTime is when the last signal was received.
	LastSignalTime time.Time

	// IsStarted indicates if the handler is currently started.
	IsStarted bool

	// IsStopped indicates if the handler has been stopped.
	IsStopped bool
}

// Config configures signal handling behavior.
type Config struct {
	// Signals is the list of signals to listen for.
	// If empty, defaults to os.Interrupt, syscall.SIGTERM, syscall.SIGINT.
	Signals []os.Signal

	// OnSignal is called when a signal is received.
	// If nil, no callback is invoked.
	OnSignal func(sig os.Signal)

	// OnSignalAsync is called asynchronously when a signal is received.
	// Unlike OnSignal, this is called in a separate goroutine, allowing
	// for non-blocking processing.
	OnSignalAsync func(sig os.Signal)

	// ShutdownTimeout is the maximum time to wait for shutdown callbacks
	// to complete. Zero means no timeout.
	ShutdownTimeout time.Duration

	// QueueSize is the size of the signal queue. Signals received when
	// the queue is full are dropped. Defaults to 10.
	QueueSize int

	// ContinueOnSignal determines whether to continue listening after
	// receiving a signal. If false, the handler stops after the first signal.
	// Defaults to false.
	ContinueOnSignal bool

	// SignalHistory enables tracking signal history. If true, the last
	// N signals are stored (up to QueueSize).
	SignalHistory bool
}

// DefaultConfig returns the default signal handler configuration.
func DefaultConfig() Config {
	return Config{
		Signals:         []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT},
		QueueSize:       10,
		ContinueOnSignal: false,
		SignalHistory:   false,
	}
}

// signalHandler implements the Handler interface.
type signalHandler struct {
	config          Config
	sigChan         chan os.Signal
	stopChan        chan struct{}
	doneChan        chan struct{}
	mu              sync.RWMutex
	started         bool
	stopped         bool
	notifyFunc      func(chan<- os.Signal, ...os.Signal)
	stopFunc        func(chan<- os.Signal)
	signalCount     int64
	lastSignal      atomic.Value // stores os.Signal
	lastSignalTime  atomic.Value // stores time.Time
	signalHistory   []os.Signal
	historyMu       sync.RWMutex
}

// NewHandler creates a new signal handler with the given configuration.
func NewHandler(config Config) Handler {
	if len(config.Signals) == 0 {
		config.Signals = DefaultConfig().Signals
	}
	if config.QueueSize == 0 {
		config.QueueSize = DefaultConfig().QueueSize
	}

	// Use default signal.Notify and signal.Stop functions
	// These can be overridden in tests
	notifyFunc := signal.Notify
	stopFunc := signal.Stop

	handler := &signalHandler{
		config:     config,
		sigChan:    make(chan os.Signal, config.QueueSize),
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
		notifyFunc: notifyFunc,
		stopFunc:   stopFunc,
	}

	if config.SignalHistory {
		handler.signalHistory = make([]os.Signal, 0, config.QueueSize)
	}

	return handler
}

// Start begins listening for signals.
func (h *signalHandler) Start(ctx context.Context) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		return NewError(ErrCodeAlreadyStarted, "signal handler is already started")
	}
	if h.stopped {
		return NewError(ErrCodeAlreadyStopped, "signal handler is already stopped")
	}

	h.started = true

	// Register signal notification
	h.notifyFunc(h.sigChan, h.config.Signals...)

	// Start goroutine to handle signals and context cancellation
	go h.run(ctx)

	return nil
}

// run is the main signal handling loop.
func (h *signalHandler) run(ctx context.Context) {
	defer close(h.doneChan)

	for {
		select {
		case sig := <-h.sigChan:
			if sig == nil {
				// Channel closed
				return
			}

			// Update statistics
			atomic.AddInt64(&h.signalCount, 1)
			h.lastSignal.Store(sig)
			h.lastSignalTime.Store(time.Now())

			// Track signal history if enabled
			if h.config.SignalHistory {
				h.historyMu.Lock()
				h.signalHistory = append(h.signalHistory, sig)
				if len(h.signalHistory) > h.config.QueueSize {
					h.signalHistory = h.signalHistory[1:]
				}
				h.historyMu.Unlock()
			}

			// Invoke synchronous callback if set
			if h.config.OnSignal != nil {
				h.config.OnSignal(sig)
			}

			// Invoke asynchronous callback if set
			if h.config.OnSignalAsync != nil {
				go h.config.OnSignalAsync(sig)
			}

			// Stop if not configured to continue
			if !h.config.ContinueOnSignal {
				h.Stop()
				return
			}

		case <-ctx.Done():
			// Context cancelled, stop listening
			h.Stop()
			return

		case <-h.stopChan:
			// Explicitly stopped
			return
		}
	}
}

// Stop stops listening for signals.
func (h *signalHandler) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopped {
		return nil
	}

	h.stopped = true
	h.started = false
	h.stopFunc(h.sigChan)

	// Close stopChan to signal the run loop to exit
	select {
	case <-h.stopChan:
		// Already closed
	default:
		close(h.stopChan)
	}

	// Wait for the run loop to finish (with optional timeout)
	if h.config.ShutdownTimeout > 0 {
		select {
		case <-h.doneChan:
			// Run loop finished
		case <-time.After(h.config.ShutdownTimeout):
			// Timeout waiting for shutdown
			return NewError(ErrCodeShutdownTimeout, fmt.Sprintf("shutdown timeout after %v", h.config.ShutdownTimeout))
		}
	} else {
		<-h.doneChan
	}

	return nil
}

// Wait waits for a shutdown signal. Returns the signal that was received.
func (h *signalHandler) Wait(ctx context.Context) (os.Signal, error) {
	if ctx == nil {
		return nil, NewError(ErrCodeNilContext, "context cannot be nil")
	}

	// Start if not already started
	h.mu.RLock()
	started := h.started
	h.mu.RUnlock()

	if !started {
		if err := h.Start(ctx); err != nil {
			return nil, err
		}
	}

	select {
	case sig := <-h.sigChan:
		if sig == nil {
			return nil, NewError(ErrCodeChannelClosed, "signal channel closed")
		}
		h.Stop()
		return sig, nil

	case <-ctx.Done():
		h.Stop()
		return nil, ctx.Err()

	case <-h.stopChan:
		return nil, NewError(ErrCodeHandlerStopped, "signal handler stopped")
	}
}

// Channel returns a channel that receives signals.
// The channel is buffered with capacity from Config.QueueSize.
func (h *signalHandler) Channel() <-chan os.Signal {
	return h.sigChan
}

// Stats returns statistics about signals received.
func (h *signalHandler) Stats() Stats {
	h.mu.RLock()
	started := h.started
	stopped := h.stopped
	h.mu.RUnlock()

	var lastSig os.Signal
	if v := h.lastSignal.Load(); v != nil {
		lastSig = v.(os.Signal)
	}

	var lastTime time.Time
	if v := h.lastSignalTime.Load(); v != nil {
		lastTime = v.(time.Time)
	}

	return Stats{
		SignalCount:    atomic.LoadInt64(&h.signalCount),
		LastSignal:     lastSig,
		LastSignalTime: lastTime,
		IsStarted:      started,
		IsStopped:      stopped,
	}
}

// GetHistory returns the signal history if enabled.
func (h *signalHandler) GetHistory() []os.Signal {
	if !h.config.SignalHistory {
		return nil
	}

	h.historyMu.RLock()
	defer h.historyMu.RUnlock()

	if len(h.signalHistory) == 0 {
		return nil
	}

	history := make([]os.Signal, len(h.signalHistory))
	copy(history, h.signalHistory)
	return history
}

// WaitForSignal is a convenience function that waits for a signal with default configuration.
func WaitForSignal(ctx context.Context) (os.Signal, error) {
	handler := NewHandler(DefaultConfig())
	return handler.Wait(ctx)
}

// WaitForSignalWithTimeout waits for a signal with a timeout.
func WaitForSignalWithTimeout(ctx context.Context, timeout time.Duration) (os.Signal, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return WaitForSignal(ctx)
}

// Listen creates a signal handler that automatically starts listening and calls
// the provided callback when a signal is received.
func Listen(ctx context.Context, config Config, callback func(os.Signal)) error {
	if callback == nil {
		return NewError(ErrCodeNilCallback, "callback cannot be nil")
	}

	config.OnSignal = callback
	handler := NewHandler(config)
	return handler.Start(ctx)
}

// ListenAsync creates a signal handler that automatically starts listening and calls
// the provided callback asynchronously when a signal is received.
func ListenAsync(ctx context.Context, config Config, callback func(os.Signal)) error {
	if callback == nil {
		return NewError(ErrCodeNilCallback, "callback cannot be nil")
	}

	config.OnSignalAsync = callback
	handler := NewHandler(config)
	return handler.Start(ctx)
}

// GracefulShutdown sets up graceful shutdown with signal handling.
// It calls onShutdown when a signal is received, with optional timeout support.
func GracefulShutdown(ctx context.Context, onShutdown func(), opts ...ShutdownOption) error {
	if onShutdown == nil {
		return NewError(ErrCodeNilCallback, "onShutdown cannot be nil")
	}

	config := DefaultConfig()

	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	config.OnSignal = func(sig os.Signal) {
		onShutdown()
	}

	handler := NewHandler(config)
	return handler.Start(ctx)
}

// ShutdownOption configures graceful shutdown behavior.
type ShutdownOption func(*Config)

// WithShutdownTimeout sets the shutdown timeout.
func WithShutdownTimeout(timeout time.Duration) ShutdownOption {
	return func(c *Config) {
		c.ShutdownTimeout = timeout
	}
}

// WithSignals sets the signals to listen for.
func WithSignals(signals ...os.Signal) ShutdownOption {
	return func(c *Config) {
		c.Signals = signals
	}
}

// WithContinueOnSignal allows continuing to listen after receiving a signal.
func WithContinueOnSignal(continueOnSignal bool) ShutdownOption {
	return func(c *Config) {
		c.ContinueOnSignal = continueOnSignal
	}
}

// WithSignalHistory enables signal history tracking.
func WithSignalHistory(enabled bool) ShutdownOption {
	return func(c *Config) {
		c.SignalHistory = enabled
	}
}
