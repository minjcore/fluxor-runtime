// Package main provides cache warming utilities
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// CacheWarmer preloads packages into the cache
type CacheWarmer struct {
	proxyURL string
	packages []string
	parallel int
	verbose  bool
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(proxyURL string, parallel int, verbose bool) *CacheWarmer {
	return &CacheWarmer{
		proxyURL: proxyURL,
		packages: []string{},
		parallel: parallel,
		verbose:  verbose,
	}
}

// AddPackage adds a package to warm
func (cw *CacheWarmer) AddPackage(pkg string) {
	cw.packages = append(cw.packages, pkg)
}

// AddPackagesFromFile reads packages from a file
func (cw *CacheWarmer) AddPackagesFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			cw.AddPackage(line)
		}
	}

	return nil
}

// Warm preloads all packages into the cache
func (cw *CacheWarmer) Warm(ctx context.Context) error {
	if len(cw.packages) == 0 {
		return fmt.Errorf("no packages to warm")
	}

	log.Printf("Warming cache with %d packages (parallel=%d)...", len(cw.packages), cw.parallel)

	// Create worker pool
	pkgChan := make(chan string, len(cw.packages))
	errChan := make(chan error, len(cw.packages))

	// Start workers
	for i := 0; i < cw.parallel; i++ {
		go cw.worker(ctx, pkgChan, errChan)
	}

	// Send packages
	go func() {
		for _, pkg := range cw.packages {
			select {
			case pkgChan <- pkg:
			case <-ctx.Done():
				return
			}
		}
		close(pkgChan)
	}()

	// Wait for completion
	var errs []error
	completed := 0
	for completed < len(cw.packages) {
		select {
		case err := <-errChan:
			if err != nil {
				errs = append(errs, err)
			}
			completed++
			if cw.verbose {
				log.Printf("Progress: %d/%d", completed, len(cw.packages))
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if len(errs) > 0 {
		log.Printf("Warnings: %d errors occurred during warming", len(errs))
		for _, err := range errs {
			log.Printf("  - %v", err)
		}
	}

	log.Printf("Cache warming complete!")
	return nil
}

// worker processes packages
func (cw *CacheWarmer) worker(ctx context.Context, pkgChan chan string, errChan chan error) {
	client := &http.Client{Timeout: 30 * time.Second}

	for pkg := range pkgChan {
		select {
		case <-ctx.Done():
			return
		default:
		}

		url := cw.proxyURL + pkg
		if cw.verbose {
			log.Printf("Fetching: %s", pkg)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			errChan <- fmt.Errorf("failed to create request for %s: %w", pkg, err)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch %s: %w", pkg, err)
			continue
		}

		// Discard body
		io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errChan <- fmt.Errorf("failed to fetch %s: status %d", pkg, resp.StatusCode)
			continue
		}

		errChan <- nil
	}
}
