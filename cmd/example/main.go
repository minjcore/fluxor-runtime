// Copyright (c) 2024-2026 Fluxor Framework
// All rights reserved.
//
// This source code is proprietary and confidential.
// Unauthorized copying, modification, distribution, or use of this software,
// via any medium is strictly prohibited without the express written permission
// of Fluxor Framework.
//
// This code is provided as an example for demonstration purposes only.
// Redistribution or sharing of this source code is not permitted.
//
// License: Proprietary - All Rights Reserved
// For licensing inquiries, please contact: caokhang91@gmail.com

// Package main demonstrates Sugar Dev (fx) pattern - Dev-Friendly & AI-Friendly
//
// 🎯 PATTERN: Sugar Dev (fx) - Function-based, Express.js-like syntax
// 💡 BEST FOR: Rapid prototyping, simple applications, developers coming from Node.js
// 🤖 AI-FRIENDLY: Clear function structure, easy to understand, well-documented
//
// Key Features:
//   - Function-based setup (no structs needed!)
//   - Express.js-like syntax (router.GETFast())
//   - Simple dependency injection (automatic via fx)
//   - Easy to read and understand
//
// Pattern Comparison:
//   - This file (Sugar Dev): Function-based, simpler, faster to write
//   - main.go.backup (BaseVerticle): Struct-based, enterprise pattern, better for complex apps
//
// What This Example Demonstrates:
//   1. EventBus publish/subscribe pattern
//   2. HTTP server with routes
//   3. Cross-cutting concerns (HTTP → EventBus)
//   4. Scaling (multiple service instances)
//   5. Graceful shutdown
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fx"
	"github.com/fluxorio/fluxor/pkg/web"
)

// setupPingService sets up a ping service that publishes messages to EventBus
//
// 🎯 Purpose: Demonstrates EventBus publish pattern in Sugar Dev (fx) style
// 💡 Why Sugar Dev: No structs needed - just functions! Perfect for rapid prototyping
// 🤖 AI-Friendly: Clear function name, single responsibility, easy to understand
//
// Pattern: Function-based setup (Express.js-like) - much simpler than Verticle pattern
// Dependencies: Automatically injected via fx dependency injection
func setupPingService(deps map[reflect.Type]interface{}) error {
	// Extract dependencies from fx dependency injection map
	// This is the Sugar Dev pattern - no manual wiring needed!
	eventBus := deps[reflect.TypeOf((*core.EventBus)(nil)).Elem()].(core.EventBus)
	gocmd := deps[reflect.TypeOf((*core.GoCMD)(nil)).Elem()].(core.GoCMD)

	log.Println("🚀 Ping service starting...")

	// Start background goroutine for periodic pings
	// 💡 Why goroutine here? This is I/O-bound async work (EventBus publish)
	//    In Sugar Dev pattern, we use plain goroutines for simplicity
	//    (In Verticle pattern, we'd use ExecuteOn() for better lifecycle management)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-gocmd.Context().Done():
				// Graceful shutdown: stop when context is cancelled
				log.Println("Ping service: context cancelled, stopping")
				return
			case <-ticker.C:
				// Publish ping message every 2 seconds
				msg := fmt.Sprintf("PING at %s", time.Now().Format(time.RFC3339))
				if err := eventBus.Publish("ping-topic", msg); err != nil {
					log.Printf("❌ Failed to publish ping: %v", err)
				} else {
					log.Printf("📤 Published: %s", msg)
				}
			}
		}
	}()

	return nil
}

// setupPongService sets up a pong service that subscribes to ping messages
//
// 🎯 Purpose: Demonstrates EventBus consumer/subscribe pattern in Sugar Dev style
// 💡 Why Sugar Dev: Simple function - no struct, no lifecycle management needed
// 🤖 AI-Friendly: Clear subscription pattern, easy to replicate for other topics
//
// Pattern: EventBus consumer with handler function - reactive message processing
// Scaling: You can deploy multiple instances of this service (see main function)
func setupPongService(deps map[reflect.Type]interface{}) error {
	// Extract EventBus from dependency injection
	eventBus := deps[reflect.TypeOf((*core.EventBus)(nil)).Elem()].(core.EventBus)

	log.Println("🎯 Pong service starting...")

	// Subscribe to ping-topic and register handler
	// 💡 This is reactive programming - handler is called automatically when message arrives
	//    Handler runs on EventBus executor (concurrent, not blocking reactor)
	consumer := eventBus.Consumer("ping-topic")
	consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
		payload := msg.Body()
		log.Printf("📥 Received ping: %v", payload)
		// Return nil = success, handler can return error to signal failure
		return nil
	})

	return nil
}

// setupHTTPServer sets up HTTP server with routes
//
// 🎯 Purpose: Demonstrates HTTP server setup and EventBus integration from HTTP handlers
// 💡 Why Sugar Dev: Express.js-like syntax - router.GETFast() is very intuitive
// 🤖 AI-Friendly: Clear route registration pattern, easy to add more routes
//
// Pattern: FastHTTP server with router - high performance (100k+ RPS target)
// Integration: HTTP handlers can publish to EventBus - demonstrates cross-cutting concerns
func setupHTTPServer(deps map[reflect.Type]interface{}) error {
	// Extract dependencies
	gocmd := deps[reflect.TypeOf((*core.GoCMD)(nil)).Elem()].(core.GoCMD)
	eventBus := deps[reflect.TypeOf((*core.EventBus)(nil)).Elem()].(core.EventBus)

	log.Println("🌐 HTTP server starting...")

	// Create HTTP server config with default settings
	// 💡 Default config targets 100k+ RPS performance
	cfg := web.DefaultFastHTTPServerConfig(":8080")
	server := web.NewFastHTTPServer(gocmd, cfg)
	router := server.FastRouter()

	// Register routes - Express.js-like syntax!
	// 🤖 AI Note: Each route handler receives FastRequestContext for request/response
	router.GETFast("/", func(c *web.FastRequestContext) error {
		return c.JSON(200, fx.JSON{
			"message": "Hello from Fluxor Example!",
			"pattern": "Sugar Dev (fx)",
			"version": "1.0.0",
		})
	})

	// Health check endpoint - simple ping/pong
	router.GETFast("/api/ping", func(c *web.FastRequestContext) error {
		return c.JSON(200, fx.JSON{
			"status": "ok",
			"time":   time.Now().Unix(),
		})
	})

	// EventBus integration from HTTP handler - demonstrates cross-cutting pattern
	// 💡 This shows how HTTP can trigger EventBus messages (useful for webhooks, etc.)
	router.GETFast("/api/eventbus/publish", func(c *web.FastRequestContext) error {
		// Publish message to EventBus from HTTP handler
		msg := fmt.Sprintf("HTTP triggered message at %s", time.Now().Format(time.RFC3339))
		if err := eventBus.Publish("ping-topic", msg); err != nil {
			return c.JSON(500, fx.JSON{
				"error": fmt.Sprintf("Failed to publish: %v", err),
			})
		}
		return c.JSON(200, fx.JSON{
			"message": "Published to EventBus",
			"payload": msg,
		})
	})

	// Info endpoint - returns application metadata
	router.GETFast("/api/info", func(c *web.FastRequestContext) error {
		return c.JSON(200, fx.JSON{
			"message": "Fluxor Example Application",
			"pattern": "Sugar Dev (fx) - Dev-Friendly & AI-Friendly",
			"features": []string{
				"Function-based setup (no structs!)",
				"EventBus Publish/Subscribe",
				"HTTP Server (100k+ RPS)",
				"Express.js-like syntax",
				"Simple dependency injection",
				"AI-friendly code structure",
			},
			"timestamp": time.Now().Unix(),
		})
	})

	// Start server in goroutine (non-blocking)
	// 💡 Why goroutine? server.Start() is blocking I/O - must not block setup function
	//    In Sugar Dev pattern, we use plain goroutines for simplicity
	//    (In Verticle pattern, we'd use ExecuteOn() for better lifecycle management)
	go func() {
		log.Println("🌐 HTTP server listening on :8080")
		log.Println("📋 Available endpoints:")
		log.Println("   GET / - Hello message")
		log.Println("   GET /api/ping - Health check")
		log.Println("   GET /api/eventbus/publish - Publish to EventBus")
		log.Println("   GET /api/info - System information")
		if err := server.Start(); err != nil {
			log.Printf("❌ HTTP server error: %v", err)
		}
	}()

	return nil
}

// Main function - demonstrates Sugar Dev (fx) pattern
//
// 🎯 Purpose: Entry point demonstrating Sugar Dev pattern (function-based, Express.js-like)
// 💡 Why Sugar Dev: Much simpler than Verticle pattern - just functions, no structs!
// 🤖 AI-Friendly: Clear structure, easy to understand flow, well-documented
//
// Pattern Comparison:
//   - Sugar Dev (fx): Function-based, Express.js-like, perfect for rapid prototyping
//   - Verticle: Struct-based, enterprise pattern, better for complex applications
//
// This example shows:
//   1. EventBus publish/subscribe
//   2. HTTP server with routes
//   3. Cross-cutting concerns (HTTP → EventBus)
//   4. Scaling (multiple pong service instances)
func main() {
	log.Println("✨ Starting Fluxor Example (Sugar Dev Pattern)")
	log.Println("💡 This example uses fx package for better Dev UX")
	log.Println("🤖 Code is optimized for both developers and AI assistants")

	// Create context for application lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create application using fx (Sugar Dev pattern)
	// Express.js-like syntax - much simpler than Verticle pattern!
	// 💡 fx.New() automatically handles dependency injection
	// 🤖 AI Note: Each fx.Invoke() registers a setup function that runs at startup
	app, err := fx.New(ctx,
		// Setup ping service (publishes messages to EventBus)
		// 💡 This service runs in background, publishing every 2 seconds
		fx.Invoke(fx.NewInvoker(setupPingService)),
		
		// Setup pong service (subscribes to ping messages)
		// 💡 This service reacts to ping messages and logs them
		fx.Invoke(fx.NewInvoker(setupPongService)),
		
		// Setup another pong service instance (demonstrates scaling)
		// 💡 Multiple instances = load distribution (both receive same messages)
		// 🤖 AI Note: This shows how easy it is to scale services in Fluxor
		fx.Invoke(fx.NewInvoker(setupPongService)),
		
		// Setup HTTP server with routes
		// 💡 HTTP server can also publish to EventBus (see /api/eventbus/publish route)
		fx.Invoke(fx.NewInvoker(setupHTTPServer)),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create app: %v", err)
	}

	// Start application
	// 💡 This initializes all services registered above
	if err := app.Start(); err != nil {
		log.Fatalf("❌ Failed to start app: %v", err)
	}

	log.Println("✅ All services started successfully!")
	log.Println("🛑 Press Ctrl+C to stop")
	log.Println("📖 Try: curl http://localhost:8080/api/info")

	// Handle graceful shutdown
	// 💡 Listen for SIGINT (Ctrl+C) or SIGTERM (k8s termination signal)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sig
	log.Println("\n🛑 Shutting down gracefully...")

	// Stop application (cleanup all services)
	if err := app.Stop(); err != nil {
		log.Printf("❌ Error stopping app: %v", err)
	}

	log.Println("👋 Goodbye!")
}