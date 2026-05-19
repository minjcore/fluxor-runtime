// Copyright (c) 2024-2028 Fluxor Framework
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

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/managers"
	"github.com/fluxorio/fluxor/pkg/web"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create GoCMD (car structure)
	gocmd := core.NewGoCMD(ctx)
	log.Println("Created GoCMD instance")

	// Create Managers with default config (control plane)
	config := managers.DefaultConfig()
	config.HTTPAddr = ":8080"
	managersInstance, err := managers.NewManagersWithGoCMD(gocmd, config)
	if err != nil {
		log.Fatalf("Failed to create Managers: %v", err)
	}
	log.Println("Created Managers instance (control plane)")

	// Register event handlers for HTTP server lifecycle events
	// These handlers will be called when the HTTP server starts/stops
	managersInstance.OnHTTPServerStart(func(componentName string, event managers.ComponentEvent) {
		log.Printf("🚀 Managers Event Handler: Component '%s' %s", componentName, event)
		log.Printf("   → HTTP server has started successfully!")
		log.Printf("   → Server is now listening on %s", config.HTTPAddr)
	})

	managersInstance.OnHTTPServerStop(func(componentName string, event managers.ComponentEvent) {
		log.Printf("🛑 Managers Event Handler: Component '%s' %s", componentName, event)
		log.Printf("   → HTTP server has stopped")
		log.Printf("   → Cleanup completed")
	})

	// Also register a general component event handler to demonstrate flexibility
	managersInstance.OnComponentEvent("http-server", func(componentName string, event managers.ComponentEvent) {
		log.Printf("📡 General Event Handler: Component '%s' event: %s", componentName, event)
	})

	// Register heartbeat event handlers
	managersInstance.OnHeartbeat(func(componentName string, event managers.ComponentEvent) {
		log.Printf("💓 Managers Heartbeat: Component '%s' event: %s", componentName, event)
		log.Printf("   → Managers is alive and functioning")
	})

	managersInstance.OnHeartbeatMissed("example-component", func(componentName string, event managers.ComponentEvent) {
		log.Printf("⚠️  Heartbeat Missed: Component '%s' event: %s", componentName, event)
		log.Printf("   → Component may be unhealthy or unresponsive")
	})

	log.Println("Registered event handlers with Managers")

	// Create HTTP server using Managers (this wraps the server to send signals)
	httpServer, err := managersInstance.CreateHTTPServer(gocmd)
	if err != nil {
		log.Fatalf("Failed to create HTTP server: %v", err)
	}
	log.Println("Created HTTP server via Managers (wrapped for signaling)")

	// Get router and setup a simple route
	router := httpServer.Router()
	router.GET("/", func(ctx *web.RequestContext) error {
		return ctx.JSON(200, map[string]interface{}{
			"message": "Hello from Managers-controlled HTTP server!",
			"managers": "control plane active",
			"time":     time.Now().Format(time.RFC3339),
		})
	})

	router.GET("/status", func(ctx *web.RequestContext) error {
		return ctx.JSON(200, map[string]interface{}{
			"status":   "ok",
			"managers": "control plane",
			"server":   "running",
		})
	})

	log.Println("Configured HTTP routes")

	// Register HTTP server with Managers (component registry)
	managersInstance.RegisterHTTPServer(httpServer)
	log.Println("Registered HTTP server with Managers")

	// Wire components (Managers coordinates car systems)
	if err := managersInstance.Wire(); err != nil {
		log.Fatalf("Failed to wire components: %v", err)
	}
	log.Println("Wired components via Managers")

	// Start heartbeat system
	if err := managersInstance.StartHeartbeat(); err != nil {
		log.Fatalf("Failed to start heartbeat: %v", err)
	}
	log.Println("Started Managers heartbeat system")

	// Subscribe to heartbeat messages on EventBus
	eventBus := managersInstance.EventBus()
	if eventBus != nil {
		consumer := eventBus.Consumer(config.HeartbeatEventBusAddress)
		consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
			var heartbeatEvent managers.HeartbeatEvent
			if err := msg.DecodeBody(&heartbeatEvent); err == nil {
				log.Printf("📨 EventBus Heartbeat: Managers alive=%v, timestamp=%s", heartbeatEvent.ManagersAlive, heartbeatEvent.Timestamp.Format(time.RFC3339))
			}
			return nil
		})
		log.Printf("Subscribed to heartbeat events on EventBus address: %s", config.HeartbeatEventBusAddress)
	}

	// Simulate a component sending heartbeats
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				managersInstance.SendHeartbeat("example-component")
				log.Println("📤 Example component sent heartbeat")
			case <-ctx.Done():
				return
			}
		}
	}()
	log.Println("Started example component heartbeat sender")

	// Start HTTP server in a goroutine
	// When Start() is called, the wrapper will signal Managers, triggering event handlers
	log.Println("Starting HTTP server...")
	log.Println("→ This will trigger Managers event handlers when server starts")
	go func() {
		if err := httpServer.Start(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Give server a moment to start and trigger the event
	time.Sleep(100 * time.Millisecond)

	// Wait for interrupt signal
	log.Println("✅ Server is running. Event handlers were triggered!")
	log.Println("   Send HTTP requests to http://localhost:8080/")
	log.Println("   Press Ctrl+C to stop...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("\n🛑 Shutting down...")

	// Stop heartbeat system
	managersInstance.StopHeartbeat()
	log.Println("Stopped Managers heartbeat system")

	// Stop HTTP server (this will trigger the stop event handler)
	if err := httpServer.Stop(); err != nil {
		log.Printf("Error stopping HTTP server: %v", err)
	}

	log.Println("Shutdown complete")
}
