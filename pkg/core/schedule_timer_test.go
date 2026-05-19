package core

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewScheduleTimer(t *testing.T) {
	timer := NewScheduleTimer(time.Second)
	if timer == nil {
		t.Fatal("NewScheduleTimer returned nil")
	}
	if timer.Tick != time.Second {
		t.Errorf("Tick = %v, want 1s", timer.Tick)
	}
	if timer.PublishAddress != ScheduleTickAddress {
		t.Errorf("PublishAddress = %q, want %q", timer.PublishAddress, ScheduleTickAddress)
	}
	if timer.Running() {
		t.Error("new timer should not be running")
	}
	if timer.Uptime() != 0 {
		t.Errorf("Uptime() = %v, want 0", timer.Uptime())
	}
	if timer.TickCount() != 0 {
		t.Errorf("TickCount() = %d, want 0", timer.TickCount())
	}
}

func TestScheduleTimer_StartStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timer := NewScheduleTimer(20 * time.Millisecond)
	timer.Start(ctx)

	// Allow at least one tick
	time.Sleep(50 * time.Millisecond)

	if !timer.Running() {
		t.Error("timer should be running")
	}
	if timer.Uptime() < 20*time.Millisecond {
		t.Errorf("Uptime() = %v, want >= 20ms", timer.Uptime())
	}
	if timer.TickCount() < 1 {
		t.Errorf("TickCount() = %d, want >= 1", timer.TickCount())
	}

	timer.Stop()
	time.Sleep(30 * time.Millisecond)
	if timer.Running() {
		t.Error("timer should not be running after Stop()")
	}
}

func TestScheduleTimer_StartIdempotent(t *testing.T) {
	ctx := context.Background()
	timer := NewScheduleTimer(10 * time.Millisecond)
	timer.Start(ctx)
	timer.Start(ctx) // second Start should no-op
	time.Sleep(25 * time.Millisecond)
	timer.Stop()
	// Should not panic and should have run
	if timer.TickCount() < 1 {
		t.Errorf("TickCount() = %d, want >= 1", timer.TickCount())
	}
}

func TestScheduleTimer_RegisterAndRunTasks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var steps int64
	timer := NewScheduleTimer(15 * time.Millisecond)
	timer.Register("count", func(uptime time.Duration, tick int64) {
		atomic.AddInt64(&steps, 1)
	})
	timer.Start(ctx)

	time.Sleep(60 * time.Millisecond)
	timer.Stop()

	n := atomic.LoadInt64(&steps)
	if n < 2 {
		t.Errorf("task step count = %d, want >= 2", n)
	}
}

func TestScheduleTimer_PublishesToEventBus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	received := make(chan *UptimeEvent, 4)
	consumer := eb.Consumer(ScheduleTickAddress)
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		var e UptimeEvent
		if err := msg.DecodeBody(&e); err != nil {
			t.Logf("DecodeBody: %v", err)
			return nil
		}
		received <- &e
		return nil
	})

	timer := NewScheduleTimer(20 * time.Millisecond)
	timer.SetEventBus(eb, ScheduleTickAddress)
	timer.Start(ctx)
	defer timer.Stop()

	// Wait for at least 2 events
	var events []*UptimeEvent
	for len(events) < 2 {
		select {
		case e := <-received:
			events = append(events, e)
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("received %d events, want >= 2", len(events))
		}
	}

	if events[0].Tick != 1 {
		t.Errorf("first event Tick = %d, want 1", events[0].Tick)
	}
	if events[0].Address != ScheduleTickAddress {
		t.Errorf("Address = %q, want %q", events[0].Address, ScheduleTickAddress)
	}
	if events[0].Uptime < 0 {
		t.Errorf("Uptime = %v", events[0].Uptime)
	}
	if events[1].Tick != 2 {
		t.Errorf("second event Tick = %d, want 2", events[1].Tick)
	}
}

func TestScheduleTimer_SetEventBusCustomAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	addr := "custom.schedule.uptime"
	received := make(chan *UptimeEvent, 2)
	consumer := eb.Consumer(addr)
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		var e UptimeEvent
		_ = msg.DecodeBody(&e)
		received <- &e
		return nil
	})

	timer := NewScheduleTimer(20 * time.Millisecond)
	timer.SetEventBus(eb, addr)
	timer.Start(ctx)
	defer timer.Stop()

	select {
	case e := <-received:
		if e.Address != addr {
			t.Errorf("Address = %q, want %q", e.Address, addr)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("no event received")
	}
}

func TestNewScheduleTimerVerticle(t *testing.T) {
	cfg := ScheduleTimerVerticleConfig{Tick: time.Second}
	v := NewScheduleTimerVerticle(cfg)
	if v == nil {
		t.Fatal("NewScheduleTimerVerticle returned nil")
	}
	if v.config.Tick != time.Second {
		t.Errorf("config.Tick = %v, want 1s", v.config.Tick)
	}
	if v.config.PublishAddress != ScheduleTickAddress {
		t.Errorf("PublishAddress = %q, want %q", v.config.PublishAddress, ScheduleTickAddress)
	}
}

func TestNewScheduleTimerVerticle_DefaultTick(t *testing.T) {
	v := NewScheduleTimerVerticle(ScheduleTimerVerticleConfig{Tick: 0})
	if v.config.Tick != time.Second {
		t.Errorf("default Tick = %v, want 1s", v.config.Tick)
	}
}

func TestScheduleTimerVerticle_StartStop(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	received := make(chan *UptimeEvent, 4)
	consumer := gocmd.EventBus().Consumer(ScheduleTickAddress)
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		var e UptimeEvent
		_ = msg.DecodeBody(&e)
		received <- &e
		return nil
	})

	verticle := NewScheduleTimerVerticle(ScheduleTimerVerticleConfig{
		Tick: 25 * time.Millisecond,
	})
	depID, err := gocmd.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle: %v", err)
	}
	defer gocmd.UndeployVerticle(depID)

	// Wait for at least one tick event
	select {
	case e := <-received:
		if e.Tick < 1 {
			t.Errorf("Tick = %d", e.Tick)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no tick event received from verticle")
	}

	err = gocmd.UndeployVerticle(depID)
	if err != nil {
		t.Errorf("UndeployVerticle: %v", err)
	}
}

func TestScheduleTickAddress_Constant(t *testing.T) {
	if ScheduleTickAddress != "core.schedule.tick" {
		t.Errorf("ScheduleTickAddress = %q, want core.schedule.tick", ScheduleTickAddress)
	}
}
