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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	StatusOnline  = "online"
	StatusStopped = "stopped"
	StatusError   = "error"
)

type ProcessInfo struct {
	Name        string    `json:"name"`
	PID         int       `json:"pid"`
	Status      string    `json:"status"`
	WorkingDir  string    `json:"working_dir"`
	Command     string    `json:"command"`
	Args        []string  `json:"args"`
	LogFile     string    `json:"log_file"`
	ErrorFile   string    `json:"error_file"`
	Restarts    int       `json:"restarts"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	AutoRestart bool      `json:"auto_restart"`
}

type ProcessManager struct {
	mu        sync.RWMutex
	processes map[string]*ProcessInfo
	dataDir   string
	dataFile  string
}

func NewProcessManager() (*ProcessManager, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	dataDir := filepath.Join(usr.HomeDir, ".fluxor-cli")
	dataFile := filepath.Join(dataDir, "processes.json")

	pm := &ProcessManager{
		processes: make(map[string]*ProcessInfo),
		dataDir:   dataDir,
		dataFile:  dataFile,
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Load existing processes
	if err := pm.load(); err != nil {
		// If file doesn't exist, that's okay
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load processes: %w", err)
		}
	}

	// Clean up stale processes
	pm.cleanupStaleProcesses()

	return pm, nil
}

func (pm *ProcessManager) load() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	data, err := os.ReadFile(pm.dataFile)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, &pm.processes)
}

// saveLocked writes processes to disk. Caller must hold pm.mu Lock (or RLock not used for write).
func (pm *ProcessManager) saveLocked() error {
	data, err := json.MarshalIndent(pm.processes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal processes: %w", err)
	}
	return os.WriteFile(pm.dataFile, data, 0644)
}

func (pm *ProcessManager) save() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.saveLocked()
}

func (pm *ProcessManager) cleanupStaleProcesses() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, proc := range pm.processes {
		if proc.Status == StatusOnline {
			// Check if process is still running
			if !isProcessRunning(proc.PID) {
				proc.Status = StatusStopped
				proc.UpdatedAt = time.Now()
			}
		}
	}

	_ = pm.saveLocked()
}

func isProcessRunning(pid int) bool {
	// On Unix systems, sending signal 0 checks if process exists
	err := syscall.Kill(pid, 0)
	return err == nil
}

func (pm *ProcessManager) Start(name, workingDir, command string, args []string, autoRestart bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if process already exists
	if existing, ok := pm.processes[name]; ok {
		if existing.Status == StatusOnline && isProcessRunning(existing.PID) {
			return fmt.Errorf("process '%s' is already running (PID: %d)", name, existing.PID)
		}
	}

	// Create log files
	logFile := filepath.Join(pm.dataDir, fmt.Sprintf("%s.log", name))
	errorFile := filepath.Join(pm.dataDir, fmt.Sprintf("%s.error.log", name))

	// Build command
	cmd := exec.Command(command, args...)
	cmd.Dir = workingDir
	cmd.Env = os.Environ()

	// Redirect output
	logFileHandle, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFileHandle.Close()

	errorFileHandle, err := os.OpenFile(errorFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open error file: %w", err)
	}
	defer errorFileHandle.Close()

	cmd.Stdout = logFileHandle
	cmd.Stderr = errorFileHandle

	// Start process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	pid := cmd.Process.Pid

	// Note: File handles are closed by defer after function returns.
	// This is safe because on Unix systems, file descriptors are reference-counted.
	// When cmd.Start() is called, the process gets its own file descriptors,
	// so closing our handles doesn't affect the process's file descriptors.

	// Create process info
	proc := &ProcessInfo{
		Name:        name,
		PID:         pid,
		Status:      StatusOnline,
		WorkingDir:  workingDir,
		Command:     command,
		Args:        args,
		LogFile:     logFile,
		ErrorFile:   errorFile,
		Restarts:    0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AutoRestart: autoRestart,
	}

	pm.processes[name] = proc

	// Save to disk (we hold Lock)
	if err := pm.saveLocked(); err != nil {
		// Try to kill the process if save failed
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to save process info: %w", err)
	}

	// Monitor process in background
	go pm.monitorProcess(name, pid)

	return nil
}

func (pm *ProcessManager) monitorProcess(name string, pid int) {
	// Wait a bit for process to start
	time.Sleep(500 * time.Millisecond)

	for {
		if !isProcessRunning(pid) {
			pm.mu.Lock()
			proc, exists := pm.processes[name]
			if !exists {
				pm.mu.Unlock()
				return
			}

			if proc.Status == StatusOnline {
				proc.Status = StatusStopped
				proc.UpdatedAt = time.Now()
				
				// Save process info before unlocking
				autoRestart := proc.AutoRestart
				workingDir := proc.WorkingDir
				command := proc.Command
				args := proc.Args
				_ = pm.saveLocked()
				pm.mu.Unlock()

				// Auto-restart if enabled
				if autoRestart {
					time.Sleep(1 * time.Second)
					// Increment restart count
					pm.mu.Lock()
					if proc, exists := pm.processes[name]; exists {
						proc.Restarts++
						_ = pm.saveLocked()
					}
					pm.mu.Unlock()
					
					// Try to restart
					if err := pm.Start(name, workingDir, command, args, autoRestart); err != nil {
						pm.mu.Lock()
						if proc, exists := pm.processes[name]; exists {
							proc.Status = StatusError
							_ = pm.saveLocked()
						}
						pm.mu.Unlock()
					}
					return
				}
			} else {
				pm.mu.Unlock()
			}
			return
		}

		time.Sleep(2 * time.Second)
	}
}

func (pm *ProcessManager) Stop(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proc, ok := pm.processes[name]
	if !ok {
		return fmt.Errorf("process '%s' not found", name)
	}

	if proc.Status != StatusOnline {
		return fmt.Errorf("process '%s' is not running (status: %s)", name, proc.Status)
	}

	process, err := os.FindProcess(proc.PID)
	if err != nil {
		proc.Status = StatusStopped
		proc.UpdatedAt = time.Now()
		_ = pm.saveLocked()
		return fmt.Errorf("process not found: %w", err)
	}

	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	proc.Status = StatusStopped
	proc.UpdatedAt = time.Now()
	_ = pm.saveLocked()

	return nil
}

func (pm *ProcessManager) Restart(name string) error {
	pm.mu.Lock()
	proc, ok := pm.processes[name]
	if !ok {
		pm.mu.Unlock()
		return fmt.Errorf("process '%s' not found", name)
	}

	workingDir := proc.WorkingDir
	command := proc.Command
	args := proc.Args
	autoRestart := proc.AutoRestart
	pm.mu.Unlock()

	// Stop if running
	if proc.Status == StatusOnline {
		if err := pm.Stop(name); err != nil {
			return fmt.Errorf("failed to stop process: %w", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Start again
	proc.Restarts++
	return pm.Start(name, workingDir, command, args, autoRestart)
}

func (pm *ProcessManager) Delete(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proc, ok := pm.processes[name]
	if !ok {
		return fmt.Errorf("process '%s' not found", name)
	}

	// Stop if running
	if proc.Status == StatusOnline {
		if process, err := os.FindProcess(proc.PID); err == nil {
			_ = process.Kill()
		}
	}

	delete(pm.processes, name)
	return pm.saveLocked()
}

func (pm *ProcessManager) List() []*ProcessInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Update status of online processes
	for _, proc := range pm.processes {
		if proc.Status == StatusOnline {
			if !isProcessRunning(proc.PID) {
				proc.Status = StatusStopped
				proc.UpdatedAt = time.Now()
			}
		}
	}

	// Return all processes
	processes := make([]*ProcessInfo, 0, len(pm.processes))
	for _, proc := range pm.processes {
		processes = append(processes, proc)
	}

	return processes
}

func (pm *ProcessManager) Get(name string) (*ProcessInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	proc, ok := pm.processes[name]
	if !ok {
		return nil, fmt.Errorf("process '%s' not found", name)
	}

	// Update status if online
	if proc.Status == StatusOnline && !isProcessRunning(proc.PID) {
		proc.Status = StatusStopped
		proc.UpdatedAt = time.Now()
	}

	return proc, nil
}

func (pm *ProcessManager) GetLogs(name string, lines int) (string, error) {
	proc, err := pm.Get(name)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(proc.LogFile)
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	// Simple line limiting (for production, use tail-like functionality)
	logContent := string(data)
	if lines > 0 {
		// Count lines and return last N lines
		allLines := []rune(logContent)
		lineCount := 0
		for i := len(allLines) - 1; i >= 0; i-- {
			if allLines[i] == '\n' {
				lineCount++
				if lineCount >= lines {
					logContent = string(allLines[i+1:])
					break
				}
			}
		}
	}

	return logContent, nil
}

func (pm *ProcessManager) GetErrorLogs(name string, lines int) (string, error) {
	proc, err := pm.Get(name)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(proc.ErrorFile)
	if err != nil {
		return "", fmt.Errorf("failed to read error log file: %w", err)
	}

	logContent := string(data)
	if lines > 0 {
		allLines := []rune(logContent)
		lineCount := 0
		for i := len(allLines) - 1; i >= 0; i-- {
			if allLines[i] == '\n' {
				lineCount++
				if lineCount >= lines {
					logContent = string(allLines[i+1:])
					break
				}
			}
		}
	}

	return logContent, nil
}

// Helper function to find Go binary
func findGoBinary() (string, error) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("go binary not found in PATH: %w", err)
	}
	return goPath, nil
}

// Helper function to check if a directory contains a Go application
func isGoApp(dir string) bool {
	mainGo := filepath.Join(dir, "main.go")
	_, err := os.Stat(mainGo)
	return err == nil
}
