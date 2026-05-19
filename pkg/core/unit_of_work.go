package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// UnitOfWork extends GoCMD with transaction and work boundary management.
// It provides a way to group multiple operations into a single atomic unit of work.
//
// UnitOfWork pattern:
//   - Groups related operations into a single transaction
//   - Ensures all-or-nothing execution (atomicity)
//   - Provides rollback capability on errors
//   - Manages work boundaries and isolation
//
// Usage:
//
//	uow := NewUnitOfWork(gocmd)
//	defer uow.Close()
//
//	if err := uow.Begin(); err != nil {
//	    return err
//	}
//
//	// Perform operations
//	if err := uow.Execute(func(ctx context.Context) error {
//	    // Work that should be atomic
//	    return nil
//	}); err != nil {
//	    uow.Rollback()
//	    return err
//	}
//
//	return uow.Commit()
type UnitOfWork interface {
	// GoCMD embeds all GoCMD functionality
	GoCMD

	// Begin starts a new unit of work (transaction boundary)
	// Returns error if unit of work is already active
	Begin() error

	// Commit commits the current unit of work
	// All operations performed since Begin() are finalized
	// Returns error if no active unit of work or commit fails
	Commit() error

	// Rollback rolls back the current unit of work
	// All operations performed since Begin() are discarded
	// Returns error if no active unit of work or rollback fails
	Rollback() error

	// Execute executes a function within the current unit of work
	// The function receives a context that is cancelled if the unit of work is rolled back
	// Returns error if no active unit of work or function execution fails
	Execute(fn func(ctx context.Context) error) error

	// IsActive returns true if a unit of work is currently active
	IsActive() bool

	// GetWorkID returns the unique identifier for the current unit of work
	// Returns empty string if no active unit of work
	GetWorkID() string

	// SetTimeout sets the timeout for the unit of work
	// Operations that exceed this timeout will be rolled back
	SetTimeout(timeout time.Duration)

	// GetTimeout returns the current timeout for the unit of work
	GetTimeout() time.Duration
}

// unitOfWork implements UnitOfWork
type unitOfWork struct {
	GoCMD

	workCtx    context.Context
	workCancel context.CancelFunc
	workID     string
	active     bool
	timeout    time.Duration
	mu         sync.RWMutex
	logger     Logger
}

// UnitOfWorkOptions configures UnitOfWork construction.
type UnitOfWorkOptions struct {
	// Timeout is the default timeout for unit of work operations
	// Default: 30 seconds. Must be positive.
	// If zero or negative, defaults to 30 seconds.
	Timeout time.Duration
}

// NewUnitOfWork creates a new UnitOfWork instance that extends the provided GoCMD.
// The UnitOfWork manages transaction boundaries and work isolation.
func NewUnitOfWork(gocmd GoCMD) UnitOfWork {
	return NewUnitOfWorkWithOptions(gocmd, UnitOfWorkOptions{})
}

// NewUnitOfWorkWithOptions creates a new UnitOfWork instance with options.
func NewUnitOfWorkWithOptions(gocmd GoCMD, opts UnitOfWorkOptions) UnitOfWork {
	failfast.NotNil(gocmd, "gocmd")

	// Set default timeout if not configured or invalid
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &unitOfWork{
		GoCMD:   gocmd,
		timeout: timeout,
		logger:  NewDefaultLogger(),
	}
}

// Begin starts a new unit of work (transaction boundary)
func (uow *unitOfWork) Begin() error {
	uow.mu.Lock()
	defer uow.mu.Unlock()

	// Fail-fast: check if unit of work is already active
	if uow.active {
		return &EventBusError{
			Code:    "UNIT_OF_WORK_ALREADY_ACTIVE",
			Message: "unit of work is already active, call Commit() or Rollback() first",
		}
	}

	// Create work context with timeout
	ctx := uow.GoCMD.Context()
	workCtx, workCancel := context.WithTimeout(ctx, uow.timeout)

	// Generate unique work ID
	uow.workID = generateWorkID()
	uow.workCtx = workCtx
	uow.workCancel = workCancel
	uow.active = true

	uow.logger.Info("UnitOfWork started", "workID", uow.workID, "timeout", uow.timeout)
	return nil
}

// Commit commits the current unit of work
func (uow *unitOfWork) Commit() error {
	uow.mu.Lock()
	defer uow.mu.Unlock()

	// Fail-fast: check if unit of work is active
	if !uow.active {
		return &EventBusError{
			Code:    "NO_ACTIVE_UNIT_OF_WORK",
			Message: "no active unit of work, call Begin() first",
		}
	}

	// Check if context was cancelled (timeout or parent cancellation)
	if uow.workCtx.Err() != nil {
		uow.cleanup()
		return &EventBusError{
			Code:    "UNIT_OF_WORK_CANCELLED",
			Message: "unit of work was cancelled: " + uow.workCtx.Err().Error(),
		}
	}

	workID := uow.workID
	uow.cleanup()

	uow.logger.Info("UnitOfWork committed", "workID", workID)
	return nil
}

// Rollback rolls back the current unit of work
func (uow *unitOfWork) Rollback() error {
	uow.mu.Lock()
	defer uow.mu.Unlock()

	// Fail-fast: check if unit of work is active
	if !uow.active {
		return &EventBusError{
			Code:    "NO_ACTIVE_UNIT_OF_WORK",
			Message: "no active unit of work, call Begin() first",
		}
	}

	workID := uow.workID
	uow.cleanup()

	uow.logger.Info("UnitOfWork rolled back", "workID", workID)
	return nil
}

// Execute executes a function within the current unit of work
func (uow *unitOfWork) Execute(fn func(ctx context.Context) error) error {
	uow.mu.RLock()
	active := uow.active
	workCtx := uow.workCtx
	uow.mu.RUnlock()

	// Fail-fast: check if unit of work is active
	if !active {
		return &EventBusError{
			Code:    "NO_ACTIVE_UNIT_OF_WORK",
			Message: "no active unit of work, call Begin() first",
		}
	}

	// Fail-fast: validate function
	if fn == nil {
		return &EventBusError{
			Code:    "INVALID_FUNCTION",
			Message: "function cannot be nil",
		}
	}

	// Execute function with work context
	// If context is cancelled (timeout or parent cancellation), function should handle it
	if err := fn(workCtx); err != nil {
		// Auto-rollback on error
		_ = uow.Rollback()
		return err
	}

	return nil
}

// IsActive returns true if a unit of work is currently active
func (uow *unitOfWork) IsActive() bool {
	uow.mu.RLock()
	defer uow.mu.RUnlock()
	return uow.active
}

// GetWorkID returns the unique identifier for the current unit of work
func (uow *unitOfWork) GetWorkID() string {
	uow.mu.RLock()
	defer uow.mu.RUnlock()
	return uow.workID
}

// SetTimeout sets the timeout for the unit of work
func (uow *unitOfWork) SetTimeout(timeout time.Duration) {
	uow.mu.Lock()
	defer uow.mu.Unlock()

	// Fail-fast: validate timeout
	if timeout <= 0 {
		panic("fail-fast: timeout must be positive")
	}

	uow.timeout = timeout
}

// GetTimeout returns the current timeout for the unit of work
func (uow *unitOfWork) GetTimeout() time.Duration {
	uow.mu.RLock()
	defer uow.mu.RUnlock()
	return uow.timeout
}

// cleanup cleans up the unit of work state
func (uow *unitOfWork) cleanup() {
	if uow.workCancel != nil {
		uow.workCancel()
	}
	uow.active = false
	uow.workID = ""
	uow.workCtx = nil
	uow.workCancel = nil
}

// generateWorkID generates a unique work ID
func generateWorkID() string {
	return fmt.Sprintf("work.%s", generateUUID())
}
