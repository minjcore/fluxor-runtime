package main

import (
	"container/list"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DiskCache manages disk cache with LRU eviction
type DiskCache struct {
	root         string
	maxSize      int64
	currentSize  int64
	fileList     *list.List
	fileMap      map[string]*list.Element
	mu           sync.RWMutex
	lastCleanup  time.Time
	cleanupEvery time.Duration
}

// CacheEntry represents a cached file
type CacheEntry struct {
	Path      string
	Size      int64
	LastUsed  time.Time
	CreatedAt time.Time
}

// NewDiskCache creates a new disk cache manager
func NewDiskCache(root string, maxSize int64) *DiskCache {
	dc := &DiskCache{
		root:         root,
		maxSize:      maxSize,
		fileList:     list.New(),
		fileMap:      make(map[string]*list.Element),
		lastCleanup:  time.Now(),
		cleanupEvery: 10 * time.Minute,
	}

	// Initialize by scanning existing cache
	dc.scanCache()

	// Start background cleanup routine
	go dc.backgroundCleanup()

	return dc
}

// scanCache scans the cache directory and builds the index
func (dc *DiskCache) scanCache() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	filepath.Walk(dc.root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		entry := &CacheEntry{
			Path:      path,
			Size:      info.Size(),
			LastUsed:  info.ModTime(),
			CreatedAt: info.ModTime(),
		}

		dc.currentSize += info.Size()
		element := dc.fileList.PushBack(entry)
		dc.fileMap[path] = element

		return nil
	})
}

// Add adds a file to the cache index
func (dc *DiskCache) Add(path string, size int64) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// If file already exists, update it
	if elem, exists := dc.fileMap[path]; exists {
		entry := elem.Value.(*CacheEntry)
		dc.currentSize -= entry.Size
		dc.currentSize += size
		entry.Size = size
		entry.LastUsed = time.Now()
		dc.fileList.MoveToBack(elem)
		return
	}

	// Add new entry
	entry := &CacheEntry{
		Path:      path,
		Size:      size,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}

	dc.currentSize += size
	element := dc.fileList.PushBack(entry)
	dc.fileMap[path] = element

	// Evict if necessary
	dc.evictIfNeeded()
}

// Touch updates the last used time for a file
func (dc *DiskCache) Touch(path string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if elem, exists := dc.fileMap[path]; exists {
		entry := elem.Value.(*CacheEntry)
		entry.LastUsed = time.Now()
		dc.fileList.MoveToBack(elem)
	}
}

// evictIfNeeded removes old files if cache size exceeds limit
func (dc *DiskCache) evictIfNeeded() {
	for dc.currentSize > dc.maxSize && dc.fileList.Len() > 0 {
		// Remove oldest (front of list)
		elem := dc.fileList.Front()
		if elem == nil {
			break
		}

		entry := elem.Value.(*CacheEntry)

		// Remove file from disk
		if err := os.Remove(entry.Path); err == nil {
			dc.currentSize -= entry.Size
		}

		// Remove from index
		dc.fileList.Remove(elem)
		delete(dc.fileMap, entry.Path)
	}
}

// backgroundCleanup periodically cleans up the cache
func (dc *DiskCache) backgroundCleanup() {
	ticker := time.NewTicker(dc.cleanupEvery)
	defer ticker.Stop()

	for range ticker.C {
		dc.cleanup()
	}
}

// cleanup removes stale entries and orphaned files
func (dc *DiskCache) cleanup() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	now := time.Now()

	// Remove files older than 7 days
	maxAge := 7 * 24 * time.Hour
	toRemove := []*list.Element{}

	for elem := dc.fileList.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*CacheEntry)
		if now.Sub(entry.LastUsed) > maxAge {
			toRemove = append(toRemove, elem)
		}
	}

	for _, elem := range toRemove {
		entry := elem.Value.(*CacheEntry)
		os.Remove(entry.Path)
		dc.currentSize -= entry.Size
		dc.fileList.Remove(elem)
		delete(dc.fileMap, entry.Path)
	}

	// Clean up empty directories
	dc.cleanupEmptyDirs()

	dc.lastCleanup = now
}

// cleanupEmptyDirs removes empty directories in the cache
func (dc *DiskCache) cleanupEmptyDirs() {
	filepath.Walk(dc.root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == dc.root {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err == nil && len(entries) == 0 {
			os.Remove(path)
		}

		return nil
	})
}

// Size returns the current cache size in bytes
func (dc *DiskCache) Size() int64 {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.currentSize
}

// FileCount returns the number of cached files
func (dc *DiskCache) FileCount() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.fileList.Len()
}

// Clear removes all cached files
func (dc *DiskCache) Clear() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	err := os.RemoveAll(dc.root)
	if err != nil {
		return err
	}

	// Use restrictive permissions (0750) to prevent unauthorized access
	err = os.MkdirAll(dc.root, 0750)
	if err != nil {
		return err
	}

	dc.currentSize = 0
	dc.fileList = list.New()
	dc.fileMap = make(map[string]*list.Element)

	return nil
}

// Stats returns cache statistics
func (dc *DiskCache) Stats() map[string]interface{} {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return map[string]interface{}{
		"total_size":    dc.currentSize,
		"max_size":      dc.maxSize,
		"file_count":    dc.fileList.Len(),
		"usage_percent": float64(dc.currentSize) / float64(dc.maxSize) * 100,
		"last_cleanup":  dc.lastCleanup,
		"cleanup_every": dc.cleanupEvery,
	}
}
