package eventbus

import (
	"github.com/fluxorio/fluxor/pkg/core"
	"context"
	"sync/atomic"
	"testing"
	"time"

	natssrv "github.com/nats-io/nats-server/v2/server"
)

func runTestNATSServer(t *testing.T) *natssrv.Server {
	t.Helper()

	opts := &natssrv.Options{
		Port: -1,
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

func TestClusterEventBusNATS_PublishSendRequest(t *testing.T) {
	s := runTestNATSServer(t)
	url := s.ClientURL()

	ctx := context.Background()

	v, err := core.NewGoCMDWithOptions(ctx, core.GoCMDOptions{
		EventBusFactory: func(ctx context.Context, gocmd core.GoCMD) (core.EventBus, error) {
			return NewClusterEventBusNATS(ctx, gocmd, ClusterNATSConfig{
				URL:            url,
				Prefix:         "fluxor.test",
				RequestTimeout: 2 * time.Second,
			})
		},
	})
	if err != nil {
		t.Fatalf("NewVertxWithOptions: %v", err)
	}
	t.Cleanup(func() {
		_ = v.Close()
	})

	bus := v.EventBus()

	// Two consumers on same address so we can verify:
	// - Publish: both receive (fanout)
	// - Send: exactly one receives (queue)
	var pubCount1 int64
	var pubCount2 int64
	var sendTotal int64
	handler := func(pubCount *int64) core.MessageHandler {
		return func(_ core.FluxorContext, msg core.Message) error {
			b, ok := msg.Body().([]byte)
			if !ok {
				t.Fatalf("expected []byte body, got %T", msg.Body())
			}

			var payload struct {
				Kind string `json:"kind"`
			}
			if err := core.JSONDecode(b, &payload); err != nil {
				t.Fatalf("core.JSONDecode: %v", err)
			}

			switch payload.Kind {
			case "pub":
				atomic.AddInt64(pubCount, 1)
			case "send":
				atomic.AddInt64(&sendTotal, 1)
			default:
				t.Fatalf("unexpected kind: %q", payload.Kind)
			}

			return nil
		}
	}

	bus.Consumer("work").Handler(func(ctx core.FluxorContext, msg core.Message) error {
		return handler(&pubCount1)(ctx, msg)
	})
	bus.Consumer("work").Handler(func(ctx core.FluxorContext, msg core.Message) error {
		return handler(&pubCount2)(ctx, msg)
	})

	// NATS subscriptions are async; give them a moment to become active.
	time.Sleep(50 * time.Millisecond)

	// Publish should hit both consumers.
	for i := 0; i < 10; i++ {
		if err := bus.Publish("work", map[string]string{"kind": "pub"}); err != nil {
			t.Fatalf("Publish: %v", err)
		}
	}

	// Send should hit exactly one consumer per message.
	for i := 0; i < 50; i++ {
		if err := bus.Send("work", map[string]string{"kind": "send"}); err != nil {
			t.Fatalf("Send: %v", err)
		}
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		pc1 := atomic.LoadInt64(&pubCount1)
		pc2 := atomic.LoadInt64(&pubCount2)
		st := atomic.LoadInt64(&sendTotal)
		if pc1+pc2 >= 20 && st >= 50 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := atomic.LoadInt64(&sendTotal); got != 50 {
		t.Fatalf("sendTotal: got %d want 50", got)
	}
	if got := atomic.LoadInt64(&pubCount1) + atomic.LoadInt64(&pubCount2); got != 20 {
		t.Fatalf("pubTotal: got %d want 20", got)
	}

	// Request/Reply.
	bus.Consumer("echo").Handler(func(_ core.FluxorContext, msg core.Message) error {
		var req struct {
			Msg string `json:"msg"`
		}
		if err := core.JSONDecode(msg.Body().([]byte), &req); err != nil {
			return err
		}
		return msg.Reply(map[string]interface{}{
			"ok":  true,
			"msg": req.Msg,
		})
	})

	reply, err := bus.Request("echo", map[string]string{"msg": "hi"}, 2*time.Second)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}

	var resp struct {
		OK  bool   `json:"ok"`
		Msg string `json:"msg"`
	}
	if err := core.JSONDecode(reply.Body().([]byte), &resp); err != nil {
		t.Fatalf("core.JSONDecode reply: %v", err)
	}
	if !resp.OK || resp.Msg != "hi" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestNewClusterEventBusNATS_FailFast_InvalidInputs(t *testing.T) {
	s := runTestNATSServer(t)
	url := s.ClientURL()

	v := core.NewGoCMD(context.Background())
	defer func() { _ = v.Close() }()

	t.Run("nil ctx", func(t *testing.T) {
		if _, err := NewClusterEventBusNATS(nil, v, ClusterNATSConfig{URL: url}); err == nil {
			t.Fatalf("expected error for nil ctx")
		}
	})

	t.Run("nil gocmd", func(t *testing.T) {
		if _, err := NewClusterEventBusNATS(context.Background(), nil, ClusterNATSConfig{URL: url}); err == nil {
			t.Fatalf("expected error for nil gocmd")
		}
	})
}
