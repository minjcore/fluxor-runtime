package mesh

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestMeshCall(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	eb := gocmd.EventBus()
	mesh := NewServiceMesh(eb)

	// 1. Register a service consumer
	serviceName := "test-service"
	action := "echo"
	address := serviceName + "." + action

	eb.Consumer(address).Handler(func(ctx core.FluxorContext, msg core.Message) error {
		body := msg.Body()
		return msg.Reply(body)
	})

	// 2. Call via mesh
	payload := map[string]string{"hello": "world"}
	resp, err := mesh.Call(ctx, serviceName, action, payload, CallOptions{})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	var result map[string]string
	if err := resp.DecodeBody(&result); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if result["hello"] != "world" {
		t.Errorf("Expected 'world', got %s", result["hello"])
	}
}

func TestCircuitBreaker(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	eb := gocmd.EventBus()
	mesh := NewServiceMesh(eb)

	serviceName := "failing-service"
	action := "fail"
	address := serviceName + "." + action

	// Failing consumer
	eb.Consumer(address).Handler(func(ctx core.FluxorContext, msg core.Message) error {
		return errors.New("simulated failure")
	})

	// Configure short reset timeout for testing
	cb := mesh.(*meshImpl).getCircuitBreaker(serviceName)
	cb.threshold = 2
	cb.resetTimeout = 100 * time.Millisecond

	opts := CallOptions{
		RetryPolicy: &RetryPolicy{MaxRetries: 0}, // No retries to fail fast
	}

	// 1. Fail twice to open breaker
	payload := map[string]string{}
	_, err := mesh.Call(ctx, serviceName, action, payload, opts)
	if err == nil {
		t.Error("Expected failure")
	}
	_, err = mesh.Call(ctx, serviceName, action, payload, opts)
	if err == nil {
		t.Error("Expected failure")
	}

	// 2. Next call should be rejected by circuit breaker immediately
	// Note: The mock EventBus returns error immediately, so we need to verify if it was the CB or the call
	// We can check the error message
	_, err = mesh.Call(ctx, serviceName, action, payload, opts)
	if err == nil || err.Error() != fmt.Sprintf("circuit breaker open for service %s", serviceName) {
		t.Errorf("Expected circuit breaker error, got: %v", err)
	}

	// 3. Wait for reset
	time.Sleep(150 * time.Millisecond)

	// 4. Next call should try (half-open)
	// It will fail again because consumer still fails
	_, err = mesh.Call(ctx, serviceName, action, payload, opts)
	if err == nil {
		t.Error("Expected failure")
	}

	// Should be open again
	if cb.state != StateOpen {
		t.Errorf("Expected StateOpen, got %v", cb.state)
	}
}

func TestRetries(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	eb := gocmd.EventBus()
	mesh := NewServiceMesh(eb)

	serviceName := "retry-service"
	action := "flaky"
	address := serviceName + "." + action

	var attempts int32

	// Flaky consumer: fails first 2 times, succeeds 3rd
	eb.Consumer(address).Handler(func(ctx core.FluxorContext, msg core.Message) error {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			return errors.New("flaky error")
		}
		return msg.Reply("success")
	})

	opts := CallOptions{
		RetryPolicy: &RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     10 * time.Millisecond,
		},
	}

	payload := map[string]string{"foo": "bar"}
	resp, err := mesh.Call(ctx, serviceName, action, payload, opts)
	if err != nil {
		t.Fatalf("Call failed after retries: %v", err)
	}

	var res string
	_ = resp.DecodeBody(&res)
	if res != "success" {
		t.Errorf("Expected success, got %s", res)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}
