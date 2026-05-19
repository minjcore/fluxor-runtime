package core_test

import (
	"sync"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/lite/core"
)

func TestBus_PublishSubscribe(t *testing.T) {
	bus := core.NewBus()

	var (
		mu  sync.Mutex
		got []any
		wg  sync.WaitGroup
	)

	wg.Add(2)
	unsub1 := bus.Subscribe("topic", func(msg any) {
		mu.Lock()
		got = append(got, msg)
		mu.Unlock()
		wg.Done()
	})
	unsub2 := bus.Subscribe("topic", func(msg any) {
		mu.Lock()
		got = append(got, msg)
		mu.Unlock()
		wg.Done()
	})
	defer unsub1()
	defer unsub2()

	bus.Publish("topic", "hello")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for subscribers")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("got %d messages, want 2", len(got))
	}
}
