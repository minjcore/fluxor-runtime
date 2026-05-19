package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestFutureT_Await(t *testing.T) {
	promise := NewPromiseT[string]()

	// Complete asynchronously
	go func() {
		time.Sleep(10 * time.Millisecond)
		promise.Complete("test-result")
	}()

	ctx := context.Background()
	result, err := promise.Await(ctx)

	if err != nil {
		t.Fatalf("Await() error = %v, want nil", err)
	}
	if result != "test-result" {
		t.Errorf("Await() = %v, want test-result", result)
	}
}

func TestFutureT_Await_Error(t *testing.T) {
	promise := NewPromiseT[string]()

	// Fail asynchronously
	go func() {
		time.Sleep(10 * time.Millisecond)
		promise.Fail(errors.New("test error"))
	}()

	ctx := context.Background()
	result, err := promise.Await(ctx)

	if err == nil {
		t.Error("Await() error = nil, want error")
	}
	if result != "" {
		t.Errorf("Await() = %v, want empty string", result)
	}
}

func TestFutureT_Await_ContextCancel(t *testing.T) {
	promise := NewPromiseT[string]()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Don't complete the promise - let context timeout
	result, err := promise.Await(ctx)

	if err == nil {
		t.Error("Await() error = nil, want context error")
	}
	if result != "" {
		t.Errorf("Await() = %v, want empty string", result)
	}
}

func TestThen(t *testing.T) {
	promise := NewPromiseT[int]()

	go func() {
		time.Sleep(10 * time.Millisecond)
		promise.Complete(10)
	}()

	ctx := context.Background()

	// Transform int to string
	transformed := Then(promise, func(n int) (string, error) {
		return fmt.Sprintf("value: %d", n), nil
	})

	result, err := transformed.Await(ctx)
	if err != nil {
		t.Fatalf("Then() error = %v, want nil", err)
	}
	if result != "value: 10" {
		t.Errorf("Then() = %v, want value: 10", result)
	}
}

func TestCatch(t *testing.T) {
	promise := NewPromiseT[string]()

	go func() {
		time.Sleep(10 * time.Millisecond)
		promise.Fail(errors.New("original error"))
	}()

	ctx := context.Background()

	// Recover from error
	recovered := Catch(promise, func(err error) (string, error) {
		return "recovered: " + err.Error(), nil
	})

	result, err := recovered.Await(ctx)
	if err != nil {
		t.Fatalf("Catch() error = %v, want nil", err)
	}
	if result != "recovered: original error" {
		t.Errorf("Catch() = %v, want recovered: original error", result)
	}
}

func TestMap(t *testing.T) {
	promise := NewPromiseT[int]()

	go func() {
		time.Sleep(10 * time.Millisecond)
		promise.Complete(5)
	}()

	ctx := context.Background()

	// Map int to int (multiply by 2)
	mapped := Map(promise, func(n int) int {
		return n * 2
	})

	result, err := mapped.Await(ctx)
	if err != nil {
		t.Fatalf("Map() error = %v, want nil", err)
	}
	if result != 10 {
		t.Errorf("Map() = %v, want 10", result)
	}
}

func TestAll(t *testing.T) {
	p1 := NewPromiseT[int]()
	p2 := NewPromiseT[int]()
	p3 := NewPromiseT[int]()

	go func() {
		time.Sleep(10 * time.Millisecond)
		p1.Complete(1)
	}()
	go func() {
		time.Sleep(20 * time.Millisecond)
		p2.Complete(2)
	}()
	go func() {
		time.Sleep(30 * time.Millisecond)
		p3.Complete(3)
	}()

	ctx := context.Background()
	all := All(ctx, p1, p2, p3)

	results, err := all.Await(ctx)
	if err != nil {
		t.Fatalf("All() error = %v, want nil", err)
	}
	if len(results) != 3 {
		t.Fatalf("All() len = %v, want 3", len(results))
	}
	if results[0] != 1 || results[1] != 2 || results[2] != 3 {
		t.Errorf("All() = %v, want [1 2 3]", results)
	}
}

func TestRace(t *testing.T) {
	p1 := NewPromiseT[string]()
	p2 := NewPromiseT[string]()
	p3 := NewPromiseT[string]()

	go func() {
		time.Sleep(100 * time.Millisecond)
		p1.Complete("first")
	}()
	go func() {
		time.Sleep(50 * time.Millisecond) // This should win
		p2.Complete("second")
	}()
	go func() {
		time.Sleep(150 * time.Millisecond)
		p3.Complete("third")
	}()

	ctx := context.Background()
	race := Race(ctx, p1, p2, p3)

	result, err := race.Await(ctx)
	if err != nil {
		t.Fatalf("Race() error = %v, want nil", err)
	}
	if result != "second" {
		t.Errorf("Race() = %v, want second", result)
	}
}
