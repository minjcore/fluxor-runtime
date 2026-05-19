// Package scheduler provides task scheduling capabilities for the Fluxor framework.
// It supports cron-style scheduling, periodic execution, and delay-based scheduling.
//
// Example:
//
//	scheduler := scheduler.NewScheduler(ctx)
//	task := &MyTask{name: "daily-cleanup"}
//	id, err := scheduler.Schedule(task, scheduler.Cron("0 0 * * *"))
//	err = scheduler.Start(ctx)
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
)

// Scheduler provides task scheduling capabilities.
// It allows scheduling tasks with various schedule specifications including
// cron expressions, fixed intervals, and one-time delays.
type Scheduler interface {
	// Schedule schedules a task with the given schedule specification.
	// Returns a unique ID for the scheduled task that can be used to unschedule it.
	//
	// Example:
	//   id, err := scheduler.Schedule(task, scheduler.Cron("0 0 * * *"))
	Schedule(task concurrency.Task, spec ScheduleSpec) (string, error)

	// Unschedule removes a scheduled task by its ID.
	// Returns an error if the task ID is not found.
	//
	// Example:
	//   err := scheduler.Unschedule(id)
	Unschedule(id string) error

	// Start starts the scheduler and begins executing scheduled tasks.
	// This method should be called after scheduling tasks.
	//
	// Example:
	//   err := scheduler.Start(ctx)
	Start(ctx context.Context) error

	// Stop stops the scheduler and cancels all scheduled tasks.
	// This method should be called for graceful shutdown.
	//
	// Example:
	//   err := scheduler.Stop()
	Stop() error

	// List returns all currently scheduled tasks.
	//
	// Example:
	//   tasks := scheduler.List()
	List() []ScheduledTask
}

// ScheduleSpec represents a schedule specification for a task.
// It can be one of: CronSpec, IntervalSpec, or DelaySpec.
type ScheduleSpec interface {
	// Next returns the next execution time from the given time.
	// Returns zero time if there are no more executions.
	Next(from time.Time) time.Time

	// IsRecurring returns true if this schedule repeats, false for one-time execution.
	IsRecurring() bool
}

// CronSpec represents a cron-style schedule specification.
// Cron expressions use the standard 5-field format:
//
//	minute hour day month weekday
//
// Example: "0 0 * * *" (daily at midnight)
type CronSpec struct {
	expression string
	cron       *cronSchedule
}

// Cron creates a new CronSpec from a cron expression string.
// The expression must be in the format: "minute hour day month weekday"
// where:
//   - minute: 0-59
//   - hour: 0-23
//   - day: 1-31
//   - month: 1-12 or JAN-DEC
//   - weekday: 0-7 (0 and 7 are Sunday) or SUN-SAT
//
// Special characters:
//   - * (any value)
//   - , (value list separator)
//   - - (range)
//   - / (step values)
//
// Example:
//
//	spec := scheduler.Cron("0 0 * * *")  // Daily at midnight
//	spec := scheduler.Cron("*/5 * * * *") // Every 5 minutes
func Cron(expression string) ScheduleSpec {
	// Fail-fast: expression cannot be empty
	if expression == "" {
		panic(fmt.Errorf("fail-fast: cron expression cannot be empty"))
	}

	cron, err := parseCron(expression)
	if err != nil {
		panic(fmt.Errorf("fail-fast: invalid cron expression %q: %w", expression, err))
	}

	return &CronSpec{
		expression: expression,
		cron:       cron,
	}
}

// Next implements ScheduleSpec interface.
func (c *CronSpec) Next(from time.Time) time.Time {
	if c.cron == nil {
		return time.Time{}
	}
	return c.cron.next(from)
}

// IsRecurring implements ScheduleSpec interface.
func (c *CronSpec) IsRecurring() bool {
	return true
}

// String returns the cron expression string.
func (c *CronSpec) String() string {
	return c.expression
}

// IntervalSpec represents a fixed interval schedule specification.
// Tasks will execute at regular intervals.
type IntervalSpec struct {
	interval time.Duration
}

// Interval creates a new IntervalSpec that executes tasks at the given interval.
// The interval must be positive.
//
// Example:
//
//	spec := scheduler.Interval(5 * time.Minute)  // Every 5 minutes
//	spec := scheduler.Interval(1 * time.Hour)     // Every hour
func Interval(interval time.Duration) ScheduleSpec {
	// Fail-fast: interval must be positive
	if interval <= 0 {
		panic(fmt.Errorf("fail-fast: interval must be positive, got %v", interval))
	}

	return &IntervalSpec{
		interval: interval,
	}
}

// Next implements ScheduleSpec interface.
func (i *IntervalSpec) Next(from time.Time) time.Time {
	return from.Add(i.interval)
}

// IsRecurring implements ScheduleSpec interface.
func (i *IntervalSpec) IsRecurring() bool {
	return true
}

// Interval returns the interval duration.
func (i *IntervalSpec) Interval() time.Duration {
	return i.interval
}

// DelaySpec represents a one-time delay schedule specification.
// Tasks will execute once after the specified delay.
type DelaySpec struct {
	delay  time.Duration
	used   bool
	usedMu sync.Mutex
}

// Delay creates a new DelaySpec that executes a task once after the given delay.
// The delay must be positive.
//
// Example:
//
//	spec := scheduler.Delay(1 * time.Hour)  // Execute after 1 hour
//	spec := scheduler.Delay(30 * time.Minute) // Execute after 30 minutes
func Delay(delay time.Duration) ScheduleSpec {
	// Fail-fast: delay must be positive
	if delay <= 0 {
		panic(fmt.Errorf("fail-fast: delay must be positive, got %v", delay))
	}

	return &DelaySpec{
		delay: delay,
		used:  false,
	}
}

// Next implements ScheduleSpec interface.
func (d *DelaySpec) Next(from time.Time) time.Time {
	d.usedMu.Lock()
	defer d.usedMu.Unlock()

	// For delay, we only return the next time once
	// After that, return zero time to indicate no more executions
	if !d.used {
		d.used = true
		return from.Add(d.delay)
	}
	return time.Time{}
}

// IsRecurring implements ScheduleSpec interface.
func (d *DelaySpec) IsRecurring() bool {
	return false
}

// Delay returns the delay duration.
func (d *DelaySpec) Delay() time.Duration {
	return d.delay
}

// ScheduledTask represents a scheduled task with its metadata.
type ScheduledTask struct {
	// ID is the unique identifier for this scheduled task.
	ID string

	// Task is the task to be executed.
	Task concurrency.Task

	// Spec is the schedule specification.
	Spec ScheduleSpec

	// NextRun is the next scheduled execution time.
	NextRun time.Time

	// LastRun is the last execution time, or zero if not yet executed.
	LastRun time.Time

	// RunCount is the number of times this task has been executed.
	RunCount int64
}
