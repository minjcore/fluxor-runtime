// Package actor provides an actor model abstraction for Fluxor applications.
//
// It is extracted from the pattern used in apps/media-tx (RTMPConnectionActor, StreamManagerActor):
// one logical actor = a routing key + EventLoopGroup (serialized dispatch) + optional WorkerPools (I/O offload).
//
// Rule: Actor callbacks MUST NOT block. Handlers passed to Dispatch, and callbacks passed to
// SubmitRead/SubmitWrite, run on the actor's EventLoop (single goroutine). Blocking there stalls
// all work for that actor. Use SubmitRead/SubmitWrite for blocking I/O; do only fast, non-blocking
// work in Dispatch handlers and in the callbacks.
//
// Supervisor: NewSupervisor creates a supervisor that starts child Runners (ChildSpec) and restarts
// them on failure. Strategies: OneForOne (restart only the failed child), AllForOne (restart all).
// Use WithMaxRestarts and WithRestartWindow to limit restart rate.
//
// Usage:
//
//	eventLoop, _ := eventloop.NewEventLoopGroup(ctx, config)
//	a := actor.NewActor("conn-1", eventLoop,
//	    actor.WithReadPool(readPool),
//	    actor.WithWritePool(writePool),
//	)
//	a.Dispatch(ctx, "start", func(ctx context.Context, event *eventloop.Event) error {
//	    return doWork()
//	})
//	a.SubmitRead("handshake", func() (interface{}, error) { return doRead() }, func(result, err) { ... })
//	a.SubmitWrite("flush", func() error { return conn.Flush() }, func(err) { ... })
//
// See pkg/core/eventloop and pkg/core/concurrency for EventLoopGroup and WorkerPool.
package actor
