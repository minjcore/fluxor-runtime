package config

import (
	"context"
	"os"
	"sync"
	"time"
)

// ConfigWatcher watches configuration files for changes and triggers reload callbacks
type ConfigWatcher struct {
	filePath  string
	interval  time.Duration
	callbacks []func() error
	mu        sync.RWMutex
	lastMod   time.Time
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// WatchConfig creates a new config watcher
// interval: how often to check for changes (e.g., 2 * time.Second)
// callbacks: functions to call when config changes
func WatchConfig(filePath string, interval time.Duration, callbacks ...func() error) (*ConfigWatcher, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Get initial file modification time
	var lastMod time.Time
	if info, err := os.Stat(filePath); err == nil {
		lastMod = info.ModTime()
	}

	watcher := &ConfigWatcher{
		filePath:  filePath,
		interval:  interval,
		callbacks: callbacks,
		ctx:       ctx,
		cancel:    cancel,
		lastMod:   lastMod,
	}

	// Start watching
	watcher.wg.Add(1)
	go watcher.watch()

	return watcher, nil
}

// watch periodically checks for file changes
func (cw *ConfigWatcher) watch() {
	defer cw.wg.Done()

	ticker := time.NewTicker(cw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-cw.ctx.Done():
			return
		case <-ticker.C:
			cw.checkForChanges()
		}
	}
}

// checkForChanges checks if the config file has changed
func (cw *ConfigWatcher) checkForChanges() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	// Get file modification time
	info, err := os.Stat(cw.filePath)
	if err != nil {
		// File doesn't exist or can't be accessed - skip this check
		return
	}

	modTime := info.ModTime()

	// Check if file has changed
	if modTime.After(cw.lastMod) {
		cw.lastMod = modTime

		// Call all registered callbacks
		for _, callback := range cw.callbacks {
			if err := callback(); err != nil {
				// Log error but continue with other callbacks
				// In production, you might want to use a logger here
				_ = err
			}
		}
	}
}

// Stop stops watching for changes
func (cw *ConfigWatcher) Stop() {
	cw.cancel()
	cw.wg.Wait()
}

// AddCallback adds a callback to be called when config changes
func (cw *ConfigWatcher) AddCallback(callback func() error) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.callbacks = append(cw.callbacks, callback)
}
