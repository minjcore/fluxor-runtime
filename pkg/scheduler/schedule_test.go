package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"
)

// testTask is a simple task implementation for testing.
type testTask struct {
	name      string
	executed  bool
	execCount int64
	mu        sync.Mutex
	err       error
}

func (t *testTask) Execute(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.executed = true
	t.execCount++
	return t.err
}

func (t *testTask) Name() string {
	return t.name
}

func (t *testTask) GetExecCount() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.execCount
}

func TestNewScheduler(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)
	if scheduler == nil {
		t.Fatal("NewScheduler returned nil")
	}
}

func TestNewScheduler_NilContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil context")
		}
	}()
	NewScheduler(nil)
}

func TestScheduler_Schedule(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "test-task"}
	spec := Interval(1 * time.Second)

	id, err := scheduler.Schedule(task, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}
	if id == "" {
		t.Error("Schedule returned empty ID")
	}
}

func TestScheduler_Schedule_NilTask(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	spec := Interval(1 * time.Second)
	_, err := scheduler.Schedule(nil, spec)
	if err == nil {
		t.Error("expected error for nil task")
	}
}

func TestScheduler_Schedule_NilSpec(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "test-task"}
	_, err := scheduler.Schedule(task, nil)
	if err == nil {
		t.Error("expected error for nil spec")
	}
}

func TestScheduler_Unschedule(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "test-task"}
	spec := Interval(1 * time.Second)

	id, err := scheduler.Schedule(task, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	err = scheduler.Unschedule(id)
	if err != nil {
		t.Fatalf("Unschedule failed: %v", err)
	}
}

func TestScheduler_Unschedule_NotFound(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	err := scheduler.Unschedule("non-existent-id")
	if err == nil {
		t.Error("expected error for non-existent task ID")
	}
}

func TestScheduler_Unschedule_EmptyID(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	err := scheduler.Unschedule("")
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestScheduler_List(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task1 := &testTask{name: "task-1"}
	task2 := &testTask{name: "task-2"}

	id1, err := scheduler.Schedule(task1, Interval(1*time.Second))
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	id2, err := scheduler.Schedule(task2, Interval(2*time.Second))
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	tasks := scheduler.List()
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	ids := make(map[string]bool)
	for _, task := range tasks {
		ids[task.ID] = true
	}

	if !ids[id1] || !ids[id2] {
		t.Error("List did not return expected task IDs")
	}
}

func TestScheduler_Start(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "test-task"}
	spec := Interval(1 * time.Second)

	_, err := scheduler.Schedule(task, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for task to execute (scheduler checks every second)
	time.Sleep(2 * time.Second)

	// Check if task was executed
	if task.GetExecCount() == 0 {
		t.Error("task was not executed")
	}

	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestScheduler_Start_NilContext(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	err := scheduler.Start(nil)
	if err == nil {
		t.Error("expected error for nil context")
	}
}

func TestScheduler_Start_AlreadyStarted(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	err := scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = scheduler.Start(ctx)
	if err == nil {
		t.Error("expected error for already started scheduler")
	}

	scheduler.Stop()
}

func TestScheduler_Stop(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "test-task"}
	spec := Interval(100 * time.Millisecond)

	_, err := scheduler.Schedule(task, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Stop should be idempotent
	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("second Stop failed: %v", err)
	}
}

func TestScheduler_Schedule_AfterStop(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	err := scheduler.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	task := &testTask{name: "test-task"}
	spec := Interval(1 * time.Second)

	_, err = scheduler.Schedule(task, spec)
	if err == nil {
		t.Error("expected error when scheduling after stop")
	}
}

func TestScheduler_IntervalExecution(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "interval-task"}
	spec := Interval(1 * time.Second)

	_, err := scheduler.Schedule(task, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait deterministically for multiple executions.
	// The scheduler checks on a 1s cadence, so use a generous deadline to avoid flakiness.
	deadline := time.After(6 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		if task.GetExecCount() >= 3 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("expected at least 3 executions, got %d", task.GetExecCount())
		case <-ticker.C:
		}
	}

	scheduler.Stop()
}

func TestScheduler_DelayExecution(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "delay-task"}
	spec := Delay(1 * time.Second)

	_, err := scheduler.Schedule(task, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for task to execute (scheduler checks every second)
	time.Sleep(2 * time.Second)

	execCount := task.GetExecCount()
	if execCount != 1 {
		t.Errorf("expected 1 execution, got %d", execCount)
	}

	// Wait a bit more to ensure it doesn't execute again
	time.Sleep(2 * time.Second)

	execCount = task.GetExecCount()
	if execCount != 1 {
		t.Errorf("expected 1 execution (one-time), got %d", execCount)
	}

	scheduler.Stop()
}

func TestScheduler_CronExecution(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "cron-task"}
	// Every minute (for testing - scheduler checks every second so it will catch it)
	spec := Cron("* * * * *")

	_, err := scheduler.Schedule(task, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for task to execute (should execute at next minute boundary)
	time.Sleep(2 * time.Second)

	scheduler.Stop()
}

func TestIntervalSpec(t *testing.T) {
	spec := Interval(5 * time.Minute)
	if !spec.IsRecurring() {
		t.Error("IntervalSpec should be recurring")
	}

	now := time.Now()
	next := spec.Next(now)
	expected := now.Add(5 * time.Minute)

	if next.Sub(expected) > time.Second {
		t.Errorf("expected next time %v, got %v", expected, next)
	}
}

func TestIntervalSpec_Invalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid interval")
		}
	}()
	Interval(0)
}

func TestDelaySpec(t *testing.T) {
	spec := Delay(1 * time.Hour)
	if spec.IsRecurring() {
		t.Error("DelaySpec should not be recurring")
	}

	now := time.Now()
	next := spec.Next(now)
	expected := now.Add(1 * time.Hour)

	if next.Sub(expected) > time.Second {
		t.Errorf("expected next time %v, got %v", expected, next)
	}

	// Second call should return zero time
	next2 := spec.Next(now)
	if !next2.IsZero() {
		t.Error("DelaySpec should return zero time after first call")
	}
}

func TestDelaySpec_Invalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid delay")
		}
	}()
	Delay(0)
}

func TestCronSpec(t *testing.T) {
	spec := Cron("0 0 * * *")
	if !spec.IsRecurring() {
		t.Error("CronSpec should be recurring")
	}

	now := time.Now()
	next := spec.Next(now)
	if next.IsZero() {
		t.Error("CronSpec should return a valid next time")
	}
}

func TestCronSpec_Invalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid cron expression")
		}
	}()
	Cron("")
}

func TestCronSpec_InvalidFormat(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid cron format")
		}
	}()
	Cron("invalid")
}
