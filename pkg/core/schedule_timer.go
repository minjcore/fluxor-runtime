// Package core's schedule_timer provides a kernel-style tick loop (like go-micron's kernel/scheduler)
// that runs at a fixed interval and can fire events each tick. Use it for uptime checks, heartbeat,
// or any periodic work that should publish to the event bus so verticles can subscribe and react.
//
// Example: deploy ScheduleTimerVerticle and consume core.schedule.tick
//
//	app.DeployVerticle(core.NewScheduleTimerVerticle(core.ScheduleTimerVerticleConfig{
//	    Tick: time.Second,
//	}))
//	app.GoCMD().EventBus().Consumer(core.ScheduleTickAddress).Handler(func(ctx core.FluxorContext, msg core.Message) error {
//	    var e core.UptimeEvent
//	    _ = msg.DecodeBody(&e)
//	    log.Printf("uptime %v tick %d", e.Uptime, e.Tick)
//	    return nil
//	})
package core

import (
	"context"
	"sync"
	"time"
)

// ScheduleTickAddress is the default EventBus address for schedule tick/uptime events.
// Subscribers can Consumer(..., ScheduleTickAddress) to react each tick (e.g. health, metrics).
const ScheduleTickAddress = "core.schedule.tick"

// UptimeEvent is the payload published on each schedule tick.
// Handlers can use it for uptime checks, heartbeat, or periodic work.
type UptimeEvent struct {
	Uptime   time.Duration `json:"uptime"`   // Time since timer start
	Tick     int64         `json:"tick"`     // Tick count (1-based)
	At       time.Time     `json:"at"`       // Wall clock time of this tick
	Address  string        `json:"address"`  // Address this was published to (e.g. ScheduleTickAddress)
}

// ScheduleTask runs one step per tick (kernel-style, like go-micron).
// Step is called each tick before publishing the event; use for lightweight periodic work.
type ScheduleTask struct {
	Name string
	Step func(uptime time.Duration, tick int64)
}

// ScheduleTimer runs a tick loop and optionally runs tasks and publishes an event each tick.
// Similar to a minimal kernel scheduler: every Tick interval it runs registered tasks and
// can fire an event (e.g. uptime) so verticles can subscribe and react.
type ScheduleTimer struct {
	Tick       time.Duration  // Interval between ticks (e.g. 1*time.Second)
	Tasks      []ScheduleTask // Optional tasks run each tick (round-robin)
	EventBus   EventBus       // Optional: if set, publish UptimeEvent each tick to PublishAddress
	PublishAddress string    // Address to publish to (default ScheduleTickAddress)

	mu         sync.Mutex
	startTime  time.Time
	tickCount  int64
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
}

// NewScheduleTimer creates a timer with the given tick interval.
// Call Start(ctx) to begin the loop; it publishes to EventBus each tick if EventBus is set.
func NewScheduleTimer(tick time.Duration) *ScheduleTimer {
	return &ScheduleTimer{
		Tick:           tick,
		Tasks:          nil,
		PublishAddress: ScheduleTickAddress,
	}
}

// Register adds a task that runs one step per tick (kernel-style).
func (st *ScheduleTimer) Register(name string, step func(uptime time.Duration, tick int64)) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.Tasks = append(st.Tasks, ScheduleTask{Name: name, Step: step})
}

// SetEventBus sets the event bus and optional publish address for tick events.
func (st *ScheduleTimer) SetEventBus(eb EventBus, address string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.EventBus = eb
	if address != "" {
		st.PublishAddress = address
	}
}

// Start starts the tick loop in a goroutine. It runs until ctx is cancelled or Stop() is called.
func (st *ScheduleTimer) Start(ctx context.Context) {
	st.mu.Lock()
	if st.running {
		st.mu.Unlock()
		return
	}
	st.ctx, st.cancel = context.WithCancel(ctx)
	st.startTime = time.Now()
	st.tickCount = 0
	st.running = true
	st.mu.Unlock()

	go st.run()
}

// run is the main loop: sleep tick, then run tasks and optionally publish event.
func (st *ScheduleTimer) run() {
	ticker := time.NewTicker(st.Tick)
	defer ticker.Stop()

	for {
		select {
		case <-st.ctx.Done():
			st.mu.Lock()
			st.running = false
			st.mu.Unlock()
			return
		case <-ticker.C:
			st.mu.Lock()
			st.tickCount++
			tick := st.tickCount
			uptime := time.Since(st.startTime)
			tasks := append([]ScheduleTask(nil), st.Tasks...)
			eb := st.EventBus
			addr := st.PublishAddress
			st.mu.Unlock()

			for _, t := range tasks {
				if t.Step != nil {
					t.Step(uptime, tick)
				}
			}

			if eb != nil && addr != "" {
				_ = eb.Publish(addr, &UptimeEvent{
					Uptime:  uptime,
					Tick:    tick,
					At:      time.Now(),
					Address: addr,
				})
			}
		}
	}
}

// Stop stops the tick loop.
func (st *ScheduleTimer) Stop() {
	st.mu.Lock()
	cancel := st.cancel
	st.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Uptime returns the time since Start() was called. Zero if not started.
func (st *ScheduleTimer) Uptime() time.Duration {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.startTime.IsZero() {
		return 0
	}
	return time.Since(st.startTime)
}

// TickCount returns the number of ticks since start. 0 if not started.
func (st *ScheduleTimer) TickCount() int64 {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.tickCount
}

// Running reports whether the timer loop is running.
func (st *ScheduleTimer) Running() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.running
}

// ScheduleTimerVerticleConfig configures the schedule timer verticle.
type ScheduleTimerVerticleConfig struct {
	Tick            time.Duration  // Interval between ticks (e.g. 1*time.Second)
	PublishAddress  string         // EventBus address for tick events (default ScheduleTickAddress)
	Tasks           []ScheduleTask // Optional tasks run each tick
}

// ScheduleTimerVerticle is a verticle that runs a ScheduleTimer and publishes tick/uptime events.
// Deploy it to get periodic events at core.schedule.tick (or a custom address); any consumer
// can subscribe to react (e.g. uptime check, heartbeat, metrics).
type ScheduleTimerVerticle struct {
	*BaseVerticle
	config ScheduleTimerVerticleConfig
	timer  *ScheduleTimer
}

// NewScheduleTimerVerticle creates a verticle that runs the schedule timer with the given config.
func NewScheduleTimerVerticle(config ScheduleTimerVerticleConfig) *ScheduleTimerVerticle {
	if config.Tick <= 0 {
		config.Tick = time.Second
	}
	if config.PublishAddress == "" {
		config.PublishAddress = ScheduleTickAddress
	}
	return &ScheduleTimerVerticle{
		BaseVerticle: NewBaseVerticle("schedule-timer"),
		config:       config,
	}
}

// Start starts the schedule timer and publishes tick events to the event bus.
func (v *ScheduleTimerVerticle) Start(ctx FluxorContext) error {
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}
	v.timer = NewScheduleTimer(v.config.Tick)
	for _, t := range v.config.Tasks {
		v.timer.Register(t.Name, t.Step)
	}
	v.timer.SetEventBus(v.eventBus, v.config.PublishAddress)
	v.timer.Start(ctx.Context())
	return nil
}

// Stop stops the schedule timer.
func (v *ScheduleTimerVerticle) Stop(ctx FluxorContext) error {
	if v.timer != nil {
		v.timer.Stop()
		v.timer = nil
	}
	return v.BaseVerticle.Stop(ctx)
}
