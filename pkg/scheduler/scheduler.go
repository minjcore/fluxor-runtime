package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/google/uuid"
)

// scheduler implements the Scheduler interface.
type scheduler struct {
	mu      sync.RWMutex
	tasks   map[string]*scheduledTaskEntry
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	stopped bool
}

// scheduledTaskEntry represents an internal scheduled task.
type scheduledTaskEntry struct {
	id       string
	task     concurrency.Task
	spec     ScheduleSpec
	nextRun  time.Time
	lastRun  time.Time
	runCount int64
	mu       sync.Mutex
}

// NewScheduler creates a new Scheduler instance.
//
// Example:
//
//	ctx := context.Background()
//	scheduler := scheduler.NewScheduler(ctx)
func NewScheduler(ctx context.Context) Scheduler {
	if ctx == nil {
		panic(fmt.Errorf("fail-fast: context cannot be nil"))
	}

	ctx, cancel := context.WithCancel(ctx)

	return &scheduler{
		tasks:  make(map[string]*scheduledTaskEntry),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Schedule implements Scheduler interface.
func (s *scheduler) Schedule(task concurrency.Task, spec ScheduleSpec) (string, error) {
	// Fail-fast: task cannot be nil
	if task == nil {
		return "", fmt.Errorf("task cannot be nil")
	}

	// Fail-fast: spec cannot be nil
	if spec == nil {
		return "", fmt.Errorf("schedule spec cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Fail-fast: cannot schedule after stop
	if s.stopped {
		return "", fmt.Errorf("scheduler is stopped")
	}

	// Generate unique ID
	id := uuid.New().String()

	// Calculate next run time
	nextRun := spec.Next(time.Now())

	entry := &scheduledTaskEntry{
		id:      id,
		task:    task,
		spec:    spec,
		nextRun: nextRun,
	}

	s.tasks[id] = entry

	return id, nil
}

// Unschedule implements Scheduler interface.
func (s *scheduler) Unschedule(id string) error {
	// Fail-fast: id cannot be empty
	if id == "" {
		return fmt.Errorf("task id cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[id]; !exists {
		return fmt.Errorf("task with id %q not found", id)
	}

	delete(s.tasks, id)
	return nil
}

// Start implements Scheduler interface.
func (s *scheduler) Start(ctx context.Context) error {
	// Fail-fast: context cannot be nil
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("scheduler is already started")
	}

	if s.stopped {
		return fmt.Errorf("scheduler is stopped")
	}

	s.started = true

	// Start the scheduler loop
	s.wg.Add(1)
	go s.run(ctx)

	return nil
}

// Stop implements Scheduler interface.
func (s *scheduler) Stop() error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true
	s.mu.Unlock()

	// Cancel context to stop the scheduler loop
	s.cancel()

	// Wait for all goroutines to finish
	s.wg.Wait()

	return nil
}

// List implements Scheduler interface.
func (s *scheduler) List() []ScheduledTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ScheduledTask, 0, len(s.tasks))
	for _, entry := range s.tasks {
		entry.mu.Lock()
		result = append(result, ScheduledTask{
			ID:       entry.id,
			Task:     entry.task,
			Spec:     entry.spec,
			NextRun:  entry.nextRun,
			LastRun:  entry.lastRun,
			RunCount: entry.runCount,
		})
		entry.mu.Unlock()
	}

	return result
}

// run is the main scheduler loop that executes tasks at their scheduled times.
func (s *scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.ctx.Done():
			return
		case now := <-ticker.C:
			s.executeDueTasks(now)
		}
	}
}

// executeDueTasks executes all tasks that are due at the current time.
func (s *scheduler) executeDueTasks(now time.Time) {
	s.mu.RLock()
	entries := make([]*scheduledTaskEntry, 0, len(s.tasks))
	for _, entry := range s.tasks {
		entries = append(entries, entry)
	}
	s.mu.RUnlock()

	for _, entry := range entries {
		entry.mu.Lock()
		due := !entry.nextRun.IsZero() && (now.After(entry.nextRun) || now.Equal(entry.nextRun))
		if due {
			// Execute task in a goroutine to avoid blocking
			entry.mu.Unlock()
			s.wg.Add(1)
			go s.executeTask(entry, now)
		} else {
			entry.mu.Unlock()
		}
	}
}

// executeTask executes a single task and schedules the next run.
func (s *scheduler) executeTask(entry *scheduledTaskEntry, now time.Time) {
	defer s.wg.Done()

	entry.mu.Lock()
	entry.lastRun = now
	entry.runCount++
	entry.mu.Unlock()

	// Execute the task
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	err := entry.task.Execute(ctx)
	if err != nil {
		// Log error but don't stop scheduling
		// In a real implementation, you might want to use a logger here
		_ = err
	}

	// Calculate next run time
	entry.mu.Lock()
	if entry.spec.IsRecurring() {
		entry.nextRun = entry.spec.Next(now)
	} else {
		// One-time task, mark as done
		entry.nextRun = time.Time{}
	}
	entry.mu.Unlock()

	// If task is not recurring and has completed, remove it
	if !entry.spec.IsRecurring() {
		s.mu.Lock()
		delete(s.tasks, entry.id)
		s.mu.Unlock()
	}
}
