package kernel

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type memComp struct {
	name  string
	fail  bool
	stopC int32
}

func (m *memComp) Name() string { return m.name }

func (m *memComp) Start(ctx context.Context) error {
	if m.fail {
		return errors.New("start fail")
	}
	return nil
}

func (m *memComp) Stop(ctx context.Context) error {
	atomic.AddInt32(&m.stopC, 1)
	return nil
}

type compWrap struct {
	name        string
	onStart     func()
	onStop      func()
}

func (c *compWrap) Name() string { return c.name }
func (c *compWrap) Start(ctx context.Context) error {
	if c.onStart != nil {
		c.onStart()
	}
	return nil
}
func (c *compWrap) Stop(ctx context.Context) error {
	if c.onStop != nil {
		c.onStop()
	}
	return nil
}

func TestKernel_StartStop_Order(t *testing.T) {
	var order []string
	k := NewKernel(0)
	_ = k.Register(&compWrap{name: "a", onStart: func() { order = append(order, "a-start") }, onStop: func() { order = append(order, "a-stop") }})
	_ = k.Register(&compWrap{name: "b", onStart: func() { order = append(order, "b-start") }, onStop: func() { order = append(order, "b-stop") }})

	if err := k.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := k.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{"a-start", "b-start", "b-stop", "a-stop"}
	if len(order) != len(want) {
		t.Fatalf("got %v want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("at %d: got %v", i, order)
		}
	}
}

func TestKernel_StartRollback(t *testing.T) {
	k := NewKernel(0)
	first := &memComp{name: "ok"}
	second := &memComp{name: "bad", fail: true}
	_ = k.Register(first)
	_ = k.Register(second)

	if err := k.Start(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if k.State() != StateStopped {
		t.Fatalf("state=%v", k.State())
	}
	if atomic.LoadInt32(&first.stopC) != 1 {
		t.Fatalf("first component should be stopped on rollback, got %d", first.stopC)
	}
}

func TestChain_MiddlewareOrder(t *testing.T) {
	var s string
	h := func(ctx AppContext) error { s += "H"; return nil }
	mw := Chain(
		func(next Handler) Handler {
			return func(ctx AppContext) error { s += "A"; return next(ctx) }
		},
		func(next Handler) Handler {
			return func(ctx AppContext) error { s += "B"; return next(ctx) }
		},
	)
	_ = mw(h)(NewAppContext(context.Background(), Meta{}))
	if s != "ABH" {
		t.Fatalf("got %q want ABH", s)
	}
}
