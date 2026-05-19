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
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
	"github.com/fluxorio/fluxor/pkg/queue"
)

// QueueRPCClientVerticle demonstrates RPC client usage
type QueueRPCClientVerticle struct {
	*core.BaseVerticle
	queueComp *queue.QueueComponent
	cache     cache.Cache
	client    *queue.RPCClient
}

func NewQueueRPCClientVerticle() *QueueRPCClientVerticle {
	return &QueueRPCClientVerticle{
		BaseVerticle: core.NewBaseVerticle("queue-rpc-client"),
	}
}

func (v *QueueRPCClientVerticle) Start(ctx core.FluxorContext) error {
	// Call base Start first to setup event loop
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	logger := core.NewDefaultLogger()
	logger.Info("QueueRPCClientVerticle Started")

	// Setup cache (not used for client, but required by RPCClient)
	v.cache = cache.NewMemoryCache()

	// Setup RabbitMQ connection
	queueConfig := queue.DefaultConfig()

	// Support connection URL (e.g., amqp://user:pass@host:port/vhost)
	if url := getEnv("RABBITMQ_URL", ""); url != "" {
		queueConfig.URL = url
		logger.Info(fmt.Sprintf("Using RabbitMQ URL: %s", maskURL(url)))
	} else {
		// Use individual configuration options
		queueConfig.Host = getEnv("RABBITMQ_HOST", "localhost")
		if portStr := getEnv("RABBITMQ_PORT", ""); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
				queueConfig.Port = port
			} else {
				queueConfig.Port = 5672
			}
		} else {
			queueConfig.Port = 5672
		}
		queueConfig.Username = getEnv("RABBITMQ_USER", "guest")
		queueConfig.Password = getEnv("RABBITMQ_PASS", "guest")
		queueConfig.VHost = getEnv("RABBITMQ_VHOST", "/")

		// Connection timeout
		if timeoutStr := getEnv("RABBITMQ_TIMEOUT", ""); timeoutStr != "" {
			if timeout, err := time.ParseDuration(timeoutStr); err == nil {
				queueConfig.ConnectionTimeout = timeout
			}
		}

		logger.Info(fmt.Sprintf("Connecting to RabbitMQ at %s:%d (vhost: %s, user: %s)",
			queueConfig.Host, queueConfig.Port, queueConfig.VHost, queueConfig.Username))
	}

	// TLS configuration
	if tlsEnabled := getEnv("RABBITMQ_TLS", ""); tlsEnabled == "true" || tlsEnabled == "1" {
		queueConfig.TLS = &queue.TLSConfig{
			Enabled:            true,
			InsecureSkipVerify: getEnv("RABBITMQ_TLS_INSECURE", "false") == "true",
		}
		logger.Info("TLS enabled for RabbitMQ connection")
	}

	v.queueComp = queue.NewQueueComponent(queueConfig)
	if err := v.queueComp.Start(ctx); err != nil {
		return fmt.Errorf("failed to start queue component: %w", err)
	}

	// Test connection
	testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := v.queueComp.Ping(testCtx); err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	logger.Info("✅ Successfully connected to RabbitMQ")

	// Create RPC client
	conn := v.queueComp.Connection()
	replyQueue := "queue_reply"
	client, err := queue.NewRPCClient(conn, v.cache, replyQueue, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to create RPC client: %w", err)
	}
	v.client = client

	logger.Info(fmt.Sprintf("RPC Client started, reply queue: %s", replyQueue))

	// Make test RPC calls
	go v.makeTestRPCCalls(ctx)

	return nil
}

func (v *QueueRPCClientVerticle) makeTestRPCCalls(ctx core.FluxorContext) {
	logger := core.NewDefaultLogger()

	// Wait a bit for server to be ready
	time.Sleep(2 * time.Second)

	// Test cache keys
	testKeys := []string{"user:123", "config:app", "data:test", "not:found"}

	for _, cacheKey := range testKeys {
		logger.Info(fmt.Sprintf("Making RPC call for cache key: %s", cacheKey))

		req := queue.RPCRequest{
			CacheKey: cacheKey,
			Data: map[string]interface{}{
				"action": "get_info",
			},
		}

		response, err := v.client.Call(ctx.Context(), "queue_ha_core", req)
		if err != nil {
			logger.Error(fmt.Sprintf("RPC call failed for %s: %v", cacheKey, err))
			continue
		}

		if response.Success {
			logger.Info(fmt.Sprintf("✅ RPC Success for %s: %s", cacheKey, string(response.Data)))
		} else {
			logger.Info(fmt.Sprintf("❌ RPC Error for %s: %s", cacheKey, response.Error))
		}

		// Wait between calls
		time.Sleep(1 * time.Second)
	}

	logger.Info("Test RPC calls completed")
}

func (v *QueueRPCClientVerticle) Stop(ctx core.FluxorContext) error {
	logger := core.NewDefaultLogger()
	logger.Info("QueueRPCClientVerticle Stopped")

	if v.client != nil {
		if err := v.client.Close(); err != nil {
			logger.Error(fmt.Sprintf("Error closing RPC client: %v", err))
		}
	}

	if v.queueComp != nil {
		if err := v.queueComp.Stop(ctx); err != nil {
			logger.Error(fmt.Sprintf("Error stopping queue component: %v", err))
		}
	}

	return v.BaseVerticle.Stop(ctx)
}

func main() {
	logger := core.NewDefaultLogger()
	logger.Info("Starting Queue RPC Client...")

	// Load .env.local file if it exists
	if err := loadEnvLocal(); err != nil {
		logger.Info(fmt.Sprintf("Note: Could not load .env.local: %v (using environment variables or defaults)", err))
	}

	// Create runtime
	app, err := entrypoint.NewMainVerticle("")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create runtime: %v", err))
		os.Exit(1)
	}

	// Deploy QueueRPCClientVerticle
	logger.Info("Deploying QueueRPCClientVerticle...")
	_, err = app.DeployVerticle(NewQueueRPCClientVerticle())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deploy QueueRPCClientVerticle: %v", err))
		os.Exit(1)
	}
	logger.Info("QueueRPCClientVerticle deployed successfully")

	// Start application
	logger.Info("Starting application...")
	if err := app.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to start application: %v", err))
		os.Exit(1)
	}

	logger.Info("Queue RPC Client started successfully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// maskURL masks sensitive information in RabbitMQ URL for logging
func maskURL(url string) string {
	// Simple masking: replace password with ***
	// Format: amqp://user:pass@host:port/vhost
	if idx := strings.Index(url, "@"); idx > 0 {
		parts := strings.Split(url[:idx], "://")
		if len(parts) == 2 {
			userPass := parts[1]
			if colonIdx := strings.Index(userPass, ":"); colonIdx > 0 {
				user := userPass[:colonIdx]
				return fmt.Sprintf("%s://%s:***@%s", parts[0], user, url[idx+1:])
			}
		}
	}
	return url
}

// findProjectRoot finds the project root by looking for go.mod file
func findProjectRoot() (string, error) {
	// Start from current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	dir := wd
	for {
		// Check if go.mod exists in current directory
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	// If go.mod not found, try from executable directory
	execDir := filepath.Dir(os.Args[0])
	if !filepath.IsAbs(execDir) {
		execDir, _ = filepath.Abs(execDir)
	}

	dir = execDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fallback to current working directory
	return wd, nil
}

// loadEnvLocal loads environment variables from .env.local file in the project root
func loadEnvLocal() error {
	// Find project root
	rootDir, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Look for .env.local in project root
	envPath := filepath.Join(rootDir, ".env.local")
	envFile, err := os.Open(envPath)
	if err != nil {
		return fmt.Errorf(".env.local file not found at %s: %w", envPath, err)
	}
	defer envFile.Close()

	scanner := bufio.NewScanner(envFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove quotes if present
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}

			// Only set if not already in environment
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .env.local: %w", err)
	}

	return nil
}
