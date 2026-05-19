package ws

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/gorilla/websocket"
)

// ConnectionHandler handles a WebSocket connection
type ConnectionHandler func(conn *Connection) error

// Server represents a WebSocket server
type Server interface {
	// Start starts the server
	Start() error

	// Stop stops the server gracefully
	Stop() error

	// SetHandler sets the connection handler
	SetHandler(handler ConnectionHandler)

	// HandleWebSocket is an HTTP handler for WebSocket upgrades
	HandleWebSocket(w http.ResponseWriter, r *http.Request)
}

// WebSocketServer implements a WebSocket server
type WebSocketServer struct {
	*core.BaseServer

	config   *ServerConfig
	upgrader websocket.Upgrader

	mu          sync.RWMutex
	connections map[string]*Connection
	nextID      int64

	handler  ConnectionHandler
	bridge   *core.WebSocketEventBusBridge
	eventBus core.EventBus

	server   *http.Server
	mux      *http.ServeMux
	stopping int32

	// Worker pool and mailbox for connection handling
	connMailbox      concurrency.Mailbox
	executor         concurrency.Executor
	workers          int
	maxQueue         int
	backpressure     *web.BackpressureController
	startWorkersOnce sync.Once

	// Metrics (atomic for thread-safety)
	queuedConnections   int64
	rejectedConnections int64
	totalAccepted       int64
	handledConnections  int64
	errorConnections    int64
	activeConns         int64
}

// NewServer creates a new WebSocket server
func NewServer(gocmd core.GoCMD, config *ServerConfig) *WebSocketServer {
	if config == nil {
		config = DefaultServerConfig(":8080", "/ws")
	}
	if config.Addr == "" {
		config.Addr = ":8080"
	}
	if config.Path == "" {
		config.Path = "/ws"
	}
	if config.MaxQueue < 1 {
		config.MaxQueue = 100
	}
	if config.Workers < 1 {
		config.Workers = 1
	}
	if config.MaxConnections < 0 {
		config.MaxConnections = 0
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  config.ReadBufferSize,
		WriteBufferSize: config.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			if config.CheckOrigin != nil {
				return config.CheckOrigin(r)
			}
			return true // Allow all origins by default
		},
	}

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:              config.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second, // Mitigate Slowloris attacks
	}

	// Setup worker pool and mailbox
	normalCapacity := config.MaxQueue + config.Workers
	connMailbox := concurrency.NewBoundedMailbox(config.MaxQueue)
	executor := concurrency.NewExecutor(gocmd.Context(), concurrency.ExecutorConfig{
		Workers:   config.Workers,
		QueueSize: config.MaxQueue,
	})

	ws := &WebSocketServer{
		BaseServer:   core.NewBaseServer("websocket-server", gocmd),
		config:       config,
		upgrader:     upgrader,
		connections:  make(map[string]*Connection),
		handler:      defaultConnectionHandler,
		server:       server,
		mux:          mux,
		connMailbox:  connMailbox,
		executor:     executor,
		workers:      config.Workers,
		maxQueue:     config.MaxQueue,
		backpressure: web.NewBackpressureController(normalCapacity, 60),
	}

	// Wire BaseServer hooks
	ws.BaseServer.SetHooks(ws.doStart, ws.doStop)

	// Register WebSocket handler
	mux.HandleFunc(config.Path, ws.HandleWebSocket)

	// Start workers once
	ws.startConnWorkers()

	return ws
}

// NewServerWithEventBus creates a new WebSocket server with EventBus integration
func NewServerWithEventBus(gocmd core.GoCMD, config *ServerConfig, eventBus core.EventBus) *WebSocketServer {
	ws := NewServer(gocmd, config)
	ws.eventBus = eventBus
	ws.bridge = core.NewWebSocketEventBusBridge(eventBus)
	return ws
}

func defaultConnectionHandler(conn *Connection) error {
	// Default: do nothing. Connection will be closed by server.
	return nil
}

// SetHandler sets the connection handler (fail-fast on nil)
func (ws *WebSocketServer) SetHandler(handler ConnectionHandler) {
	if handler == nil {
		panic("websocket handler cannot be nil")
	}
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.handler = handler
}

// HandleWebSocket handles WebSocket upgrade and connection
func (ws *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// If EventBus bridge is enabled, use it (bypasses worker pool)
	if ws.bridge != nil {
		ws.bridge.HandleWebSocket(w, r)
		return
	}

	// Check max connections
	if !ws.tryAcquireConnSlot() {
		atomic.AddInt64(&ws.rejectedConnections, 1)
		http.Error(w, "maximum connections reached", http.StatusServiceUnavailable)
		return
	}

	// Upgrade connection
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.releaseConnSlot()
		ws.Logger().Error("WebSocket upgrade failed", "error", err)
		return
	}

	atomic.AddInt64(&ws.totalAccepted, 1)

	// Queue connection for worker pool processing
	ws.enqueueConn(conn)
}

// onConnectionClose is called when a connection is closed
func (ws *WebSocketServer) onConnectionClose(conn *Connection) {
	ws.mu.Lock()
	delete(ws.connections, conn.ID())
	ws.mu.Unlock()
}

// doStart is called by BaseServer.Start() - implements hook method
func (ws *WebSocketServer) doStart() error {
	// Ensure workers are running
	ws.startConnWorkers()

	ws.Logger().Info(fmt.Sprintf("Starting WebSocket server on %s%s", ws.config.Addr, ws.config.Path))
	err := ws.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		ws.Logger().Error(fmt.Sprintf("WebSocket server error: %v", err))
	}
	return err
}

// doStop is called by BaseServer.Stop() - implements hook method
func (ws *WebSocketServer) doStop() error {
	atomic.StoreInt32(&ws.stopping, 1)

	// Close all connections
	ws.mu.Lock()
	for _, conn := range ws.connections {
		conn.Close()
	}
	ws.connections = make(map[string]*Connection)
	ws.mu.Unlock()

	// Close mailbox and shutdown executor
	ws.connMailbox.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ws.executor.Shutdown(ctx); err != nil {
		return err
	}

	// Shutdown HTTP server
	return ws.server.Shutdown(ctx)
}

// Connections returns the current number of active connections
func (ws *WebSocketServer) Connections() int {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return len(ws.connections)
}

// startConnWorkers starts worker goroutines for connection processing
func (ws *WebSocketServer) startConnWorkers() {
	ws.startWorkersOnce.Do(func() {
		for i := 0; i < ws.workers; i++ {
			task := concurrency.NewNamedTask(
				fmt.Sprintf("ws-worker-%d", i),
				func(ctx context.Context) error {
					return ws.processConnFromMailbox(ctx)
				},
			)
			if err := ws.executor.Submit(task); err != nil {
				ws.Logger().Error(fmt.Sprintf("failed to start ws worker %d: %v", i, err))
			}
		}
	})
}

// processConnFromMailbox processes connections from the mailbox
func (ws *WebSocketServer) processConnFromMailbox(ctx context.Context) error {
	for {
		msg, err := ws.connMailbox.Receive(ctx)
		if err != nil {
			return err
		}

		conn, ok := msg.(*websocket.Conn)
		if !ok || conn == nil {
			// Fail-fast: unexpected mailbox payload
			ws.backpressure.Release()
			ws.releaseConnSlot()
			continue
		}

		atomic.AddInt64(&ws.queuedConnections, -1)
		atomic.AddInt64(&ws.handledConnections, 1)

		// Generate connection ID
		id := fmt.Sprintf("conn-%d", atomic.AddInt64(&ws.nextID, 1))

		// Create connection wrapper
		wsConn := newConnection(conn, id, ws.onConnectionClose, nil)

		// Register connection
		ws.mu.Lock()
		ws.connections[id] = wsConn
		ws.mu.Unlock()

		// Set up ping/pong
		conn.SetReadDeadline(time.Now().Add(ws.config.PongWait))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(ws.config.PongWait))
			return nil
		})

		// Start ping ticker
		ticker := time.NewTicker(ws.config.PingPeriod)

		// Handle connection with panic isolation
		func() {
			defer func() {
				ticker.Stop()
				wsConn.Close()
				ws.backpressure.Release()
				ws.releaseConnSlot()
			}()

			// Ping loop
			go func() {
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(ws.config.WriteDeadline)); err != nil {
							return
						}
					case <-wsConn.Context().Done():
						return
					}
				}
			}()

			// Call handler with panic isolation
			if err := func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt64(&ws.errorConnections, 1)
						ws.Logger().Error(fmt.Sprintf("panic in ws handler (isolated): %v", r))
						err = fmt.Errorf("panic: %v", r)
					}
				}()
				return ws.handler(wsConn)
			}(); err != nil {
				atomic.AddInt64(&ws.errorConnections, 1)
				ws.Logger().Error("connection handler error", "error", err, "id", id)
			}
		}()
	}
}

// enqueueConn queues a connection for worker processing
func (ws *WebSocketServer) enqueueConn(conn *websocket.Conn) {
	// Backpressure baseline: reject immediately when normal capacity exceeded
	if !ws.backpressure.TryAcquire() {
		atomic.AddInt64(&ws.rejectedConnections, 1)
		ws.releaseConnSlot()
		conn.Close()
		return
	}

	// Bounded queue: fail-fast when queue is full
	if err := ws.connMailbox.Send(conn); err != nil {
		ws.backpressure.Release()
		atomic.AddInt64(&ws.rejectedConnections, 1)
		ws.releaseConnSlot()
		conn.Close()
		return
	}

	atomic.AddInt64(&ws.queuedConnections, 1)
}

// tryAcquireConnSlot attempts to acquire a connection slot
func (ws *WebSocketServer) tryAcquireConnSlot() bool {
	if ws.config.MaxConnections <= 0 {
		atomic.AddInt64(&ws.activeConns, 1)
		return true
	}
	for {
		cur := atomic.LoadInt64(&ws.activeConns)
		if int(cur) >= ws.config.MaxConnections {
			return false
		}
		if atomic.CompareAndSwapInt64(&ws.activeConns, cur, cur+1) {
			return true
		}
	}
}

// releaseConnSlot releases a connection slot
func (ws *WebSocketServer) releaseConnSlot() {
	atomic.AddInt64(&ws.activeConns, -1)
}
