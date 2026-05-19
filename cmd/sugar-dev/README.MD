┌─────────────────────────────────────────┐
│             Fluxor App                  │
│  ┌─────────────────────────────────┐    │
│  │     Verticle                    │    │
│  │  ┌───────────────────────────┐  │    │
│  │  │   ? goroutine             │  │    │
│  │  │   ? OS thread             │  │    │
│  │  │   ? event loop            │  │    │
│  │  └───────────────────────────┘  │    │
│  └─────────────────────────────────┘    │
│                                         │
│  Workers: 500 goroutines (HTTP pool)    │
└─────────────────────────────────────────┘
Submit()                          Shutdown()
   │                                  │
   ├── RLock()                        ├── atomic.Set(closed=1)
   │   │                              │
   │   ├── check closed ──────────────┼── Lock() (waits for RUnlock)
   │   │                              │   │
   │   ├── chan <- task (safe)        │   ├── cancel()
   │   │                              │   │
   │   └── RUnlock()                  │   ├── close(taskChan) (safe)
   │                                  │   │
   │                                  │   └── Unlock()