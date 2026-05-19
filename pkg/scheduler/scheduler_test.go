package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// errorTask is a task that always returns an error.
type errorTask struct {
	name     string
	executed bool
	mu       sync.Mutex
}

func (t *errorTask) Execute(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.executed = true
	return errors.New("task execution error")
}

func (t *errorTask) Name() string {
	return t.name
}

func (t *errorTask) IsExecuted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.executed
}

func TestScheduler_List_Empty(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	tasks := scheduler.List()
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestScheduler_List_AfterUnschedule(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "test-task"}
	spec := Interval(1 * time.Second)

	id, err := scheduler.Schedule(task, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	tasks := scheduler.List()
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	err = scheduler.Unschedule(id)
	if err != nil {
		t.Fatalf("Unschedule failed: %v", err)
	}

	tasks = scheduler.List()
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks after unschedule, got %d", len(tasks))
	}
}

func TestScheduler_Start_AfterStop(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	err := scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Try to start after stop - should fail
	err = scheduler.Start(ctx)
	if err == nil {
		t.Error("expected error when starting after stop")
	}
}

func TestScheduler_Unschedule_AfterStop(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "test-task"}
	spec := Interval(1 * time.Second)

	id, err := scheduler.Schedule(task, spec)
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

	// Unschedule should still work after stop (tasks are still in map)
	err = scheduler.Unschedule(id)
	if err != nil {
		t.Errorf("Unschedule after stop should still work, got error: %v", err)
	}
}

func TestScheduler_List_AfterStop(t *testing.T) {
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

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// List should still work after stop
	tasks := scheduler.List()
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks after stop, got %d", len(tasks))
	}

	ids := make(map[string]bool)
	for _, task := range tasks {
		ids[task.ID] = true
	}

	if !ids[id1] || !ids[id2] {
		t.Error("List did not return expected task IDs after stop")
	}
}

func TestScheduler_TaskExecution_Error(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &errorTask{name: "error-task"}
	spec := Interval(100 * time.Millisecond)

	_, err := scheduler.Schedule(task, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for task to execute (scheduler checks every second, but we use 100ms interval)
	// Actually, scheduler checks every 1 second, so we need to wait at least 1 second
	time.Sleep(1500 * time.Millisecond)

	// Task should still be executed despite error
	if !task.IsExecuted() {
		t.Error("task should be executed even if it returns an error")
	}

	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestScheduler_OneTimeTask_Removal(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "one-time-task"}
	spec := Delay(500 * time.Millisecond)

	id, err := scheduler.Schedule(task, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for task to execute and be removed
	time.Sleep(2 * time.Second)

	// Task should be executed
	if task.GetExecCount() == 0 {
		t.Error("one-time task should be executed")
	}

	// Task should be removed from scheduler (one-time tasks are removed after execution)
	tasks := scheduler.List()
	for _, task := range tasks {
		if task.ID == id {
			t.Error("one-time task should be removed after execution")
		}
	}

	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestScheduler_MultipleTasks(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task1 := &testTask{name: "task-1"}
	task2 := &testTask{name: "task-2"}
	task3 := &testTask{name: "task-3"}

	id1, err := scheduler.Schedule(task1, Interval(100*time.Millisecond))
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	id2, err := scheduler.Schedule(task2, Interval(200*time.Millisecond))
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	id3, err := scheduler.Schedule(task3, Delay(500*time.Millisecond))
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	// Verify all tasks are scheduled
	tasks := scheduler.List()
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}

	ids := make(map[string]bool)
	for _, task := range tasks {
		ids[task.ID] = true
	}

	if !ids[id1] || !ids[id2] || !ids[id3] {
		t.Error("List did not return all expected task IDs")
	}

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for tasks to execute (scheduler checks every second)
	time.Sleep(2 * time.Second)

	// Task1 and Task2 should have executed multiple times (recurring)
	if task1.GetExecCount() == 0 {
		t.Error("task1 should be executed")
	}
	if task2.GetExecCount() == 0 {
		t.Error("task2 should be executed")
	}

	// Task3 should be executed once (one-time)
	if task3.GetExecCount() != 1 {
		t.Errorf("task3 (one-time) should be executed once, got %d", task3.GetExecCount())
	}

	// Task3 should be removed (one-time task)
	tasks = scheduler.List()
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks after one-time task execution, got %d", len(tasks))
	}

	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestScheduler_Concurrent_Schedule(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	var wg sync.WaitGroup
	numTasks := 10
	errors := make(chan error, numTasks)

	// Schedule tasks concurrently
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			task := &testTask{name: "concurrent-task"}
			spec := Interval(1 * time.Second)
			_, err := scheduler.Schedule(task, spec)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Schedule error: %v", err)
	}

	// Verify all tasks were scheduled
	tasks := scheduler.List()
	if len(tasks) != numTasks {
		t.Errorf("expected %d tasks, got %d", numTasks, len(tasks))
	}
}

func TestScheduler_Concurrent_Unschedule(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	// Schedule tasks first
	numTasks := 10
	ids := make([]string, numTasks)
	for i := 0; i < numTasks; i++ {
		task := &testTask{name: "task"}
		spec := Interval(1 * time.Second)
		id, err := scheduler.Schedule(task, spec)
		if err != nil {
			t.Fatalf("Schedule failed: %v", err)
		}
		ids[i] = id
	}

	// Unschedule tasks concurrently
	var wg sync.WaitGroup
	errors := make(chan error, numTasks)

	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			err := scheduler.Unschedule(id)
			if err != nil {
				errors <- err
			}
		}(id)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Unschedule error: %v", err)
	}

	// Verify all tasks were unscheduled
	tasks := scheduler.List()
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks after concurrent unschedule, got %d", len(tasks))
	}
}

func TestScheduler_Concurrent_List(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	// Schedule some tasks
	for i := 0; i < 5; i++ {
		task := &testTask{name: "task"}
		spec := Interval(1 * time.Second)
		_, err := scheduler.Schedule(task, spec)
		if err != nil {
			t.Fatalf("Schedule failed: %v", err)
		}
	}

	// Call List concurrently
	var wg sync.WaitGroup
	numCalls := 20
	results := make(chan int, numCalls)

	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tasks := scheduler.List()
			results <- len(tasks)
		}()
	}

	wg.Wait()
	close(results)

	// All List calls should return the same number of tasks
	expectedCount := 5
	for count := range results {
		if count != expectedCount {
			t.Errorf("expected %d tasks, got %d", expectedCount, count)
		}
	}
}

func TestScheduler_List_TaskDetails(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	task := &testTask{name: "test-task"}
	spec := Interval(1 * time.Second)

	id, err := scheduler.Schedule(task, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	tasks := scheduler.List()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	scheduledTask := tasks[0]
	if scheduledTask.ID != id {
		t.Errorf("expected ID %q, got %q", id, scheduledTask.ID)
	}

	if scheduledTask.Task != task {
		t.Error("Task reference should match")
	}

	if scheduledTask.Spec != spec {
		t.Error("Spec reference should match")
	}

	if scheduledTask.RunCount != 0 {
		t.Errorf("expected RunCount 0, got %d", scheduledTask.RunCount)
	}

	if !scheduledTask.LastRun.IsZero() {
		t.Error("LastRun should be zero before execution")
	}

	if scheduledTask.NextRun.IsZero() {
		t.Error("NextRun should not be zero")
	}
}

func TestScheduler_Stop_DuringExecution(t *testing.T) {
	ctx := context.Background()
	scheduler := NewScheduler(ctx)

	// Create a slow task
	slowTask := &testTask{name: "slow-task"}
	spec := Interval(100 * time.Millisecond)

	_, err := scheduler.Schedule(slowTask, spec)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait a bit for scheduler to start
	time.Sleep(100 * time.Millisecond)

	// Stop scheduler (should wait for executing tasks)
	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Stop should complete (waiting for tasks)
	// Task may or may not have executed, but scheduler should stop gracefully
}
