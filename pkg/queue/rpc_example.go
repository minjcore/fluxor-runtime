package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
)

// ExampleRPCUsage demonstrates RPC pattern with queue_ha_core and cache
func ExampleRPCUsage() {
	// 1. Setup cache
	memCache := cache.NewMemoryCache()
	ctx := context.Background()

	// Store some data in cache
	cacheKey := "user:123"
	userData := []byte(`{"id": 123, "name": "John Doe", "email": "john@example.com"}`)
	if err := memCache.Set(ctx, cacheKey, userData, 10*time.Minute); err != nil {
		log.Fatalf("Failed to set cache: %v", err)
	}

	// 2. Setup RabbitMQ connection
	config := DefaultConfig()
	config.Host = "localhost"
	config.Port = 5672

	conn, err := NewConnection(config)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// 3. Create RPC Server (consumes from queue_ha_core)
	requestQueue := "queue_ha_core"
	server, err := NewRPCServer(conn, memCache, requestQueue)
	if err != nil {
		log.Fatalf("Failed to create RPC server: %v", err)
	}
	defer server.Close()

	// Start server in goroutine
	go func() {
		serverCtx := context.Background()
		if err := server.Start(serverCtx); err != nil {
			log.Printf("RPC server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(1 * time.Second)

	// 4. Create RPC Client (sends to queue_ha_core, consumes from reply queue)
	replyQueue := "queue_reply"
	client, err := NewRPCClient(conn, memCache, replyQueue, 30*time.Second)
	if err != nil {
		log.Fatalf("Failed to create RPC client: %v", err)
	}
	defer client.Close()

	// 5. Make RPC call
	req := RPCRequest{
		CacheKey: cacheKey,
		Data: map[string]interface{}{
			"action": "get_user_info",
		},
	}

	response, err := client.Call(ctx, requestQueue, req)
	if err != nil {
		log.Fatalf("RPC call failed: %v", err)
	}

	// 6. Process response
	if response.Success {
		fmt.Printf("RPC Success! Data from cache: %s\n", string(response.Data))
	} else {
		fmt.Printf("RPC Error: %s\n", response.Error)
	}
}

// ExampleRPCWithComponent demonstrates RPC using QueueComponent
func ExampleRPCWithComponent() {
	// 1. Setup cache
	memCache := cache.NewMemoryCache()
	ctx := context.Background()

	// Store data in cache
	cacheKey := "config:app"
	configData := []byte(`{"app_name": "fluxor", "version": "1.0.0"}`)
	if err := memCache.Set(ctx, cacheKey, configData, 5*time.Minute); err != nil {
		log.Fatalf("Failed to set cache: %v", err)
	}

	// 2. Create Queue Component
	config := DefaultConfig()
	config.Host = "localhost"
	config.Port = 5672

	// In real usage, deploy to Vertx:
	// queueComp := NewQueueComponent(config)
	// vertx := core.NewVertx()
	// vertx.Deploy(queueComp)
	// queueComp.Start(ctx)

	// For example, we'll use connection directly
	conn, err := NewConnection(config)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// 3. Start RPC Server
	requestQueue := "queue_ha_core"
	server, err := NewRPCServer(conn, memCache, requestQueue)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	go func() {
		serverCtx := context.Background()
		if err := server.Start(serverCtx); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	// 4. Create RPC Client
	replyQueue := "queue_reply"
	client, err := NewRPCClient(conn, memCache, replyQueue, 30*time.Second)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 5. Make RPC call
	req := RPCRequest{
		CacheKey: cacheKey,
	}

	response, err := client.Call(ctx, requestQueue, req)
	if err != nil {
		log.Fatalf("RPC call failed: %v", err)
	}

	if response.Success {
		fmt.Printf("Got data from cache: %s\n", string(response.Data))
	} else {
		fmt.Printf("Error: %s\n", response.Error)
	}
}
