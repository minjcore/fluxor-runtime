# Database Runtime Architecture Flow

## Overview
This document describes the flow of database operations in the separated architecture using PoolManager (dedicated goroutine) and ConnectionExecutor (worker goroutines).

## Architecture Components

### 1. PostgresDemoVerticle (Main Service)
- Main application verticle
- Manages database component lifecycle
- Executes business logic with database operations

### 2. DatabaseComponent
- Wraps PoolManager and ConnectionExecutor
- Provides synchronous and asynchronous database APIs
- Manages component lifecycle

### 3. PoolManager
- Runs in dedicated goroutine
- Manages connection pool lifecycle
- Performs health checks and monitoring
- Publishes pool statistics via EventBus

### 4. ConnectionExecutor
- Runs in worker goroutine pool
- Executes database operations asynchronously
- Handles job queue and worker distribution

## Flow Diagrams

### Component Initialization Flow

```mermaid
sequenceDiagram
    participant Main as Main Application
    participant Verticle as PostgresDemoVerticle
    participant DBComp as DatabaseComponent
    participant PM as PoolManager
    participant Exec as ConnectionExecutor
    participant Pool as Connection Pool
    participant DB as PostgreSQL

    Main->>Verticle: DeployVerticle()
    Main->>Verticle: Start()
    
    Verticle->>DBComp: NewDatabaseComponent(config)
    Note over DBComp: Creates component with hooks
    
    Verticle->>DBComp: Start(ctx)
    DBComp->>PM: NewPoolManager(config, eventBus)
    DBComp->>PM: Start()
    PM->>Pool: NewPool(config)
    Pool->>DB: sql.Open() + Ping()
    DB-->>Pool: Connection established
    Pool-->>PM: Pool ready
    PM->>PM: Start manager goroutine
    Note over PM: Health checks every 30s
    PM-->>DBComp: Started
    
    DBComp->>Exec: NewConnectionExecutor(pm, workers, queue, timeout)
    DBComp->>Exec: Start()
    Exec->>Exec: Start worker goroutines
    Note over Exec: N workers (default: CPU count)
    Exec-->>DBComp: Started
    
    DBComp-->>Verticle: Started successfully
    Verticle->>Verticle: Start health check goroutine
    Note over Verticle: Periodic query every 2s
    Verticle-->>Main: Ready
```

### Database Operation Flow (Synchronous)

```mermaid
sequenceDiagram
    participant Verticle as PostgresDemoVerticle
    participant DBComp as DatabaseComponent
    participant PM as PoolManager
    participant Pool as Connection Pool
    participant DB as PostgreSQL

    Verticle->>DBComp: QueryRow(ctx, "SELECT 1")
    DBComp->>PM: GetPool()
    PM-->>DBComp: Pool reference
    DBComp->>Pool: QueryRowContext(ctx, query)
    Pool->>DB: Get connection from pool
    DB-->>Pool: Connection
    Pool->>DB: Execute query
    DB-->>Pool: Result
    Pool-->>DBComp: *sql.Row
    DBComp-->>Verticle: Result
    Verticle->>Verticle: Process result
```

### Database Operation Flow (Asynchronous)

```mermaid
sequenceDiagram
    participant Verticle as PostgresDemoVerticle
    participant DBComp as DatabaseComponent
    participant Exec as ConnectionExecutor
    participant Worker as Worker Goroutine
    participant PM as PoolManager
    participant Pool as Connection Pool
    participant DB as PostgreSQL

    Verticle->>DBComp: QueryRowAsync(ctx, "SELECT 2")
    DBComp->>Exec: QueryRow(ctx, query)
    Exec->>Exec: Create DBJob
    Exec->>Exec: Submit to jobQueue
    Exec-->>DBComp: resultChan
    DBComp-->>Verticle: <-chan *JobResult
    
    Note over Exec,Worker: Worker picks up job
    Exec->>Worker: Job from queue
    Worker->>PM: GetPool()
    PM-->>Worker: Pool reference
    Worker->>Pool: DB()
    Pool-->>Worker: *sql.DB
    Worker->>DB: QueryRowContext(ctx, query)
    DB-->>Worker: Result
    Worker->>Worker: Send result to channel
    Worker-->>Exec: JobResult
    Exec-->>Verticle: Result via channel
    Verticle->>Verticle: Process result
```

### Health Check Flow

```mermaid
sequenceDiagram
    participant Verticle as PostgresDemoVerticle
    participant Ticker as Time.Ticker (2s)
    participant DBComp as DatabaseComponent
    participant PM as PoolManager
    participant Pool as Connection Pool
    participant DB as PostgreSQL

    loop Every 2 seconds
        Ticker->>Verticle: Tick
        Verticle->>DBComp: QueryRow(ctx, "SELECT 1")
        DBComp->>PM: GetPool()
        PM-->>DBComp: Pool reference
        DBComp->>Pool: QueryRowContext(ctx, query)
        Pool->>DB: Get connection
        DB-->>Pool: Connection
        Pool->>DB: Execute SELECT 1
        DB-->>Pool: Result (1)
        Pool-->>DBComp: *sql.Row
        DBComp-->>Verticle: Result
        Verticle->>Verticle: Log colored result
    end
```

### Pool Manager Monitoring Flow

```mermaid
sequenceDiagram
    participant PM as PoolManager Goroutine
    participant Pool as Connection Pool
    participant DB as PostgreSQL
    participant EventBus as EventBus

    loop Every 30 seconds
        PM->>Pool: Ping(ctx)
        Pool->>DB: Ping connection
        alt Connection OK
            DB-->>Pool: Success
            Pool-->>PM: Success
            PM->>PM: Update lastHealthCheck
        else Connection Failed
            DB-->>Pool: Error
            Pool-->>PM: Error
            PM->>EventBus: Publish "pool.health.check.failed"
        end
        
        PM->>Pool: Stats()
        Pool-->>PM: sql.DBStats
        PM->>EventBus: Publish "pool.stats"
        Note over EventBus: {open_connections, in_use, idle, wait_count}
    end
```

### Shutdown Flow

```mermaid
sequenceDiagram
    participant Main as Main Application
    participant Verticle as PostgresDemoVerticle
    participant HealthCheck as Health Check Goroutine
    participant DBComp as DatabaseComponent
    participant Exec as ConnectionExecutor
    participant PM as PoolManager
    participant Pool as Connection Pool

    Main->>Verticle: Stop() (SIGINT/SIGTERM)
    
    Verticle->>HealthCheck: Cancel context
    Verticle->>HealthCheck: Stop ticker
    HealthCheck->>HealthCheck: Exit loop
    Verticle->>HealthCheck: Wait for completion
    
    Verticle->>DBComp: Stop(ctx)
    DBComp->>Exec: Stop()
    Exec->>Exec: Cancel context
    Exec->>Exec: Close jobQueue
    Exec->>Exec: Wait for workers
    Note over Exec: All workers finish current jobs
    Exec-->>DBComp: Stopped
    
    DBComp->>PM: Stop()
    PM->>PM: Cancel context
    PM->>PM: Wait for manager goroutine
    PM->>Pool: Close()
    Pool->>Pool: Close all connections
    Pool-->>PM: Closed
    PM-->>DBComp: Stopped
    
    DBComp-->>Verticle: Stopped
    Verticle->>Verticle: Stop BaseVerticle
    Verticle-->>Main: Stopped
```

## Goroutine Architecture

```mermaid
graph TB
    subgraph MainProcess["Main Process"]
        MainApp[Main Application]
    end
    
    subgraph VerticleProcess["PostgresDemoVerticle"]
        Verticle[PostgresDemoVerticle]
        HealthCheckGoroutine[Health Check Goroutine<br/>Every 2 seconds]
    end
    
    subgraph DatabaseComponent["DatabaseComponent"]
        DBComp[DatabaseComponent]
    end
    
    subgraph PoolManagerProcess["PoolManager (Dedicated Goroutine)"]
        PMGoroutine[Pool Manager Goroutine<br/>Health checks + Monitoring]
        PoolManager[PoolManager]
    end
    
    subgraph ConnectionExecutorProcess["ConnectionExecutor (Worker Pool)"]
        Executor[ConnectionExecutor]
        JobQueue[Job Queue<br/>Size: 1000]
        Worker1[Worker Goroutine 1]
        Worker2[Worker Goroutine 2]
        WorkerN[Worker Goroutine N<br/>Default: CPU count]
    end
    
    subgraph DatabasePool["Connection Pool"]
        Pool[Pool<br/>MaxOpenConns: 5<br/>MaxIdleConns: 2]
    end
    
    subgraph PostgreSQL["PostgreSQL Database"]
        DB[(PostgreSQL<br/>fluxor_db)]
    end
    
    MainApp -->|Deploy| Verticle
    Verticle -->|Manages| DBComp
    Verticle -->|Starts| HealthCheckGoroutine
    
    DBComp -->|Manages| PoolManager
    DBComp -->|Manages| Executor
    
    PoolManager -->|Runs in| PMGoroutine
    PMGoroutine -->|Monitors| Pool
    
    Executor -->|Manages| JobQueue
    JobQueue -->|Distributes| Worker1
    JobQueue -->|Distributes| Worker2
    JobQueue -->|Distributes| WorkerN
    
    Worker1 -->|Uses| Pool
    Worker2 -->|Uses| Pool
    WorkerN -->|Uses| Pool
    
    HealthCheckGoroutine -->|Queries| DBComp
    DBComp -->|Sync ops| Pool
    DBComp -->|Async ops| Executor
    
    Pool -->|Connections| DB
    PMGoroutine -->|Health checks| DB
```

## Key Points

1. **Separation of Concerns**:
   - PoolManager: Lifecycle and monitoring (dedicated goroutine)
   - ConnectionExecutor: Operation execution (worker pool)
   - DatabaseComponent: Unified API

2. **Concurrency**:
   - Each component runs in separate goroutines
   - No blocking between pool management and operations
   - Worker pool handles concurrent operations

3. **Thread Safety**:
   - All shared state protected by mutexes
   - Pool access is thread-safe
   - Job queue is channel-based (thread-safe)

4. **Scalability**:
   - Configurable worker count
   - Configurable queue size
   - Pool size limits prevent connection exhaustion

5. **Monitoring**:
   - Health checks every 30s (PoolManager)
   - Periodic queries every 2s (Verticle)
   - Statistics published via EventBus
