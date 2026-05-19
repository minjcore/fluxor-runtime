package ws

import "time"

// ServerConfig configures the WebSocket server
type ServerConfig struct {
	// Addr is the address to listen on (e.g., ":8080")
	Addr string

	// Path is the WebSocket endpoint path (e.g., "/ws")
	Path string

	// ReadBufferSize specifies the size of the read buffer
	ReadBufferSize int

	// WriteBufferSize specifies the size of the write buffer
	WriteBufferSize int

	// ReadDeadline specifies the read deadline
	ReadDeadline time.Duration

	// WriteDeadline specifies the write deadline
	WriteDeadline time.Duration

	// PongWait is the duration to wait for a pong response
	PongWait time.Duration

	// PingPeriod is the duration between pings
	PingPeriod time.Duration

	// MaxConnections is the maximum number of concurrent connections (0 = unlimited)
	MaxConnections int

	// MaxQueue is the maximum queue size for connection handling (bounded queue)
	MaxQueue int

	// Workers is the number of worker goroutines for connection handling
	Workers int

	// CheckOrigin function to check the origin header
	// If nil, all origins are allowed
	CheckOrigin func(r interface{}) bool
}

// DefaultServerConfig returns a sensible default configuration
func DefaultServerConfig(addr, path string) *ServerConfig {
	if addr == "" {
		addr = ":8080"
	}
	if path == "" {
		path = "/ws"
	}
	return &ServerConfig{
		Addr:            addr,
		Path:            path,
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		ReadDeadline:    60 * time.Second,
		WriteDeadline:   10 * time.Second,
		PongWait:        60 * time.Second,
		PingPeriod:      (60 * time.Second * 9) / 10, // 90% of PongWait
		MaxConnections:  0,
		MaxQueue:        1000, // Bounded queue for connection handling
		Workers:         50,   // Worker pool size
		CheckOrigin:     nil,  // Allow all origins by default
	}
}
