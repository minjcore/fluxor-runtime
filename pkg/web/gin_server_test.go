package web

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewGinHTTPServer(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	defer func() { _ = gocmd.Close() }()

	config := DefaultGinHTTPServerConfig(":0")
	server := NewGinHTTPServer(gocmd, config)
	if server == nil {
		t.Fatal("NewGinHTTPServer returned nil")
	}
	if server.GinRouter() == nil {
		t.Fatal("GinRouter() returned nil")
	}
	if server.Engine() == nil {
		t.Fatal("Engine() returned nil")
	}
	if server.Router() == nil {
		t.Fatal("Router() returned nil")
	}
}

func TestGinHTTPServerReadyAfterStart(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	defer func() { _ = gocmd.Close() }()

	s := NewGinHTTPServer(gocmd, DefaultGinHTTPServerConfig("127.0.0.1:0"))
	select {
	case <-s.Ready():
		t.Fatal("Ready() should not be closed before Start()")
	default:
	}

	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = s.Stop() }()

	select {
	case <-s.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("Ready() was not closed after successful listen")
	}
}
