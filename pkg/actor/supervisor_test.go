package actor

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSupervisor_OneForOne(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var runs int32
	children := []ChildSpec{
		{
			Name: "worker",
			Start: func(ctx context.Context) (Runner, error) {
				return &runnerFunc{run: func(ctx context.Context) error {
					n := atomic.AddInt32(&runs, 1)
					if n <= 2 {
						return errors.New("fail")
					}
					<-ctx.Done()
					return nil
				}}, nil
			},
		},
	}

	sv := NewSupervisor("test", children, WithStrategy(OneForOne), WithMaxRestarts(3), WithRestartWindow(time.Second))
	go sv.Start(ctx)
	defer sv.Stop()

	time.Sleep(500 * time.Millisecond)
	if atomic.LoadInt32(&runs) < 2 {
		t.Errorf("expected at least 2 runs (initial + 1 restart), got %d", runs)
	}
}

func TestSupervisor_Stop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	children := []ChildSpec{
		{
			Name: "blocker",
			Start: func(ctx context.Context) (Runner, error) {
				return &runnerFunc{run: func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				}}, nil
			},
		},
	}

	sv := NewSupervisor("test", children)
	go sv.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	sv.Stop()
}

type runnerFunc struct {
	run func(ctx context.Context) error
}

func (r *runnerFunc) Run(ctx context.Context) error {
	return r.run(ctx)
}
