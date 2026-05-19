package fluxor

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fluxorio/fluxor/pkg/lite/core"
	"github.com/google/uuid"
)

type App struct {
	bus    *core.Bus
	worker *core.WorkerPool

	ctx    context.Context
	cancel context.CancelFunc

	mu          sync.Mutex
	deployments []core.Component
}

func New() *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		bus:         core.NewBus(),
		worker:      core.NewWorkerPool(10, 1024),
		ctx:         ctx,
		cancel:      cancel,
		deployments: make([]core.Component, 0),
	}
}

func (a *App) Bus() *core.Bus {
	return a.bus
}

func (a *App) Worker() *core.WorkerPool {
	return a.worker
}

func (a *App) Deploy(c core.Component) {
	id := uuid.New().String()
	fctx := core.NewFluxorContext(a.ctx, a.bus, a.worker, id)

	if err := c.OnStart(fctx); err != nil {
		fmt.Printf("❌ Deploy failed: %v\n", err)
		return
	}

	a.mu.Lock()
	a.deployments = append(a.deployments, c)
	a.mu.Unlock()
}

func (a *App) Run() {
	fmt.Println("🚀 Fluxor (lite) running... (Ctrl+C to stop)")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\n🛑 Fluxor shutdown")
	a.cancel()

	a.mu.Lock()
	deps := append([]core.Component(nil), a.deployments...)
	a.mu.Unlock()

	for _, c := range deps {
		_ = c.OnStop()
	}

	a.worker.Shutdown()
}
