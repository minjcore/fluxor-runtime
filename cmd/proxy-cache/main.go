// Package main provides a caching proxy server for package managers (Go, npm, pip, etc.)
// Built on fluxor-cache architecture for high-performance package caching
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
)

var (
	cacheDir  = flag.String("cache", "./proxycache", "Cache directory for packages")
	port      = flag.Int("port", 8080, "Port to listen on")
	host      = flag.String("host", "0.0.0.0", "Host to listen on")
	upstream  = flag.String("upstream", "https://proxy.golang.org", "Upstream proxy URL")
	cacheType = flag.String("cache-type", "memory", "Cache type: memory, redis, or disk")
	redisAddr = flag.String("redis", "localhost:6379", "Redis address (if using redis cache)")
	verbose   = flag.Bool("v", false, "Enable verbose logging")
	cacheTTL  = flag.Duration("ttl", 24*time.Hour, "Cache TTL duration")
	maxSize   = flag.Int64("max-size", 10*1024*1024*1024, "Maximum cache size in bytes (default 10GB)")
	rateLimit = flag.Int("rate-limit", 10000, "Rate limit in requests per second")
	configFile = flag.String("config", "", "Path to JSON config file")
)

// initCache initializes the cache backend based on configuration
func initCache(cfg *Config) (cache.Cache, error) {
	switch cfg.CacheType {
	case "memory":
		return cache.NewMemoryCache(), nil
	case "redis":
		// Note: Redis cache requires a RedisClient implementation
		// For now, fall back to memory cache with a warning
		log.Printf("WARNING: Redis cache requires implementing RedisClient interface")
		log.Printf("Falling back to memory cache. See pkg/cache/redis.go for implementation details")
		return cache.NewMemoryCache(), nil
	case "disk":
		return cache.NewMemoryCache(), nil // Still use memory for metadata
	default:
		return nil, fmt.Errorf("invalid cache type: %s (must be memory, redis, or disk)", cfg.CacheType)
	}
}

// loadConfig loads configuration from flags or config file
func loadConfig() (*Config, error) {
	flag.Parse()

	// Load from config file if provided
	if *configFile != "" {
		cfg, err := LoadConfig(*configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		// Validate config
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}
		return cfg, nil
	}

	// Load from flags
	cfg := &Config{
		Port:            *port,
		Host:            *host,
		Verbose:         *verbose,
		CacheDir:        *cacheDir,
		CacheType:       *cacheType,
		CacheTTL:        *cacheTTL,
		MaxCacheSize:    *maxSize,
		RedisAddr:       *redisAddr,
		Upstream:        *upstream,
		UpstreamTimeout: 30 * time.Second,
		RateLimit:       *rateLimit,
		CleanupInterval: 10 * time.Minute,
		MaxFileAge:      7 * 24 * time.Hour,
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// setupServer creates and configures the HTTP server
func setupServer(cfg *Config, proxyCache *ProxyCache) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyCache.handleRequest)
	mux.HandleFunc("/_stats", proxyCache.statsHandler)
	mux.HandleFunc("/_health", proxyCache.healthHandler)

	// Add middleware
	limiter := NewRateLimiter(cfg.RateLimit)
	logger := &RequestLogger{}

	handler := CORSMiddleware(cfg.AllowedOrigins)(
		RateLimitMiddleware(limiter)(
			LoggingMiddleware(logger, cfg.Verbose)(mux),
		),
	)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// setupGracefulShutdown handles graceful server shutdown
func setupGracefulShutdown(server *http.Server) {
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("\nShutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()
}

func main() {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize cache backend
	c, err := initCache(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}

	// Create proxy cache service
	proxyCache, err := NewProxyCache(cfg, c)
	if err != nil {
		log.Fatalf("Failed to create proxy cache: %v", err)
	}

	// Setup HTTP server
	server := setupServer(cfg, proxyCache)

	// Setup graceful shutdown
	setupGracefulShutdown(server)

	// Log startup information
	log.Printf("🚀 ProxyCache starting on %s:%d", cfg.Host, cfg.Port)
	log.Printf("   Cache directory: %s", cfg.CacheDir)
	log.Printf("   Upstream: %s", cfg.Upstream)
	log.Printf("   Cache type: %s", cfg.CacheType)
	log.Printf("   Max cache size: %d bytes", cfg.MaxCacheSize)
	log.Printf("   Cache TTL: %v", cfg.CacheTTL)
	log.Printf("   Rate limit: %d req/s", cfg.RateLimit)
	log.Println("   Endpoints:")
	log.Println("     - / (proxy)")
	log.Println("     - /_stats (statistics)")
	log.Println("     - /_health (health check)")

	// Start server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
