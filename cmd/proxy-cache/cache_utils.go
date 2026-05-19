// Package main provides cache utility functions for export and purging
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// ExportCacheMetadata exports cache metadata to a file or stdout
func ExportCacheMetadata(cacheDir string, outputFile string) error {
	dc := NewDiskCache(cacheDir, 1024*1024*1024*10)
	stats := dc.Stats()

	output := fmt.Sprintf(`Cache Metadata
==============
Directory:      %s
Total Size:     %d bytes (%.2f GB)
File Count:     %d
Usage:          %.2f%%
Last Cleanup:   %s
Cleanup Interval: %v

`, cacheDir, stats["total_size"], float64(stats["total_size"].(int64))/1024/1024/1024,
		stats["file_count"], stats["usage_percent"], stats["last_cleanup"], stats["cleanup_every"])

	if outputFile == "" {
		fmt.Print(output)
		return nil
	}

	// Use restrictive permissions (0600) to prevent unauthorized access
	return os.WriteFile(outputFile, []byte(output), 0600)
}

// PurgeCacheByAge removes files older than specified age
func PurgeCacheByAge(cacheDir string, maxAge time.Duration) error {
	log.Printf("Purging cache files older than %v...", maxAge)

	removed := 0
	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || path == cacheDir {
			return nil
		}

		age := time.Since(info.ModTime())
		if age > maxAge {
			if err := os.Remove(path); err != nil {
				log.Printf("Failed to remove %s: %v", path, err)
				return nil
			}
			removed++
		}

		return nil
	})

	log.Printf("Purged %d files", removed)
	return err
}

// PurgeCacheBySize removes oldest files until size threshold is met
func PurgeCacheBySize(cacheDir string, targetSize int64) error {
	log.Printf("Purging cache to target size: %.2f GB...", float64(targetSize)/1024/1024/1024)

	dc := NewDiskCache(cacheDir, targetSize)

	currentSize := dc.Size()
	if currentSize <= targetSize {
		log.Printf("Cache size already under target: %.2f GB", float64(currentSize)/1024/1024/1024)
		return nil
	}

	toRemove := currentSize - targetSize
	removed := 0

	filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || path == cacheDir {
			return nil
		}

		if toRemove <= 0 {
			return filepath.SkipDir
		}

		if err := os.Remove(path); err != nil {
			return nil
		}

		toRemove -= info.Size()
		removed++
		return nil
	})

	log.Printf("Purged %d files, freed %.2f GB", removed, float64(currentSize-dc.Size())/1024/1024/1024)
	return nil
}
