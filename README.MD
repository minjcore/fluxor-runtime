# Fluxor Runtime

**Reactive runtime for Go** — Vert.x/OTP-style semantics: event bus, verticles, structural concurrency, backpressure. Build high-performance, predictable backends as a single binary.

---

## Architecture

```
                        ┌─────────────────────────────────┐
  HTTP Request          │         FastHTTP Server          │
  ─────────────────────▶│  CCU-based backpressure (503)   │
                        │  Workers + Queue + fail-fast     │
                        └──────────────┬──────────────────┘
                                       │
                        ┌──────────────▼──────────────────┐
                        │         FastRouter               │
                        │  middleware chain (metrics)      │
                        └──────────────┬──────────────────┘
                                       │
               ┌───────────────────────▼──────────────────────┐
               │              EventLoopGroup                   │
               │   Loop 0          Loop 1          Loop N      │
               │  ┌────────┐      ┌────────┐      ┌────────┐  │
               │  │Disruptor│      │Disruptor│      │Disruptor│ │
               │  │RingBuf │      │RingBuf │      │RingBuf │  │
               │  └────────┘      └────────┘      └────────┘  │
               └───────────────────────────────────────────────┘
                                       │
                        ┌──────────────▼──────────────────┐
                        │          EventBus                │
                        │  Verticle deploy/undeploy        │
                        │  Publish / Send / Request        │
                        └─────────────────────────────────┘
```

### Key components

| Component | Description |
|-----------|-------------|
| **FastHTTPServer** | CCU-based backpressure — operates at 67% target utilization, fail-fast 503 beyond capacity |
| **EventLoopGroup** | N event loops backed by LMAX Disruptor RingBuffer (power-of-2 size, YieldWait strategy) |
| **EventBus** | Vert.x-style pub/sub — publish, send, request/reply across verticles |
| **Verticle** | Unit of deployment — isolated actor with start/stop lifecycle |
| **FastRouter** | High-performance router with middleware chain |

---

## LMAX Disruptor EventLoop

The `EventLoopGroup` uses a lock-free ring buffer (Disruptor pattern) instead of Go channels:

- **`Next()`** — claims a sequence slot via CAS
- **`Publish(seq, ptr)`** — writes data then marks `available[slot] = seq`
- **`WaitFor(seq, ctx)`** — polls per-slot availability (YieldWait: spin 100× then `Gosched()`)
- **`Consume(seq)`** — called per-event so blocking handlers don't starve producers

```go
// EventLoop core — two-phase publish
seq := rb.Next()           // claim slot
rb.Publish(seq, unsafe.Pointer(event))  // write + mark available

// consumer side
avail, _ := rb.WaitFor(next, ctx)
for seq := next; seq <= avail; seq++ {
    event := (*Event)(rb.Get(seq))
    event.Handler(ctx, event)
    rb.Consume(seq)        // per-event, not batch
}
```

---

## Observability

Full Prometheus + Grafana stack included.

### Metrics exposed (`/metrics`)

| Metric | Type | Description |
|--------|------|-------------|
| `fluxor_http_requests_total` | counter | Request count by method, path, status |
| `fluxor_http_request_duration_seconds` | histogram | Latency p50/p95/p99 |
| `fluxor_server_rejected_requests_total` | counter | Backpressure 503 rejections |
| `fluxor_eventloop_queue_length` | gauge | Per-loop queue depth |
| `fluxor_eventloop_latency_seconds` | gauge | Per-loop avg processing latency |
| `fluxor_eventloop_processed_messages_total` | counter | Per-loop processed events |
| `go_goroutines` | gauge | Runtime goroutine count |
| `go_memstats_alloc_bytes` | gauge | Heap in-use |
| `go_gc_duration_seconds` | summary | GC pause distribution |

### Alert rules (`observability/alerts.yml`)

| Alert | Condition | Severity |
|-------|-----------|----------|
| `HTTPErrorRequests` | Any 5xx response in 1m | critical |
| `BackpressureRejections` | Any 503 rejection in 1m | critical |
| `HighErrorRate` | 5xx rate > 1% over 2m | warning |
| `HighLatencyP99` | p99 latency > 1s for 2m | warning |

### Grafana dashboard

13 panels auto-provisioned:

```
┌──────────┬────────────┬────────────┬────────────┬──────────┬────────────┐
│   RPS    │ Goroutines │ Heap Alloc │  GC Pause  │   CPU    │ EventLoop  │
│          │            │            │   (p99)    │  cores   │ Processed  │
├──────────┴──────────────────────────┬───────────┴────────────────────────┤
│         Request Rate                │       Latency (p50/p95/p99)        │
├─────────────┬──────────┬────────────┼──────────────────┬─────────────────┤
│   Memory    │Goroutines│ GC Duration│ EventLoop Queue  │ EventLoop Avg   │
│             │          │            │     Length       │    Latency      │
└─────────────┴──────────┴────────────┴──────────────────┴─────────────────┘
```

---

## Benchmarks

Load tested with `wrk` on VPS (single node):

| Endpoint | Connections | RPS | p50 | p99 |
|----------|-------------|-----|-----|-----|
| `GET /api/status` | 1,000 | **6,756 req/s** | 152ms | 285ms |
| `POST /api/echo` | 1,000 | **11,903 req/s** | 82ms | 160ms |
| `GET /api/status` | 5,000 | **11,531 req/s** | 407ms | 1s (backpressure) |

At 5,000 connections the backpressure system kicks in — excess requests receive `503` immediately (fail-fast), keeping the server stable rather than degrading.

---

## Quick start

```bash
# Run full stack (app + Prometheus + Grafana)
docker compose up -d

# Endpoints
curl http://localhost:8080/          # hello
curl http://localhost:8080/api/status
curl http://localhost:8080/metrics   # Prometheus scrape target
curl http://localhost:8080/live      # liveness
curl http://localhost:8080/ready     # readiness

# Grafana dashboard
open http://localhost:3000           # admin / admin

# Prometheus alerts
open http://localhost:9090/alerts
```

```bash
# Local build
go build ./cmd/...
```

---

## Project structure

```
cmd/                    # main entry point
pkg/
  core/                 # GoCMD, EventBus, Verticle, deployment
    eventloop/          # Disruptor RingBuffer EventLoopGroup
  web/                  # FastHTTPServer, FastRouter, backpressure
  observability/
    prometheus/         # metrics registry, middleware, exporter
observability/
  prometheus.yml        # scrape config
  alerts.yml            # alert rules
  grafana/
    provisioning/       # auto-provision datasource + dashboard
    dashboards/         # fluxor.json — 13-panel dashboard
docker-compose.yml      # fluxor + prometheus + grafana
```

---

## License

Proprietary — All Rights Reserved. For licensing: caokhang91@gmail.com
