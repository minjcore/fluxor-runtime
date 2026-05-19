// Package main provides HTTP handlers for the proxy cache
package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// handleRequest is the main HTTP handler for proxy requests
func (p *ProxyCache) handleRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	// Decode module path (Go module proxy format)
	modulePath, err := url.PathUnescape(path)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Try cache first
	if p.ServeFromCache(w, r, modulePath) {
		return
	}

	// Cache miss - fetch from upstream
	p.FetchAndCache(w, r, modulePath)
}

// statsHandler provides cache statistics
func (p *ProxyCache) statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	stats := p.GetStats()
	
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "failed to encode stats", http.StatusInternalServerError)
		return
	}
}

// healthHandler provides health check endpoint
func (p *ProxyCache) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	health := map[string]interface{}{
		"status":      "healthy",
		"cache_root":  p.cacheRoot,
		"upstream":    p.upstreamURL,
	}
	
	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, "failed to encode health", http.StatusInternalServerError)
		return
	}
}
