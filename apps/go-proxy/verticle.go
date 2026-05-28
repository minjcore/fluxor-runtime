// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/goproxy/handlers"
	"github.com/fluxorio/goproxy/storage"
)

//go:embed web/*
var webFS embed.FS

// GoProxyVerticle is the main verticle for the Go module proxy
type GoProxyVerticle struct {
	*core.BaseVerticle

	httpServer *web.FastHTTPServer
	storage    storage.Storage
	config     *Config
}

// NewGoProxyVerticle creates a new GoProxy verticle
func NewGoProxyVerticle() *GoProxyVerticle {
	return &GoProxyVerticle{
		BaseVerticle: core.NewBaseVerticle("goproxy"),
	}
}

// Start initializes and starts the GoProxy server
func (v *GoProxyVerticle) Start(ctx core.FluxorContext) error {
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	// Load configuration
	v.config = DefaultConfig()
	configPath := os.Getenv("GOPROXY_CONFIG")
	if configPath == "" {
		configPath = "config.json"
	}
	if err := config.LoadJSON(configPath, v.config); err != nil {
		log.Printf("Warning: failed to load config from %s: %v, using defaults", configPath, err)
	}

	log.Printf("Configuration loaded:")
	log.Printf("  Server address: %s", v.config.Server.Address)
	log.Printf("  Storage type: %s", v.config.Storage.Type)
	log.Printf("  Auth enabled: %v", v.config.Auth.Enabled)

	// Initialize storage backend
	storageConfig := storage.StorageConfig{
		Type:             v.config.Storage.Type,
		BasePath:         v.config.Storage.BasePath,
		S3Endpoint:       v.config.Storage.S3.Endpoint,
		S3Bucket:         v.config.Storage.S3.Bucket,
		S3Region:         v.config.Storage.S3.Region,
		S3AccessKey:      v.config.Storage.S3.AccessKey,
		S3SecretKey:      v.config.Storage.S3.SecretKey,
		S3ForcePathStyle: v.config.Storage.S3.ForcePathStyle,
		S3DisableSSL:     v.config.Storage.S3.DisableSSL,
	}
	if m := v.config.Storage.Mirror; m != nil {
		storageConfig.MirrorEnabled        = true
		storageConfig.MirrorS3Endpoint     = m.S3.Endpoint
		storageConfig.MirrorS3Bucket       = m.S3.Bucket
		storageConfig.MirrorS3Region       = m.S3.Region
		storageConfig.MirrorS3AccessKey    = m.S3.AccessKey
		storageConfig.MirrorS3SecretKey    = m.S3.SecretKey
		storageConfig.MirrorS3ForcePathStyle = m.S3.ForcePathStyle
		storageConfig.MirrorS3DisableSSL   = m.S3.DisableSSL
		log.Printf("  Storage mirror: OSS bucket %s", m.S3.Bucket)
	}

	var err error
	v.storage, err = storage.NewStorage(storageConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	log.Printf("Storage backend initialized: %s", v.config.Storage.Type)

	// Parse timeouts
	readTimeout, _ := time.ParseDuration(v.config.Server.ReadTimeout)
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	writeTimeout, _ := time.ParseDuration(v.config.Server.WriteTimeout)
	if writeTimeout <= 0 {
		writeTimeout = 30 * time.Second
	}

	// Create HTTP server
	serverConfig := &web.FastHTTPServerConfig{
		Addr:         v.config.Server.Address,
		MaxQueue:     v.config.Server.MaxQueue,
		Workers:      v.config.Server.Workers,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	v.httpServer = web.NewFastHTTPServer(ctx.GoCMD(), serverConfig)
	router := v.httpServer.FastRouter()

	// Setup authentication middleware
	authConfig := handlers.AuthConfig{
		Enabled: v.config.Auth.Enabled,
		Users:   make(map[string]string),
	}
	for _, user := range v.config.Auth.Users {
		authConfig.Users[user.Username] = user.Password
	}
	authMiddleware := handlers.NewAuthMiddleware(authConfig)

	// Create handlers
	proxyHandler := handlers.NewProxyHandler(v.storage)
	apiHandler := handlers.NewAPIHandler(v.storage)

	// Register static file handler for web UI
	webContent, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("failed to get web content: %w", err)
	}
	
	// Serve static files
	router.GETFast("/", v.serveIndex)
	router.GETFast("/static/*filepath", v.serveStaticFiles(webContent))

	// Register API routes (with read-only auth for GET, full auth for writes)
	router.UseFast(authMiddleware.ReadOnlyMiddleware())
	apiHandler.RegisterRoutes(router)

	// Register GOPROXY protocol routes (needs to be last due to wildcard)
	// The proxy handler will catch all module requests
	proxyHandler.RegisterRoutes(router)

	// Start HTTP server in a goroutine
	v.ExecuteOn(func() {
		log.Printf("Starting GoProxy HTTP server on %s...", v.config.Server.Address)
		if err := v.httpServer.Start(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	})

	log.Println("GoProxy started successfully!")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("  Web UI:  http://localhost%s", v.config.Server.Address)
	log.Printf("  API:     http://localhost%s/api/v1", v.config.Server.Address)
	log.Printf("  GOPROXY: http://localhost%s", v.config.Server.Address)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println()
	log.Println("To use this proxy, set:")
	log.Printf("  export GOPROXY=http://localhost%s,direct", v.config.Server.Address)
	log.Println()

	return nil
}

// Stop stops the GoProxy server
func (v *GoProxyVerticle) Stop(ctx core.FluxorContext) error {
	log.Println("Stopping GoProxy...")

	if v.httpServer != nil {
		if err := v.httpServer.Stop(); err != nil {
			log.Printf("Error stopping HTTP server: %v", err)
		} else {
			log.Println("HTTP server stopped")
		}
	}

	if v.storage != nil {
		if err := v.storage.Close(); err != nil {
			log.Printf("Error closing storage: %v", err)
		} else {
			log.Println("Storage closed")
		}
	}

	return v.BaseVerticle.Stop(ctx)
}

// serveIndex serves the main index.html page
func (v *GoProxyVerticle) serveIndex(ctx *web.FastRequestContext) error {
	content, err := webFS.ReadFile("web/index.html")
	if err != nil {
		return ctx.Text(500, "Internal Server Error")
	}

	ctx.RequestCtx.Response.Header.Set("Content-Type", "text/html; charset=utf-8")
	ctx.RequestCtx.Response.SetBody(content)
	return nil
}

// serveStaticFiles returns a handler that serves static files
func (v *GoProxyVerticle) serveStaticFiles(fsys fs.FS) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		filepath := ctx.Param("filepath")
		if filepath == "" {
			return ctx.Text(404, "Not Found")
		}

		// Set content type based on extension
		contentType := "application/octet-stream"
		if len(filepath) > 3 {
			switch filepath[len(filepath)-3:] {
			case ".js":
				contentType = "application/javascript"
			case "css":
				contentType = "text/css"
			case "tml":
				contentType = "text/html"
			}
		}
		if len(filepath) > 4 && filepath[len(filepath)-4:] == ".css" {
			contentType = "text/css"
		}

		// Read file from embedded FS
		content, err := fs.ReadFile(fsys, filepath[1:]) // Remove leading slash
		if err != nil {
			return ctx.Text(404, "Not Found")
		}

		ctx.RequestCtx.Response.Header.Set("Content-Type", contentType)
		ctx.RequestCtx.Response.Header.Set("Cache-Control", "public, max-age=86400")
		ctx.RequestCtx.Response.SetBody(content)
		return nil
	}
}
