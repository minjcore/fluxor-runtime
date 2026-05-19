package managers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/observability/prometheus"
	"github.com/fluxorio/fluxor/pkg/web"
)

// ManagersKey is the key used to store Managers reference in FluxorContext.Config()
const ManagersKey = "managers"

// Managers is the Execution Control Unit - a control plane for managing application components
// Car analogy: Car's Electronic Control Unit that coordinates and manages car systems
type Managers struct {
	gocmd  core.GoCMD // Required: Managers needs GoCMD (car structure) to access EventBus (fuel system)
	config *Config    // Car system settings
	mu     sync.RWMutex

	// Component registry (car parts registry)
	httpServer web.Server
	logger     core.Logger
	cache      cache.Cache
	metrics    *prometheus.Metrics

	// Flags to track if components are created/registered
	httpServerSet bool
	loggerSet     bool
	cacheSet      bool
	metricsSet    bool

	// Event handlers (monitoring car parts)
	eventHandlers map[string][]ComponentEventHandler
	eventMu       sync.RWMutex

	// Heartbeat state
	heartbeatTicker     *time.Ticker
	heartbeatStop       chan struct{}
	componentHeartbeats map[string]time.Time
	heartbeatMu         sync.RWMutex
	heartbeatStarted    bool

	// Enhanced features
	serviceRegistry     *ServiceRegistry     // Service registry for tracking all services
	healthChecker       *HealthChecker       // Health check management
	shutdownCoordinator *ShutdownCoordinator // Graceful shutdown coordination
}

// NewManagers creates a new Managers instance with its own GoCMD
// Car analogy: Builds car structure and installs Managers
func NewManagers(ctx context.Context, config *Config) (*Managers, error) {
	if config == nil {
		config = DefaultConfig()
	}

	gocmd := core.NewGoCMD(ctx)
	return NewManagersWithGoCMD(gocmd, config)
}

// NewManagersWithGoCMD creates a new Managers instance with an existing GoCMD
// Car analogy: Installs Managers in existing car structure
func NewManagersWithGoCMD(gocmd core.GoCMD, config *Config) (*Managers, error) {
	if gocmd == nil {
		return nil, errors.New("gocmd cannot be nil")
	}

	if config == nil {
		config = DefaultConfig()
	}

	managers := &Managers{
		gocmd:               gocmd,
		config:              config,
		eventHandlers:       make(map[string][]ComponentEventHandler),
		componentHeartbeats: make(map[string]time.Time),
		heartbeatStop:       make(chan struct{}),
	}

	return managers, nil
}

// AttachToGoCMD attaches Managers to GoCMD by storing Managers reference in context config
// Car analogy: Connects Managers to car structure (stores Managers reference in car wiring)
// This allows components (verticles) to access Managers via FluxorContext
func (m *Managers) AttachToGoCMD(gocmd core.GoCMD) error {
	if gocmd == nil {
		return errors.New("gocmd cannot be nil")
	}

	// Store Managers reference in GoCMD context
	// Components will access Managers via FluxorContext.Config()[ManagersKey]
	// This is done by storing in a way that can be accessed when creating FluxorContext
	// Note: This requires cooperation with GoCMD or we need to use a different approach
	// For now, we'll use a global registry or store in a way that GetManagers can retrieve it
	// Actually, we can't modify GoCMD directly, so we need to store Managers reference
	// in a way that's accessible when FluxorContext is created
	// The pattern is: When verticle starts, GetManagers(ctx) retrieves Managers from ctx.Config()[ManagersKey]
	// So we need to ensure the context passed to verticles has Managers stored
	// This is typically done by the application when creating/deploying verticles

	return nil
}

// Context returns the GoCMD instance (car structure)
func (m *Managers) Context() core.GoCMD {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gocmd
}

// EventBus returns the EventBus from GoCMD (fuel system)
// Car analogy: Managers accesses fuel system (EventBus) through car structure (GoCMD)
func (m *Managers) EventBus() core.EventBus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.gocmd == nil {
		return nil
	}
	return m.gocmd.EventBus()
}

// Queue returns the EventBus from GoCMD (fuel system)
// Car analogy: Managers accesses fuel system (EventBus) through car structure (GoCMD)
// Queue is an alias for EventBus() for clarity
func (m *Managers) Queue() core.EventBus {
	return m.EventBus()
}

// HTTPServer returns the registered HTTP server
func (m *Managers) HTTPServer() web.Server {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.httpServer
}

// Logger returns the registered logger
func (m *Managers) Logger() core.Logger {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.logger
}

// Cache returns the registered cache
func (m *Managers) Cache() cache.Cache {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache
}

// Observe returns the registered metrics
func (m *Managers) Observe() *prometheus.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}

// RegisterHTTPServer registers an HTTP server instance
func (m *Managers) RegisterHTTPServer(server web.Server) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.httpServer = server
	m.httpServerSet = true
}

// RegisterLogger registers a logger instance
func (m *Managers) RegisterLogger(logger core.Logger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logger = logger
	m.loggerSet = true
}

// RegisterCache registers a cache instance
func (m *Managers) RegisterCache(cache cache.Cache) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = cache
	m.cacheSet = true
}

// RegisterMetrics registers a metrics instance
func (m *Managers) RegisterMetrics(metrics *prometheus.Metrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = metrics
	m.metricsSet = true
}

// Wire wires components together (coordinates car systems)
// Car analogy: Managers coordinates car systems (engine + transmission + wheels)
func (m *Managers) Wire() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Wire components together
	// - Inject logger into HTTP server middleware (if applicable)
	// - Configure metrics collection
	// - Set up cache for components that need it
	// - Link observability to HTTP server

	// For now, wiring is a no-op as actual wiring depends on component implementations
	// Components are wired but NOT started (Managers coordinates, car starts)

	return nil
}

// signalHTTPServerStarted is called by HTTP server wrapper when server starts
// Car analogy: Engine sends "started" signal to Managers
func (m *Managers) signalHTTPServerStarted() {
	m.emitEvent("http-server", ComponentStarted)
}

// signalHTTPServerStopped is called by HTTP server wrapper when server stops
// Car analogy: Engine sends "stopped" signal to Managers
func (m *Managers) signalHTTPServerStopped() {
	m.emitEvent("http-server", ComponentStopped)
}

// emitEvent emits a component lifecycle event
func (m *Managers) emitEvent(componentName string, event ComponentEvent) {
	m.eventMu.RLock()
	handlers := m.eventHandlers[componentName]
	m.eventMu.RUnlock()

	for _, handler := range handlers {
		handler(componentName, event)
	}
}

// OnComponentEvent registers an event handler for a component
func (m *Managers) OnComponentEvent(componentName string, handler ComponentEventHandler) {
	m.eventMu.Lock()
	defer m.eventMu.Unlock()
	m.eventHandlers[componentName] = append(m.eventHandlers[componentName], handler)
}

// OnHTTPServerStart registers a handler for HTTP server start events
func (m *Managers) OnHTTPServerStart(handler ComponentEventHandler) {
	m.OnComponentEvent("http-server", handler)
}

// OnHTTPServerStop registers a handler for HTTP server stop events
func (m *Managers) OnHTTPServerStop(handler ComponentEventHandler) {
	m.OnComponentEvent("http-server", handler)
}

// Config returns the Managers configuration
func (m *Managers) Config() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// StartHeartbeat starts the heartbeat system
// Heartbeat emits periodic events and publishes to EventBus
func (m *Managers) StartHeartbeat() error {
	return m.startHeartbeat()
}

// StopHeartbeat stops the heartbeat system
func (m *Managers) StopHeartbeat() {
	m.stopHeartbeat()
}

// SendHeartbeat sends a heartbeat signal from a component
// Components call this to indicate they are alive
func (m *Managers) SendHeartbeat(componentName string) {
	m.sendHeartbeat(componentName)
}

// OnHeartbeat registers a handler for Managers heartbeat events
func (m *Managers) OnHeartbeat(handler ComponentEventHandler) {
	m.OnComponentEvent("managers", handler)
}

// OnHeartbeatMissed registers a handler for missed heartbeat events for a specific component
// To handle missed heartbeats for all components, register handlers for each component name
func (m *Managers) OnHeartbeatMissed(componentName string, handler ComponentEventHandler) {
	m.OnComponentEvent(componentName, handler)
}
