# Actor Package

The `actor` package provides an **actor model** abstraction built on Fluxor's EventLoop and WorkerPool. It is extracted from the pattern used in **apps/media-tx** (RTMPConnectionActor, StreamManagerActor).

## Model

- **One logical actor** = a **routing key** + **EventLoopGroup** (serialized dispatch) + optional **WorkerPools** (I/O offload).
- All work for the same key runs on the same EventLoop (single-threaded, no locks).
- Blocking I/O is offloaded to ReadPool/WritePool; callbacks are dispatched back to the actor's EventLoop for thread safety.

## Rule: Callbacks must not block

**Actor callbacks MUST NOT block.** Handlers passed to `Dispatch`, and callbacks passed to `SubmitRead`/`SubmitWrite`, run on the actor's EventLoop (one goroutine per key). Blocking there stalls all work for that actor. Use **SubmitRead** / **SubmitWrite** for blocking I/O; keep Dispatch handlers and callbacks fast and non-blocking.

## Usage

```go
eventLoop, _ := eventloop.NewEventLoopGroup(ctx, eventloop.EventLoopConfig{...})
readPool := concurrency.NewWorkerPool(ctx, concurrency.DefaultWorkerPoolConfig())
readPool.Start()
writePool := concurrency.NewWorkerPool(ctx, concurrency.DefaultWorkerPoolConfig())
writePool.Start()

a := actor.NewActor("conn-1", eventLoop,
    actor.WithReadPool(readPool),
    actor.WithWritePool(writePool),
)

// Dispatch work to actor's EventLoop (serialized)
a.Dispatch(ctx, "start", func(ctx context.Context, event *eventloop.Event) error {
    return doHandshake()
})

// Offload blocking read; callback runs on actor's EventLoop
a.SubmitRead("handshake", func() (interface{}, error) {
    return doBlockingRead()
}, func(result interface{}, err error) {
    if err != nil { ... }
})

// Offload blocking write; callback runs on actor's EventLoop
a.SubmitWrite("flush", func() error { return conn.Flush() }, func(err error) { ... })
```

## Supervisor

A **Supervisor** monitors child processes (Runners) and restarts them when they fail.

- **Runner**: interface with `Run(ctx context.Context) error`. When `Run` returns, the child has stopped; non-nil error triggers a restart (subject to strategy and limits).
- **ChildSpec**: `Name` + `Start(ctx) (Runner, error)` to create a child.
- **Strategies**: `OneForOne` (restart only the failed child), `AllForOne` (restart all children when any fails).
- **Limits**: `WithMaxRestarts(n)` and `WithRestartWindow(d)` cap restarts (e.g. 5 restarts in 10s) to avoid thrashing.

```go
sv := actor.NewSupervisor("my-supervisor", []actor.ChildSpec{
    {Name: "worker-1", Start: func(ctx context.Context) (actor.Runner, error) {
        return myWorker1(ctx), nil
    }},
}, actor.WithStrategy(actor.OneForOne), actor.WithMaxRestarts(5))
go sv.Start(ctx)
defer sv.Stop()
```

## API

| Method | Description |
|--------|-------------|
| `NewActor(key, eventLoop, opts...)` | Create actor with key and EventLoopGroup; optional WithReadPool, WithWritePool. |
| `Key()` | Return routing key (same key → same EventLoop). |
| `Dispatch(ctx, name, handler)` | Run handler on this actor's EventLoop (serialized). |
| `SubmitRead(name, work, callback)` | Run work on ReadPool; callback dispatched to actor's EventLoop. |
| `SubmitWrite(name, work, callback)` | Run work on WritePool; callback dispatched to actor's EventLoop. |
| `NewSupervisor(name, children, opts...)` | Create supervisor; optional WithStrategy, WithMaxRestarts, WithRestartWindow. |
| `Supervisor.Start(ctx)` | Start all children and monitor; restarts on failure. |
| `Supervisor.Stop()` | Cancel all children and wait for Start to return. |

## Origin

This pattern comes from **apps/media-tx** (`rtmp_verticle.go`):

- **RTMPConnectionActor**: one connection per actor; `dispatch(name, handler)` and `submitRead`/`submitWrite` with I/O callbacks back on the actor's EventLoop.
- **StreamManagerActor**: manages streams; forwards media via EventLoop.

Use this package when you need the same pattern (key-based serialized work + optional I/O pools) without duplicating the boilerplate.

## WorkerPool size and profile (4.2)

Demo apps (e.g. actor-demo) may use a small pool, e.g. `Workers: 2`. At framework level, pool size should follow **profile**:

- **local**: more workers (e.g. `runtime.GOMAXPROCS(0)`).
- **autopilot** (e.g. GKE Autopilot): fewer workers to respect resource limits.

Later, WorkerPool config can be: `Workers: runtime.GOMAXPROCS(0)` or read from entrypoint profile (e.g. `WithProfile("gke-autopilot")`). The actor package does not define profiles; the app or entrypoint supplies pool config.

## Actor key lifecycle (4.3)

**Current**: An actor lives for the whole verticle (or until the app explicitly stops it). There is no built-in idle timeout or key GC.

**Optional future** (only when scale is very large): idle timeout per key, GC of idle actor keys to reclaim resources. Not required for typical use; add only if needed at scale.

## See Also

- `pkg/core/eventloop` – EventLoopGroup, Event, Dispatcher
- `pkg/core/concurrency` – WorkerPool, Task
- `pkg/entrypoint` – MainVerticle, WithProfile (runtime profile)
- `apps/media-tx/rtmp_verticle.go` – RTMPConnectionActor, StreamManagerActor

Supervisor: OTP-style process supervision; use when you need to run and restart child workers (Runner) on failure.
