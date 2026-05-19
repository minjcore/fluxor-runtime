package eventbus

import (
	"github.com/fluxorio/fluxor/pkg/core"
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/proto/fluxor/common"
	natssrv "github.com/nats-io/nats-server/v2/server"
)

func runTestNATSJetStreamServer(t *testing.T) *natssrv.Server {
	t.Helper()

	opts := &natssrv.Options{
		Port:      -1,
		JetStream: true,
		StoreDir:  t.TempDir(),
	}
	s, err := natssrv.NewServer(opts)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	go s.Start()
	if !s.ReadyForConnections(5 * time.Second) {
		s.Shutdown()
		t.Fatalf("nats server not ready")
	}
	t.Cleanup(func() {
		s.Shutdown()
	})
	return s
}

func TestClusterEventBusJetStream_PublishAndSend(t *testing.T) {
	s := runTestNATSJetStreamServer(t)
	url := s.ClientURL()

	ctx := context.Background()

	// Two services consuming the same "topic" should both receive Publish once.
	vA := core.NewGoCMD(ctx)
	vB := core.NewGoCMD(ctx)
	defer func() { _ = vA.Close() }()
	defer func() { _ = vB.Close() }()

	busA, err := NewClusterEventBusJetStream(ctx, vA, ClusterJetStreamConfig{
		URL:     url,
		Prefix:  "fluxor.js.test",
		Service: "api-gateway",
	})
	if err != nil {
		t.Fatalf("NewClusterEventBusJetStream A: %v", err)
	}
	defer func() { _ = busA.Close() }()

	busB, err := NewClusterEventBusJetStream(ctx, vB, ClusterJetStreamConfig{
		URL:     url,
		Prefix:  "fluxor.js.test",
		Service: "payment-service",
	})
	if err != nil {
		t.Fatalf("NewClusterEventBusJetStream B: %v", err)
	}
	defer func() { _ = busB.Close() }()

	var gotA int64
	var gotB int64

	busA.Consumer("topic").Handler(func(_ core.FluxorContext, msg core.Message) error {
		atomic.AddInt64(&gotA, 1)
		return nil
	})
	busB.Consumer("topic").Handler(func(_ core.FluxorContext, msg core.Message) error {
		atomic.AddInt64(&gotB, 1)
		return nil
	})

	time.Sleep(50 * time.Millisecond)

	if err := busA.Publish("topic", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&gotA) >= 1 && atomic.LoadInt64(&gotB) >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := atomic.LoadInt64(&gotA); got != 1 {
		t.Fatalf("service A publish count: got %d want 1", got)
	}
	if got := atomic.LoadInt64(&gotB); got != 1 {
		t.Fatalf("service B publish count: got %d want 1", got)
	}

	// Send should be load-balanced (delivered once) across two consumers.
	vW1 := core.NewGoCMD(ctx)
	vW2 := core.NewGoCMD(ctx)
	defer func() { _ = vW1.Close() }()
	defer func() { _ = vW2.Close() }()

	w1, err := NewClusterEventBusJetStream(ctx, vW1, ClusterJetStreamConfig{
		URL:     url,
		Prefix:  "fluxor.js.test",
		Service: "worker-1",
	})
	if err != nil {
		t.Fatalf("NewClusterEventBusJetStream W1: %v", err)
	}
	defer func() { _ = w1.Close() }()

	w2, err := NewClusterEventBusJetStream(ctx, vW2, ClusterJetStreamConfig{
		URL:     url,
		Prefix:  "fluxor.js.test",
		Service: "worker-2",
	})
	if err != nil {
		t.Fatalf("NewClusterEventBusJetStream W2: %v", err)
	}
	defer func() { _ = w2.Close() }()

	var w1Count int64
	var w2Count int64

	w1.Consumer("work").Handler(func(_ core.FluxorContext, msg core.Message) error {
		atomic.AddInt64(&w1Count, 1)
		return nil
	})
	w2.Consumer("work").Handler(func(_ core.FluxorContext, msg core.Message) error {
		atomic.AddInt64(&w2Count, 1)
		return nil
	})

	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 50; i++ {
		if err := busA.Send("work", map[string]any{"n": i}); err != nil {
			t.Fatalf("Send: %v", err)
		}
	}

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		total := atomic.LoadInt64(&w1Count) + atomic.LoadInt64(&w2Count)
		if total >= 50 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if total := atomic.LoadInt64(&w1Count) + atomic.LoadInt64(&w2Count); total != 50 {
		t.Fatalf("send total: got %d want 50 (w1=%d w2=%d)", total, atomic.LoadInt64(&w1Count), atomic.LoadInt64(&w2Count))
	}
	// Not strictly required, but distribution should not be pathological.
	if atomic.LoadInt64(&w1Count) == 0 || atomic.LoadInt64(&w2Count) == 0 {
		t.Fatalf("expected send distribution across workers, got w1=%d w2=%d", atomic.LoadInt64(&w1Count), atomic.LoadInt64(&w2Count))
	}
}

func TestNewClusterEventBusJetStream_FailFast_InvalidInputs(t *testing.T) {
	s := runTestNATSJetStreamServer(t)
	url := s.ClientURL()

	t.Run("nil ctx", func(t *testing.T) {
		v := core.NewGoCMD(context.Background())
		defer func() { _ = v.Close() }()

		if _, err := NewClusterEventBusJetStream(nil, v, ClusterJetStreamConfig{
			URL:     url,
			Prefix:  "fluxor.js.failfast",
			Service: "svc",
		}); err == nil {
			t.Fatalf("expected error for nil ctx")
		}
	})

	t.Run("nil gocmd", func(t *testing.T) {
		if _, err := NewClusterEventBusJetStream(context.Background(), nil, ClusterJetStreamConfig{
			URL:     url,
			Prefix:  "fluxor.js.failfast",
			Service: "svc",
		}); err == nil {
			t.Fatalf("expected error for nil gocmd")
		}
	})

	t.Run("missing service", func(t *testing.T) {
		v := core.NewGoCMD(context.Background())
		defer func() { _ = v.Close() }()

		if _, err := NewClusterEventBusJetStream(context.Background(), v, ClusterJetStreamConfig{
			URL:    url,
			Prefix: "fluxor.js.failfast",
			// Service intentionally missing
		}); err == nil {
			t.Fatalf("expected error for missing service")
		}
	})
}

func TestClusterEventBusJetStream_Request(t *testing.T) {
	s := runTestNATSJetStreamServer(t)
	url := s.ClientURL()

	ctx := context.Background()
	v := core.NewGoCMD(ctx)
	defer func() { _ = v.Close() }()

	bus, err := NewClusterEventBusJetStream(ctx, v, ClusterJetStreamConfig{
		URL:     url,
		Prefix:  "fluxor.js.request",
		Service: "test-service",
	})
	if err != nil {
		t.Fatalf("NewClusterEventBusJetStream: %v", err)
	}
	defer func() { _ = bus.Close() }()

	// Test fail-fast: invalid timeout
	_, err = bus.Request("test.address", "test", 0)
	if err == nil {
		t.Error("Request() with zero timeout should fail")
	}

	// Test fail-fast: empty address
	_, err = bus.Request("", "test", 1*time.Second)
	if err == nil {
		t.Error("Request() with empty address should fail")
	}

	// Test fail-fast: nil body
	_, err = bus.Request("test.address", nil, 1*time.Second)
	if err == nil {
		t.Error("Request() with nil body should fail")
	}

	// Test fail-fast: no handlers (timeout expected)
	_, err = bus.Request("no.handlers", "test", 100*time.Millisecond)
	if err == nil {
		t.Error("Request() with no handlers should fail")
	}

	// Setup handler that replies
	bus.Consumer("echo").Handler(func(_ core.FluxorContext, msg core.Message) error {
		var req map[string]interface{}
		if err := msg.DecodeBody(&req); err != nil {
			return err
		}
		return msg.Reply(map[string]interface{}{
			"ok":  true,
			"msg": req["msg"],
		})
	})

	// Wait for subscription to be ready
	time.Sleep(50 * time.Millisecond)

	// Test valid request
	reply, err := bus.Request("echo", map[string]string{"msg": "hello"}, 2*time.Second)
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}

	if reply == nil {
		t.Fatal("Request() returned nil message")
	}

	// Decode reply
	var resp map[string]interface{}
	if err := reply.DecodeBody(&resp); err != nil {
		t.Fatalf("DecodeBody() error = %v", err)
	}

	if resp["ok"] != true {
		t.Errorf("Request() reply ok = %v, want true", resp["ok"])
	}
	if resp["msg"] != "hello" {
		t.Errorf("Request() reply msg = %q, want %q", resp["msg"], "hello")
	}
}

func TestClusterEventBusJetStream_Request_Protobuf(t *testing.T) {
	s := runTestNATSJetStreamServer(t)
	url := s.ClientURL()

	ctx := context.Background()
	v := core.NewGoCMD(ctx)
	defer func() { _ = v.Close() }()

	bus, err := NewClusterEventBusJetStream(ctx, v, ClusterJetStreamConfig{
		URL:     url,
		Prefix:  "fluxor.js.request.pb",
		Service: "test-service",
	})
	if err != nil {
		t.Fatalf("NewClusterEventBusJetStream: %v", err)
	}
	defer func() { _ = bus.Close() }()

	// Setup handler that replies with protobuf
	bus.Consumer("echo.pb").Handler(func(_ core.FluxorContext, msg core.Message) error {
		var user common.User
		if err := msg.DecodeBody(&user); err != nil {
			return err
		}
		// Reply with modified user
		replyUser := &common.User{
			Id:        user.GetId(),
			Name:      "Reply: " + user.GetName(),
			Email:     user.GetEmail(),
			CreatedAt: user.GetCreatedAt(),
			Active:    user.GetActive(),
		}
		return msg.Reply(replyUser)
	})

	// Wait for subscription to be ready
	time.Sleep(50 * time.Millisecond)

	// Send protobuf request
	requestUser := &common.User{
		Id:        "123",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: 1234567890,
		Active:    true,
	}

	reply, err := bus.Request("echo.pb", requestUser, 2*time.Second)
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}

	if reply == nil {
		t.Fatal("Request() returned nil message")
	}

	// Decode protobuf reply
	var replyUser common.User
	if err := reply.DecodeBody(&replyUser); err != nil {
		t.Fatalf("DecodeBody() error = %v", err)
	}

	if replyUser.GetId() != "123" {
		t.Errorf("Request() reply id = %q, want %q", replyUser.GetId(), "123")
	}
	if replyUser.GetName() != "Reply: John Doe" {
		t.Errorf("Request() reply name = %q, want %q", replyUser.GetName(), "Reply: John Doe")
	}
}
