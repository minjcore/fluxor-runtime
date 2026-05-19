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

	"github.com/fluxorio/fluxor/pkg/config"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
	"github.com/fluxorio/fluxor/pkg/fx"
	"github.com/fluxorio/fluxor/pkg/observability/otel"
	"github.com/fluxorio/fluxor/pkg/observability/prometheus"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/health"
	"github.com/fluxorio/fluxor/pkg/web/middleware"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth"
	"github.com/fluxorio/fluxor/pkg/web/middleware/security"
	"github.com/valyala/fasthttp"
)

// Enterprise Application Configuration
type AppConfig struct {
	Server        ServerConfig        `yaml:"server"`
	Database      DatabaseConfig      `yaml:"database"`
	Auth          AuthConfig          `yaml:"auth"`
	Observability ObservabilityConfig `yaml:"observability"`
}

type ServerConfig struct {
	Port               string `yaml:"port"`
	MaxCCU             int    `yaml:"max_ccu"`
	UtilizationPercent int    `yaml:"utilization_percent"`
}

type DatabaseConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Database       string `yaml:"database"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	MaxConnections int    `yaml:"max_connections"`
	MinConnections int    `yaml:"min_connections"`
	MaxIdleTime    int    `yaml:"max_idle_time"`
}

type AuthConfig struct {
	JWTSecret      string   `yaml:"jwt_secret"`
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type ObservabilityConfig struct {
	EnableTracing  bool   `yaml:"enable_tracing"`
	EnableMetrics  bool   `yaml:"enable_metrics"`
	JaegerEndpoint string `yaml:"jaeger_endpoint"`
	PrometheusPort string `yaml:"prometheus_port"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create structured JSON logger
	logger := core.NewJSONLogger()
	logger.Info("Starting enterprise application")

	// Create Fluxor application with dependency injection
	app, err := fx.New(ctx,
		fx.Provide(fx.NewValueProvider(cfg)),
		fx.Provide(fx.NewValueProvider(logger)),
		fx.Invoke(fx.NewInvoker(func(deps map[reflect.Type]interface{}) error {
			return setupEnterpriseApplication(deps, cfg, logger)
		})),
	)
	if err != nil {
		log.Fatalf("Failed to create Fluxor app: %v", err)
	}

	// Start the application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start Fluxor app: %v", err)
	}

	logger.Info("Application started successfully")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutting down gracefully...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := app.Stop(); err != nil {
		logger.Error("Error during shutdown", "error", err)
	}

	<-shutdownCtx.Done()
	logger.Info("Application stopped")
}

func loadConfig() (*AppConfig, error) {
	// Default configuration
	cfg := &AppConfig{
		Server: ServerConfig{
			Port:               ":8080",
			MaxCCU:             5000,
			UtilizationPercent: 67,
		},
		Database: DatabaseConfig{
			Host:           "localhost",
			Port:           5432,
			Database:       "fluxor",
			User:           "fluxor",
			Password:       "password",
			MaxConnections: 100,
			MinConnections: 10,
			MaxIdleTime:    300,
		},
		Auth: AuthConfig{
			JWTSecret:      "your-secret-key-change-in-production",
			AllowedOrigins: []string{"http://localhost:3000"},
		},
		Observability: ObservabilityConfig{
			EnableTracing:  true,
			EnableMetrics:  true,
			JaegerEndpoint: "http://localhost:14268/api/traces",
			PrometheusPort: ":9090",
		},
	}

	// Try to load from config file if exists
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	if _, err := os.Stat(configPath); err == nil {
		if err := config.LoadYAML(configPath, cfg); err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Override with environment variables
	if port := os.Getenv("PORT"); port != "" {
		cfg.Server.Port = ":" + port
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.Auth.JWTSecret = jwtSecret
	}

	return cfg, nil
}

func setupEnterpriseApplication(deps map[reflect.Type]interface{}, cfg *AppConfig, logger core.Logger) error {
	vertx := deps[reflect.TypeOf((*core.GoCMD)(nil)).Elem()].(core.GoCMD)
	eventBus := vertx.EventBus()

	// 1. Setup OpenTelemetry Tracing
	if cfg.Observability.EnableTracing {
		otelConfig := otel.Config{
			ServiceName:    "fluxor-enterprise",
			ServiceVersion: "1.0.0",
			Environment:    getEnv("ENVIRONMENT", "development"),
			Exporter:       "jaeger",
			Endpoint:       cfg.Observability.JaegerEndpoint,
			SampleRate:     1.0,
		}

		if err := otel.Initialize(context.Background(), otelConfig); err != nil {
			logger.Info("Failed to initialize OpenTelemetry", "error", err)
		} else {
			logger.Info("OpenTelemetry tracing enabled", "endpoint", cfg.Observability.JaegerEndpoint)
		}
	}

	// 2. Setup Prometheus Metrics
	if cfg.Observability.EnableMetrics {
		// Prometheus metrics will be exposed via /metrics endpoint on the main server
		logger.Info("Prometheus metrics enabled")
	}

	// 3. Setup Database Connection Pooling
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Database,
	)

	dbConfig := dbruntime.PoolConfig{
		DSN:             dsn,
		DriverName:      "postgres",
		MaxOpenConns:    cfg.Database.MaxConnections,
		MaxIdleConns:    cfg.Database.MinConnections,
		ConnMaxIdleTime: time.Duration(cfg.Database.MaxIdleTime) * time.Second,
		ConnMaxLifetime: 30 * time.Minute,
	}

	dbComponent := dbruntime.NewDatabaseComponent(dbConfig)
	logger.Info("Database connection pool configured",
		"max", cfg.Database.MaxConnections,
		"min", cfg.Database.MinConnections)

	// 4. Deploy Business Verticles
	userVerticle := &UserServiceVerticle{
		eventBus: eventBus,
		logger:   logger,
		db:       dbComponent,
	}

	deploymentID, err := vertx.DeployVerticle(userVerticle)
	if err != nil {
		return fmt.Errorf("failed to deploy user verticle: %w", err)
	}
	logger.Info("User service verticle deployed", "deployment_id", deploymentID)

	// 5. Setup FastHTTP Server with CCU-based Backpressure
	maxCCU := cfg.Server.MaxCCU
	utilizationPercent := cfg.Server.UtilizationPercent
	serverConfig := web.CCUBasedConfigWithUtilization(cfg.Server.Port, maxCCU, utilizationPercent)
	normalCapacity := serverConfig.MaxQueue + serverConfig.Workers

	logger.Info("Server configured with CCU-based backpressure",
		"max_ccu", maxCCU,
		"normal_capacity", normalCapacity,
		"utilization", fmt.Sprintf("%d%%", utilizationPercent),
		"workers", serverConfig.Workers,
		"queue", serverConfig.MaxQueue)

	server := web.NewFastHTTPServer(vertx, serverConfig)
	router := server.FastRouter()

	// 6. Setup Middleware Chain (Express-like)

	// Security middleware (CORS, Security Headers, Rate Limiting)
	corsMiddleware := security.CORS(security.CORSConfig{
		AllowedOrigins: cfg.Auth.AllowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:         3600,
	})

	securityHeadersMiddleware := security.Headers(security.HeadersConfig{
		HSTS:                true,
		HSTSMaxAge:          31536000,
		HSTSIncludeSub:      true,
		CSP:                 "default-src 'self'",
		XFrameOptions:       "DENY",
		XContentTypeOptions: true,
	})

	rateLimitMiddleware := security.RateLimit(security.RateLimitConfig{
		RequestsPerMinute: 1000, // 1000 requests per minute per IP
		KeyFunc: func(ctx *web.FastRequestContext) string {
			return ctx.RequestCtx.RemoteIP().String()
		},
	})

	// Logging middleware
	loggingMiddleware := middleware.Logging(middleware.LoggingConfig{
		Logger:       logger,
		LogRequestID: true,
		SkipPaths:    []string{"/health"},
	})

	// Recovery middleware (panic recovery)
	recoveryMiddleware := middleware.Recovery(middleware.RecoveryConfig{
		Logger: logger,
	})

	// Compression middleware
	compressionMiddleware := middleware.Compression(middleware.CompressionConfig{})

	// OpenTelemetry middleware for distributed tracing
	var otelMiddleware web.FastMiddleware
	if cfg.Observability.EnableTracing && otel.IsInitialized() {
		otelMiddleware = otel.HTTPMiddleware()
	}

	// Apply middleware chain
	middlewareChain := []web.FastMiddleware{
		recoveryMiddleware,
		loggingMiddleware,
		corsMiddleware,
		securityHeadersMiddleware,
		rateLimitMiddleware,
		compressionMiddleware,
	}

	if otelMiddleware != nil {
		middlewareChain = append(middlewareChain, otelMiddleware)
	}

	// 7. Setup Routes

	// Public routes (no auth required)
	router.GETFast("/", applyMiddleware(middlewareChain, handleHome(logger)))
	router.GETFast("/health", applyMiddleware(middlewareChain, handleHealth(dbComponent)))
	router.GETFast("/ready", applyMiddleware(middlewareChain, handleReady(server, dbComponent)))

	// Prometheus metrics endpoint
	if cfg.Observability.EnableMetrics {
		prometheus.RegisterMetricsEndpoint(router, "/metrics")
	}

	// API routes with authentication
	apiMiddleware := append(middlewareChain, createAuthMiddleware(cfg.Auth.JWTSecret))

	router.GETFast("/api/users", applyMiddleware(apiMiddleware, handleGetUsers(eventBus, logger)))
	router.POSTFast("/api/users", applyMiddleware(apiMiddleware, handleCreateUser(eventBus, logger)))
	router.GETFast("/api/users/:id", applyMiddleware(apiMiddleware, handleGetUser(eventBus, logger)))

	// Admin routes with RBAC
	adminMiddleware := append(apiMiddleware, createRBACMiddleware([]string{"admin"}))
	router.GETFast("/api/admin/metrics", applyMiddleware(adminMiddleware, handleMetrics(server)))
	router.GETFast("/api/admin/stats", applyMiddleware(adminMiddleware, handleStats(dbComponent)))

	// Auth routes
	router.POSTFast("/api/auth/login", applyMiddleware(middlewareChain, handleLogin(cfg.Auth.JWTSecret, logger)))
	router.POSTFast("/api/auth/register", applyMiddleware(middlewareChain, handleRegister(eventBus, logger)))

	// 8. Setup Enhanced Health Checks
	registry := health.NewRegistry()

	// Add database health check
	registry.Register("database", health.DatabaseComponentCheck(dbComponent))

	// Add external service health check (example - optional, may fail if service doesn't exist)
	// registry.Register("external_api", health.HTTPCheck("https://api.example.com/health", 5*time.Second))

	healthAggregator := health.NewAggregator(registry)

	router.GETFast("/health/detailed", applyMiddleware(middlewareChain, healthAggregator.HandleHealth))

	// 9. Setup Server Handler
	server.SetHandler(func(ctx *fasthttp.RequestCtx) {
		reqCtx := &web.FastRequestContext{
			RequestCtx: ctx,
			GoCMD:      vertx,
			EventBus:   eventBus,
			Params:     make(map[string]string),
		}
		router.ServeFastHTTP(reqCtx)
	})

	// 10. Start Server
	go func() {
		logger.Info("Starting FastHTTP server", "port", cfg.Server.Port)
		if err := server.Start(); err != nil {
			logger.Error("FastHTTP server error", "error", err)
		}
	}()

	return nil
}

// Helper function to apply middleware chain
func applyMiddleware(middlewares []web.FastMiddleware, handler web.FastRequestHandler) web.FastRequestHandler {
	result := handler
	// Apply middleware in reverse order (last middleware wraps first)
	for i := len(middlewares) - 1; i >= 0; i-- {
		if middlewares[i] != nil {
			result = middlewares[i](result)
		}
	}
	return result
}

func createAuthMiddleware(jwtSecret string) web.FastMiddleware {
	return auth.JWT(auth.JWTConfig{
		SecretKey:   jwtSecret,
		ClaimsKey:   "user",
		TokenLookup: "header:Authorization",
		AuthScheme:  "Bearer",
	})
}

func createRBACMiddleware(roles []string) web.FastMiddleware {
	return auth.RequireAnyRole(roles...)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Route Handlers

func handleHome(logger core.Logger) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{
			"message":    "Welcome to Fluxor Enterprise API",
			"version":    "1.0.0",
			"request_id": ctx.RequestID(),
			"features": []string{
				"OpenTelemetry Distributed Tracing",
				"Prometheus Metrics",
				"JWT Authentication",
				"RBAC Authorization",
				"Database Connection Pooling",
				"CCU-based Backpressure",
				"Security Headers",
				"CORS",
				"Rate Limiting",
				"Structured Logging",
				"Enhanced Health Checks",
			},
		})
	}
}

func handleHealth(dbComponent *dbruntime.DatabaseComponent) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{
			"status":    "healthy",
			"service":   "fluxor-enterprise",
			"timestamp": time.Now().Unix(),
		})
	}
}

func handleReady(server *web.FastHTTPServer, dbComponent *dbruntime.DatabaseComponent) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		metrics := server.Metrics()

		// Check server readiness
		serverReady := metrics.QueueUtilization < 90.0 && metrics.CCUUtilization < 90.0

		// Check database readiness (example)
		dbReady := true // dbComponent.IsHealthy()

		ready := serverReady && dbReady
		statusCode := 200
		if !ready {
			statusCode = 503
		}

		return ctx.JSON(statusCode, map[string]interface{}{
			"ready": ready,
			"checks": map[string]interface{}{
				"server": map[string]interface{}{
					"ready":             serverReady,
					"queue_utilization": fmt.Sprintf("%.2f%%", metrics.QueueUtilization),
					"ccu_utilization":   fmt.Sprintf("%.2f%%", metrics.CCUUtilization),
				},
				"database": map[string]interface{}{
					"ready": dbReady,
				},
			},
		})
	}
}

func handleGetUsers(eventBus core.EventBus, logger core.Logger) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		logger.WithContext(ctx.Context()).Info("Fetching users")

		// Example: Send request to user service via event bus
		// msg, err := eventBus.Request("user.service.get", nil, 5*time.Second)

		return ctx.JSON(200, map[string]interface{}{
			"users": []map[string]interface{}{
				{"id": "1", "name": "John Doe", "email": "john@example.com"},
				{"id": "2", "name": "Jane Smith", "email": "jane@example.com"},
			},
			"request_id": ctx.RequestID(),
		})
	}
}

func handleCreateUser(eventBus core.EventBus, logger core.Logger) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		var userData map[string]interface{}
		if err := ctx.BindJSON(&userData); err != nil {
			return ctx.JSON(400, map[string]interface{}{
				"error": "invalid request body",
			})
		}

		logger.WithContext(ctx.Context()).Info("Creating user", "data", userData)

		return ctx.JSON(201, map[string]interface{}{
			"message":    "User created successfully",
			"user":       userData,
			"request_id": ctx.RequestID(),
		})
	}
}

func handleGetUser(eventBus core.EventBus, logger core.Logger) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		userID := ctx.Param("id")

		logger.WithContext(ctx.Context()).Info("Fetching user", "user_id", userID)

		return ctx.JSON(200, map[string]interface{}{
			"user": map[string]interface{}{
				"id":    userID,
				"name":  "John Doe",
				"email": "john@example.com",
			},
			"request_id": ctx.RequestID(),
		})
	}
}

func handleMetrics(server *web.FastHTTPServer) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		metrics := server.Metrics()

		return ctx.JSON(200, map[string]interface{}{
			"server": map[string]interface{}{
				"queued_requests":     metrics.QueuedRequests,
				"rejected_requests":   metrics.RejectedRequests,
				"queue_capacity":      metrics.QueueCapacity,
				"queue_utilization":   fmt.Sprintf("%.2f%%", metrics.QueueUtilization),
				"workers":             metrics.Workers,
				"normal_ccu":          metrics.NormalCCU,
				"current_ccu":         metrics.CurrentCCU,
				"ccu_utilization":     fmt.Sprintf("%.2f%%", metrics.CCUUtilization),
				"backpressure_active": metrics.CCUUtilization >= 100.0,
				"total_requests":      metrics.TotalRequests,
				"successful_requests": metrics.SuccessfulRequests,
				"error_requests":      metrics.ErrorRequests,
			},
			"request_id": ctx.RequestID(),
		})
	}
}

func handleStats(dbComponent *dbruntime.DatabaseComponent) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		stats := dbComponent.Stats()

		return ctx.JSON(200, map[string]interface{}{
			"database":   stats,
			"request_id": ctx.RequestID(),
		})
	}
}

func handleLogin(jwtSecret string, logger core.Logger) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		var credentials map[string]interface{}
		if err := ctx.BindJSON(&credentials); err != nil {
			return ctx.JSON(400, map[string]interface{}{
				"error": "invalid request body",
			})
		}

		// TODO: Validate credentials against database
		// For demo purposes, accept any credentials

		// Generate JWT token
		tokenGenerator := auth.NewJWTTokenGenerator([]byte(jwtSecret))
		token, err := tokenGenerator.Generate(map[string]interface{}{
			"user_id": "123",
			"email":   credentials["email"],
			"roles":   []string{"user"},
		}, 24*time.Hour)

		if err != nil {
			logger.Error("Failed to generate token", "error", err)
			return ctx.JSON(500, map[string]interface{}{
				"error": "failed to generate token",
			})
		}

		return ctx.JSON(200, map[string]interface{}{
			"token":      token,
			"expires_in": 86400,
			"request_id": ctx.RequestID(),
		})
	}
}

func handleRegister(eventBus core.EventBus, logger core.Logger) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		var userData map[string]interface{}
		if err := ctx.BindJSON(&userData); err != nil {
			return ctx.JSON(400, map[string]interface{}{
				"error": "invalid request body",
			})
		}

		logger.WithContext(ctx.Context()).Info("Registering new user", "email", userData["email"])

		// TODO: Create user in database
		// TODO: Send welcome email via event bus

		return ctx.JSON(201, map[string]interface{}{
			"message": "User registered successfully",
			"user": map[string]interface{}{
				"id":    "new-user-id",
				"email": userData["email"],
			},
			"request_id": ctx.RequestID(),
		})
	}
}

// UserServiceVerticle - Business logic verticle
type UserServiceVerticle struct {
	eventBus core.EventBus
	logger   core.Logger
		db       *dbruntime.DatabaseComponent
}

func (v *UserServiceVerticle) Start(ctx core.FluxorContext) error {
	v.logger.Info("UserServiceVerticle started")

	// Register event bus consumers for user service
	consumer := ctx.EventBus().Consumer("user.service.get")
	consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
		v.logger.Info("Handling user.service.get request")

		// TODO: Fetch from database
		users := []map[string]interface{}{
			{"id": "1", "name": "John Doe"},
			{"id": "2", "name": "Jane Smith"},
		}

		return msg.Reply(users)
	})

	return nil
}

func (v *UserServiceVerticle) Stop(ctx core.FluxorContext) error {
	v.logger.Info("UserServiceVerticle stopped")
	return nil
}
