# Managers Package

The `managers` package (Exmanagerstion Control Unit) provides a **control plane** for managing core application services: HTTP server, context, queue/EventBus, logging, observability, and cache.

## Car Analogy

**Runtime (GoCMD) = Car Structure**
- Wheels, engine, chassis, transmission - the physical components
- The actual structure that makes the car work
- Manages lifecycle (start/stop the car)
- Provides infrastructure (EventBus = fuel system, Verticles = car parts)

**Managers = Car's Electronic Control Unit**
- The control system that coordinates and manages the car's systems
- Monitors and coordinates engine, transmission, wheels, etc.
- Doesn't build the car or make it move
- Receives signals from components (engine started, wheel turned, etc.)
- Coordinates systems but doesn't control physical movement
- The "brain" that manages, but the car (runtime) does the actual work

**Components (Verticles) = Car Parts**
- Engine, wheels, transmission - individual parts
- Communicate with Managers through the car structure (runtime)
- Managers coordinates them, runtime provides infrastructure

**Application = Driver**
- Uses both the car (runtime) and Managers together
- Managers coordinates systems, runtime provides the infrastructure
- Driver starts the car (runtime starts components), Managers manages coordination

## Key Design Principles

1. **Control Plane Only**: Managers coordinates and manages, but does NOT build or move the car (runtime does that)
2. **Car Analogy**: Runtime = car structure, Managers = car's control unit, Components = car parts
3. **Composition with Runtime**: Managers is composed with GoCMD (car + control unit work together)
4. **Communication via Runtime**: Components access Managers through FluxorContext (via car structure)
5. **Service Registry**: Managers registers component instances for retrieval
6. **Configuration Management**: Unified configuration for all components (car system settings)
7. **Component Wiring**: Managers wires components together (coordinates car systems)
8. **Lifecycle Awareness**: Managers receives signals when components start/stop (monitors car parts)
9. **Event Observer**: Managers can observe and react to component lifecycle events

## Architecture

```
Application (Driver)
│
├── Runtime (GoCMD) ──── Car Structure (wheels, engine, chassis)
│   ├── EventBus ─────── Fuel system
│   ├── Verticles ────── Car parts (engine, wheels, transmission)
│   └── Context ──────── Car's internal wiring
│
└── Managers ───────────────── Car's Electronic Control Unit
    ├── Monitors components (receives signals)
    ├── Coordinates systems (wires components)
    ├── Manages configuration (system settings)
    └── Observes lifecycle (engine started, wheel turned)

Communication:
Components → Runtime → Managers (via FluxorContext)
Managers → Runtime → Components (via EventBus or signals)
```

## Quick Start

### Application Setup (Driver + Car + Managers)

```go
package main

import (
	"context"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/managers"
)

func main() {
	// Driver (Application) creates car (runtime) and Managers (control unit)
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)          // Build car structure
	managersInstance, _ := managers.NewManagersWithGoCMD(gocmd, managers.DefaultConfig()) // Install Managers

	// Create components via Managers
	logger, _ := managersInstance.CreateLogger()
	cache, _ := managersInstance.CreateCache()
	httpServer, _ := managersInstance.CreateHTTPServer(gocmd)

	// Register components with Managers
	managersInstance.RegisterLogger(logger)
	managersInstance.RegisterCache(cache)
	managersInstance.RegisterHTTPServer(httpServer)

	// Wire components together
	managersInstance.Wire()

	// Register event handlers
	managersInstance.OnHTTPServerStart(func(componentName string, event managers.ComponentEvent) {
		managersInstance.Logger().Info("Engine started", "component", componentName)
	})

	// Store Managers reference in context for verticles to access
	// This is typically done when creating/deploying verticles
	// Note: FluxorContext is created internally by GoCMD when deploying verticles
	// You can store Managers reference in a way that's accessible to verticles

	// Start components externally (NOT by Managers)
	httpServer.Start() // Engine starts → Managers receives signal
	defer httpServer.Stop()
}
```

### Component Access (Car Parts → Managers)

```go
// Car part (Verticle) accesses Managers through car wiring (FluxorContext)
type MyVerticle struct {
	*core.BaseVerticle
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
	// Car part accesses Managers through car wiring
	managersInstance, err := managers.GetManagers(ctx) // Access Managers via FluxorContext (car wiring)
	if err != nil {
		return err
	}
	
	// Car part uses Managers to coordinate with other parts
	managersInstance.Logger().Info("Car part started")
	managersInstance.Cache().Set(ctx.Context(), "key", []byte("value"), time.Hour)
	managersInstance.Queue().Publish("events", data) // Use fuel system (EventBus)
	
	return nil
}
```

## API Reference

### Factory Functions

- `NewManagers(ctx context.Context, config *Config) (*Managers, error)` - Creates Managers with new GoCMD
- `NewManagersWithGoCMD(gocmd core.GoCMD, config *Config) (*Managers, error)` - Creates Managers with existing GoCMD
- `DefaultConfig() *Config` - Returns default configuration

### Component Creation

- `CreateLogger() (core.Logger, error)` - Creates logger based on config
- `CreateJSONLogger() core.Logger` - Creates JSON logger
- `CreateCache() (cache.Cache, error)` - Creates cache based on config
- `CreateMemoryCache() cache.Cache` - Creates in-memory cache
- `CreateHTTPServer(gocmd core.GoCMD) (web.Server, error)` - Creates HTTP server (wrapped to send signals)
- `CreateFastHTTPServer(gocmd core.GoCMD, config *web.FastHTTPServerConfig) (*web.FastHTTPServer, error)` - Creates FastHTTP server
- `CreateMetrics() (*prometheus.Metrics, error)` - Creates metrics collector

### Component Registration

- `RegisterHTTPServer(server web.Server)` - Register HTTP server instance
- `RegisterLogger(logger core.Logger)` - Register logger instance
- `RegisterCache(cache cache.Cache)` - Register cache instance
- `RegisterMetrics(metrics *prometheus.Metrics)` - Register metrics instance

### Component Access

- `HTTPServer() web.Server` - Get registered HTTP server
- `Logger() core.Logger` - Get registered logger
- `Queue() core.EventBus` - Get EventBus from GoCMD (fuel system)
- `EventBus() core.EventBus` - Alias for Queue()
- `Cache() cache.Cache` - Get registered cache
- `Observe() *prometheus.Metrics` - Get registered metrics
- `Metrics() *prometheus.Metrics` - Alias for Observe()
- `Context() core.GoCMD` - Get GoCMD (car structure)

### Event Handling

- `OnComponentEvent(componentName string, handler ComponentEventHandler)` - Register event handler
- `OnHTTPServerStart(handler ComponentEventHandler)` - Register HTTP server start handler
- `OnHTTPServerStop(handler ComponentEventHandler)` - Register HTTP server stop handler
- `OnHeartbeat(handler ComponentEventHandler)` - Register Managers heartbeat event handler
- `OnHeartbeatMissed(componentName string, handler ComponentEventHandler)` - Register missed heartbeat handler for a component

### Coordination

- `Wire() error` - Wires components together (coordinates car systems)
- `AttachToContext(ctx core.FluxorContext)` - Attaches Managers to FluxorContext
- `Config() *Config` - Returns Managers configuration

### Heartbeat

- `StartHeartbeat() error` - Starts the heartbeat system (emits periodic events, publishes to EventBus)
- `StopHeartbeat()` - Stops the heartbeat system
- `SendHeartbeat(componentName string)` - Component sends heartbeat signal to Managers

### Context Helpers

- `GetManagers(ctx core.FluxorContext) (*Managers, error)` - Retrieves Managers from FluxorContext
- `WithManagers(ctx core.FluxorContext, managers *Managers)` - Stores Managers in FluxorContext config

## Communication Flow

```
Driver (Application) Setup:
1. Driver builds car (creates GoCMD/runtime)
2. Driver installs Managers (creates Managers with GoCMD)
3. Managers stores reference in context (for component access)
4. Managers creates car parts (components)
5. Managers registers car parts (component registry)
6. Managers coordinates car systems (Wire() connects systems)

Car Part (Component) Access:
1. Car part receives car wiring (FluxorContext) in Start()
2. Car part accesses Managers through car wiring (managers.GetManagers(ctx))
3. Car part uses Managers to coordinate with other parts

Car Part Lifecycle:
1. Driver starts car part (httpServer.Start() or gocmd.DeployVerticle())
2. Car part sends signal to Managers (via wrapper or manually)
3. Managers receives signal and coordinates (logs, updates metrics, etc.)
4. Managers monitors and coordinates, but car (runtime) does the actual work

Managers → Car Parts (Optional):
1. Managers can send control signals via fuel system (EventBus)
2. Car parts subscribe to Managers control addresses
3. Managers coordinates car systems through control signals
```

## Key Differences from Lifecycle Manager

- **NO Start() method**: Managers doesn't start the car (runtime does)
- **NO Stop() method**: Managers doesn't stop the car (runtime does)
- **NO Wait() method**: Managers doesn't wait (runtime does)
- **YES Configuration**: Managers manages car system settings
- **YES Registry**: Managers registers car parts
- **YES Coordination**: Managers coordinates car systems
- **YES Event Observation**: Managers monitors car parts (receives signals)
- **YES Event Handling**: Managers reacts to car part events
- **YES Composition**: Managers is composed with car (runtime), driver uses both
- **YES Runtime Access**: Car parts access Managers via car wiring (FluxorContext)

## Configuration

```go
config := &managers.Config{
	HTTPAddr:                 ":8080",
	LogLevel:                 "INFO",
	LogJSON:                  false,
	CacheType:                "memory",
	EnableMetrics:            true,
	MetricsPort:              ":9090",
	HeartbeatInterval:        10 * time.Second,
	EnableHeartbeat:          true,
	HeartbeatEventBusAddress: "managers.heartbeat",
}
```

## Heartbeat System

Managers provides a comprehensive heartbeat system for health monitoring and coordination:

### Features

1. **Periodic Heartbeat Events**: Managers emits periodic heartbeat events (default: every 10 seconds)
2. **EventBus Integration**: Heartbeats are published to EventBus for decoupled consumption
3. **Component Tracking**: Managers tracks component heartbeats and detects missed heartbeats
4. **Health Monitoring**: Components can send heartbeats to Managers for health monitoring

### Usage

```go
// Start heartbeat system
managersInstance.StartHeartbeat()

// Register heartbeat event handlers
managersInstance.OnHeartbeat(func(componentName string, event managers.ComponentEvent) {
    logger.Info("Managers heartbeat", "component", componentName)
})

// Register missed heartbeat handler for a component
managersInstance.OnHeartbeatMissed("my-component", func(componentName string, event managers.ComponentEvent) {
    logger.Warn("Component missed heartbeat", "component", componentName)
})

// Component sends heartbeat
managersInstance.SendHeartbeat("my-component")

// Subscribe to heartbeat messages on EventBus
consumer := eventBus.Consumer("managers.heartbeat")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var heartbeatEvent managers.HeartbeatEvent
    msg.DecodeBody(&heartbeatEvent)
    logger.Info("Heartbeat received", "alive", heartbeatEvent.ManagersAlive, "timestamp", heartbeatEvent.Timestamp)
    return nil
})
```

### Configuration

- `HeartbeatInterval`: Interval between heartbeats (default: 10 seconds)
- `EnableHeartbeat`: Enable/disable heartbeat system (default: true)
- `HeartbeatEventBusAddress`: EventBus address for heartbeat messages (default: "managers.heartbeat")

### Heartbeat Events

- `ComponentHeartbeat`: Emitted periodically by Managers (component name: "managers")
- `ComponentHeartbeatMissed`: Emitted when a component misses its heartbeat (threshold: 3 * interval)

## Notes

- **Car Analogy**: Runtime = car structure, Managers = car's control unit, Components = car parts
- Managers is a control plane, not a lifecycle manager (doesn't build or move the car)
- Managers is composed with GoCMD (car + control unit work together)
- Components communicate with Managers through FluxorContext (car wiring)
- Components are created/configured by Managers but started/stopped externally (by car/runtime)
- Managers receives signals/events when components start/stop (monitors car parts)
- Managers can react to events (coordinate, log, update metrics)
- Managers provides unified configuration and component coordination
- Thread-safe component registry and event handling
- Fail-fast validation (invalid config returns error immediately)
- Managers requires GoCMD to access EventBus (car needs fuel system)
- HTTP server wrapper sends signals to Managers on start/stop (engine signals Managers)
- Managers reference stored in FluxorContext.Config() for component access (Managers accessible via car wiring)
- Heartbeat system provides health monitoring and coordination capabilities
- Heartbeat stops automatically when GoCMD context is cancelled

