package core

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// ServerState is the lifecycle of a BaseServer (single writer, RWMutex readers).
type ServerState uint8

const (
	// ServerIdle — never successfully started, or fully reset (initial state).
	ServerIdle ServerState = iota
	// ServerStarting — start hook is running (IsStarted is true for observability).
	ServerStarting
	// ServerRunning — start hook succeeded.
	ServerRunning
	// ServerStopping — stop hook is running.
	ServerStopping
	// ServerHalted — Stop() completed; same as old “started && stopped” terminal state.
	ServerHalted
)

// BaseServer provides a Java-style abstract base class for HTTP servers
// It implements common lifecycle management and provides hook methods for customization
// Similar to Java's abstract base class pattern
type BaseServer struct {
	// Name of the server (can be set by subclasses)
	name string

	// GoCMD reference - GoCMD instance
	gocmd GoCMD

	// State management (replaces ambiguous started+stopped pair)
	mu    sync.RWMutex
	state ServerState

	// Root lifecycle context: created when entering ServerStarting, canceled on Stop.
	rootCtx    context.Context
	rootCancel context.CancelFunc

	// Logger for server operations
	logger Logger

	// Hook functions for template method pattern.
	// In Go, embedded-method "overrides" are not dispatched dynamically when the
	// embedded type calls its own methods, so we store explicit hooks instead.
	startHookCtx func(context.Context) error
	stopHookCtx  func(context.Context) error
}

// NewBaseServer creates a new BaseServer
func NewBaseServer(name string, gocmd GoCMD) *BaseServer {
	failfast.NotNil(gocmd, "gocmd") // Fail-fast: gocmd cannot be nil
	return &BaseServer{
		name:   name,
		gocmd:  gocmd,
		logger: NewDefaultLogger(),
	}
}

// SetHooks configures hook functions for Start/Stop (legacy signatures; context is ignored).
// Call this from the concrete server after construction:
//
//	s.BaseServer.SetHooks(s.doStart, s.doStop)
func (bs *BaseServer) SetHooks(startHook func() error, stopHook func() error) {
	bs.SetHooksContext(
		func(ctx context.Context) error { return startHook() },
		func(ctx context.Context) error { return stopHook() },
	)
}

// SetHooksContext configures hooks that receive the server root context (cancellation, deadlines, tracing).
func (bs *BaseServer) SetHooksContext(startHook func(context.Context) error, stopHook func(context.Context) error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.startHookCtx = startHook
	bs.stopHookCtx = stopHook
}

// Start implements Server.Start with template method pattern
// Subclasses should override doStart() for custom initialization
func (bs *BaseServer) Start() error {
	bs.mu.Lock()
	switch bs.state {
	case ServerStarting, ServerRunning:
		bs.mu.Unlock()
		return &EventBusError{Code: "ALREADY_STARTED", Message: "server already started"}
	case ServerHalted:
		bs.mu.Unlock()
		return &EventBusError{Code: "ALREADY_STARTED", Message: "server already started"}
	case ServerStopping:
		bs.mu.Unlock()
		return &EventBusError{Code: "ALREADY_STARTED", Message: "server already started"}
	}
	if bs.state != ServerIdle {
		bs.mu.Unlock()
		return &EventBusError{Code: "ALREADY_STARTED", Message: "server already started"}
	}

	bs.state = ServerStarting
	startHook := bs.startHookCtx
	if startHook == nil {
		startHook = bs.doStart
	}
	rootCtx, cancel := context.WithCancel(context.Background())
	bs.rootCtx = rootCtx
	bs.rootCancel = cancel
	bs.mu.Unlock()

	err := startHook(rootCtx)
	if err != nil {
		bs.mu.Lock()
		cancel()
		bs.rootCtx = nil
		bs.rootCancel = nil
		bs.state = ServerIdle
		bs.mu.Unlock()
		return err
	}

	bs.mu.Lock()
	bs.state = ServerRunning
	bs.mu.Unlock()
	return nil
}

// Stop implements Server.Stop with template method pattern
// Subclasses should override doStop() for custom cleanup
func (bs *BaseServer) Stop() error {
	bs.mu.Lock()
	switch bs.state {
	case ServerIdle, ServerHalted:
		bs.mu.Unlock()
		return nil
	case ServerStopping:
		bs.mu.Unlock()
		return nil
	}
	// ServerStarting or ServerRunning
	bs.state = ServerStopping
	stopHook := bs.stopHookCtx
	if stopHook == nil {
		stopHook = bs.doStop
	}
	rootCtx := bs.rootCtx
	cancel := bs.rootCancel
	bs.mu.Unlock()

	var err error
	if stopHook != nil {
		if rootCtx == nil {
			rootCtx = context.Background()
		}
		err = stopHook(rootCtx)
	}
	if cancel != nil {
		cancel()
	}

	bs.mu.Lock()
	bs.rootCtx = nil
	bs.rootCancel = nil
	bs.state = ServerHalted
	bs.mu.Unlock()
	return err
}

// doStart is a hook method for subclasses to override
// Default implementation does nothing
func (bs *BaseServer) doStart(ctx context.Context) error {
	_ = ctx
	return nil
}

// doStop is a hook method for subclasses to override
// Default implementation does nothing
func (bs *BaseServer) doStop(ctx context.Context) error {
	_ = ctx
	return nil
}

// Context returns the server root context after Start() begins (until Stop() completes).
// Before Start or after Stop, returns context.Background().
func (bs *BaseServer) Context() context.Context {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	if bs.rootCtx != nil {
		return bs.rootCtx
	}
	return context.Background()
}

// State returns the current lifecycle state (for health / diagnostics).
func (bs *BaseServer) State() ServerState {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.state
}

// Name returns the server name
func (bs *BaseServer) Name() string {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.name
}

// GoCMD returns the GoCMD reference (kept as GoCMD for backward compatibility)
func (bs *BaseServer) GoCMD() GoCMD {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.gocmd
}

// EventBus returns the EventBus reference
func (bs *BaseServer) EventBus() EventBus {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	if bs.gocmd == nil {
		return nil
	}
	return bs.gocmd.EventBus()
}

// Logger returns the logger instance
func (bs *BaseServer) Logger() Logger {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.logger
}

// SetLogger sets a custom logger for this server
func (bs *BaseServer) SetLogger(logger Logger) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.logger = logger
}

// IsStarted returns true after Start() has entered the start hook until the process is halted
// (matches legacy: true while start hook blocks, and true after Stop()).
func (bs *BaseServer) IsStarted() bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	switch bs.state {
	case ServerStarting, ServerRunning, ServerHalted:
		return true
	default:
		return false
	}
}

// IsStopped returns true after Stop() has completed successfully.
func (bs *BaseServer) IsStopped() bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.state == ServerHalted
}
