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

// QueueRPCServerVerticle listens to RPC requests from queue_ha_core
type QueueRPCServerVerticle struct {
	*core.BaseVerticle
	queueComp *queue.QueueComponent
	cache     cache.Cache
	server    *queue.RPCServer
}

func NewQueueRPCServerVerticle() *QueueRPCServerVerticle {
	return &QueueRPCServerVerticle{
		BaseVerticle: core.NewBaseVerticle("queue-rpc-server"),
	}
}

func (v *QueueRPCServerVerticle) Start(ctx core.FluxorContext) error {
	// Call base Start first to setup event loop
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	logger := core.NewDefaultLogger()
	logger.Info("QueueRPCServerVerticle Started")

	// Setup cache
	v.cache = cache.NewMemoryCache()

	// Pre-populate cache with some test data
	testCtx := context.Background()
	testData := map[string][]byte{
		"user:123":   []byte(`{"id": 123, "name": "John Doe", "email": "john@example.com"}`),
		"config:app": []byte(`{"app_name": "fluxor", "version": "1.0.0"}`),
		"data:test":  []byte(`{"key": "test", "value": "test_data"}`),
	}

	for key, data := range testData {
		if err := v.cache.Set(testCtx, key, data, 10*time.Minute); err != nil {
			logger.Error(fmt.Sprintf("Failed to set cache for %s: %v", key, err))
		} else {
			logger.Info(fmt.Sprintf("Pre-populated cache: %s", key))
		}
	}

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
	logger.Info("Starting queue component...")
	if err := v.queueComp.Start(ctx); err != nil {
		logger.Error(fmt.Sprintf("Failed to start queue component: %v", err))
		return fmt.Errorf("failed to start queue component: %w", err)
	}

	// Verify component is actually started
	if !v.queueComp.IsStarted() {
		logger.Error("Queue component Start() returned no error but IsStarted() is false")
		return fmt.Errorf("queue component start incomplete")
	}

	logger.Info("Queue component started, waiting for connection to be ready...")

	// Wait a bit for connection to be established (NewConnection might still be running)
	// Retry ping with backoff
	maxRetries := 10
	retryDelay := 100 * time.Millisecond
	testCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pingErr error
	for i := 0; i < maxRetries; i++ {
		logger.Info(fmt.Sprintf("Pinging RabbitMQ server (attempt %d/%d)...", i+1, maxRetries))
		pingErr = v.queueComp.Ping(testCtx)
		if pingErr == nil {
			logger.Info("✅ Successfully connected to RabbitMQ")
			break
		}
		if i < maxRetries-1 {
			logger.Info(fmt.Sprintf("Connection not ready yet, retrying in %v...", retryDelay))
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}

	if pingErr != nil {
		logger.Error(fmt.Sprintf("Connection test failed after %d attempts: %v", maxRetries, pingErr))
		return fmt.Errorf("failed to connect to RabbitMQ: %w", pingErr)
	}

	// Create RPC server
	logger.Info("Creating RPC server...")
	conn := v.queueComp.Connection()
	requestQueue := "queue_ha_core"
	server, err := queue.NewRPCServer(conn, v.cache, requestQueue)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create RPC server: %v", err))
		return fmt.Errorf("failed to create RPC server: %w", err)
	}
	v.server = server
	logger.Info("RPC server created successfully")

	// Start consuming requests
	logger.Info(fmt.Sprintf("Starting RPC server, consuming from queue: %s", requestQueue))
	go func() {
		serverCtx := context.Background()
		logger.Info("RPC server goroutine started")
		if err := server.Start(serverCtx); err != nil {
			logger.Error(fmt.Sprintf("RPC server error: %v", err))
		}
	}()

	logger.Info(fmt.Sprintf("✅ RPC Server started, consuming from queue: %s", requestQueue))
	logger.Info("✅ QueueRPCServerVerticle ready - listening to RPC requests")
	return nil
}

func (v *QueueRPCServerVerticle) Stop(ctx core.FluxorContext) error {
	logger := core.NewDefaultLogger()
	logger.Info("QueueRPCServerVerticle Stopped")

	if v.server != nil {
		if err := v.server.Close(); err != nil {
			logger.Error(fmt.Sprintf("Error closing RPC server: %v", err))
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
	logger.Info("Starting Queue RPC runtime...")

	// Load .env.local file if it exists
	if err := loadEnvLocal(); err != nil {
		logger.Info(fmt.Sprintf("Note: Could not load .env.local: %v (using environment variables or defaults)", err))
	}

	// Create runtime (simple, no NATS needed for this example)
	app, err := entrypoint.NewMainVerticle("")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create runtime: %v", err))
		os.Exit(1)
	}

	// Deploy QueueRPCServerVerticle
	logger.Info("Deploying QueueRPCServerVerticle...")
	_, err = app.DeployVerticle(NewQueueRPCServerVerticle())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deploy QueueRPCServerVerticle: %v", err))
		os.Exit(1)
	}
	logger.Info("QueueRPCServerVerticle deployed successfully")

	// Start application
	logger.Info("Starting application...")
	if err := app.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to start application: %v", err))
		os.Exit(1)
	}

	logger.Info("Queue RPC runtime started successfully - listening to RPC requests on queue_ha_core")
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

	// Validate path to prevent directory traversal attacks
	if err := validateEnvPath(envPath, rootDir); err != nil {
		return fmt.Errorf("invalid .env.local path: %w", err)
	}

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

// validateEnvPath validates the .env.local file path to prevent directory traversal attacks.
func validateEnvPath(envPath, rootDir string) error {
	// Resolve absolute paths to normalize them
	absEnvPath, err := filepath.Abs(envPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("invalid root directory: %w", err)
	}

	// Clean paths
	cleanEnvPath := filepath.Clean(absEnvPath)
	cleanRootDir := filepath.Clean(absRootDir)

	// Check for directory traversal sequences
	if strings.Contains(cleanEnvPath, "..") {
		return fmt.Errorf("path contains directory traversal sequence")
	}

	// Ensure the env file path is within the root directory
	if !strings.HasPrefix(cleanEnvPath, cleanRootDir) {
		return fmt.Errorf("env file path is outside project root directory")
	}

	return nil
}
