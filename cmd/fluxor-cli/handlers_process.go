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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// handleStart handles the "start" command for local process management
func handleStart() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: app name is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli start <name> [directory]\n")
		fmt.Fprintf(os.Stderr, "  If directory is not specified, current directory is used\n\n")
		os.Exit(1)
	}

	name := os.Args[2]
	workingDir := "."
	if len(os.Args) >= 4 {
		workingDir = os.Args[3]
	}

	// Resolve absolute path
	absDir, err := filepath.Abs(workingDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to resolve directory: %v\n", err)
		os.Exit(1)
	}

	// Check if it's a Go app
	if !isGoApp(absDir) {
		fmt.Fprintf(os.Stderr, "Error: directory '%s' does not contain a Go application (main.go not found)\n", absDir)
		os.Exit(1)
	}

	// Find go binary
	goPath, err := findGoBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create process manager
	pm, err := NewProcessManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create process manager: %v\n", err)
		os.Exit(1)
	}

	// Start process
	autoRestart := false // Can be made configurable later
	args := []string{"run", "."}
	if err := pm.Start(name, absDir, goPath, args, autoRestart); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to start process: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Started application '%s' (PID: %d)\n", name, pm.processes[name].PID)
	fmt.Printf("   Working directory: %s\n", absDir)
	fmt.Printf("   Logs: %s\n", pm.processes[name].LogFile)
}

// handleStop handles the "stop" command
func handleStop() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: app name is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli stop <name>\n\n")
		os.Exit(1)
	}

	name := os.Args[2]

	pm, err := NewProcessManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create process manager: %v\n", err)
		os.Exit(1)
	}

	if err := pm.Stop(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Stopped application '%s'\n", name)
}

// handleRestart handles the "restart" command for local processes
func handleRestart() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: app name is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli restart <name>\n\n")
		os.Exit(1)
	}

	name := os.Args[2]

	pm, err := NewProcessManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create process manager: %v\n", err)
		os.Exit(1)
	}

	if err := pm.Restart(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	proc, _ := pm.Get(name)
	fmt.Printf("✅ Restarted application '%s' (PID: %d)\n", name, proc.PID)
}

// handleList handles the "list" command
func handleList() {
	pm, err := NewProcessManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create process manager: %v\n", err)
		os.Exit(1)
	}

	processes := pm.List()

	if len(processes) == 0 {
		fmt.Println("No applications managed.")
		return
	}

	// Print header
	fmt.Printf("%-20s %-8s %-8s %-50s\n", "NAME", "STATUS", "PID", "WORKING DIR")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────")

	// Print processes
	for _, proc := range processes {
		status := proc.Status
		pidStr := strconv.Itoa(proc.PID)
		if proc.Status == StatusStopped {
			pidStr = "-"
		}

		fmt.Printf("%-20s %-8s %-8s %-50s\n", proc.Name, status, pidStr, proc.WorkingDir)
	}
}

// handleLogs handles the "logs" command
func handleLogs() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: app name is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli logs <name> [--lines N]\n")
		fmt.Fprintf(os.Stderr, "  --lines N: Show last N lines (default: all)\n\n")
		os.Exit(1)
	}

	name := os.Args[2]
	lines := 0

	// Parse flags
	if len(os.Args) >= 4 {
		for i := 3; i < len(os.Args); i++ {
			if os.Args[i] == "--lines" && i+1 < len(os.Args) {
				var err error
				lines, err = strconv.Atoi(os.Args[i+1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: invalid line count: %v\n", err)
					os.Exit(1)
				}
				break
			}
		}
	}

	pm, err := NewProcessManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create process manager: %v\n", err)
		os.Exit(1)
	}

	logs, err := pm.GetLogs(name, lines)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(logs)
}

// handleDelete handles the "delete" command
func handleDelete() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: app name is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli delete <name>\n\n")
		os.Exit(1)
	}

	name := os.Args[2]

	pm, err := NewProcessManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create process manager: %v\n", err)
		os.Exit(1)
	}

	if err := pm.Delete(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Deleted application '%s' from process list\n", name)
}

// handleStatus handles the "status" command for local processes
func handleStatus() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: app name is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli status <name>\n\n")
		os.Exit(1)
	}

	name := os.Args[2]

	pm, err := NewProcessManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create process manager: %v\n", err)
		os.Exit(1)
	}

	proc, err := pm.Get(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Application: %s\n", proc.Name)
	fmt.Printf("Status:      %s\n", proc.Status)
	if proc.Status == StatusOnline {
		fmt.Printf("PID:         %d\n", proc.PID)
	}
	fmt.Printf("Working Dir: %s\n", proc.WorkingDir)
	fmt.Printf("Command:     %s %v\n", proc.Command, proc.Args)
	fmt.Printf("Restarts:    %d\n", proc.Restarts)
	fmt.Printf("Auto-restart: %v\n", proc.AutoRestart)
	fmt.Printf("Created:     %s\n", proc.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", proc.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Log file:    %s\n", proc.LogFile)
	fmt.Printf("Error file:  %s\n", proc.ErrorFile)
}

// handleRun runs a script or Go file in the foreground (uv-style: fluxor run main.go, fluxor run app.py).
// Stdout/stderr are forwarded to the terminal; exit code is the process exit code.
func handleRun() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli run <file>\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli run main.go    Run Go file (go run main.go)\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli run .          Run current Go package (go run .)\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli run app.py    Run Python script\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli run server.js Run Node.js script\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli run plugin.so Load Fluxor verticle from Go plugin (linux/darwin)\n\n")
		os.Exit(1)
	}
	arg := os.Args[2]
	extraArgs := os.Args[3:]

	var cmd *exec.Cmd
	ext := strings.ToLower(filepath.Ext(arg))
	if arg == "." {
		ext = ".go" // fluxor run . → go run .
	}
	switch ext {
	case ".go":
		goPath, err := exec.LookPath("go")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: go not found in PATH\n")
			os.Exit(1)
		}
		cmd = exec.Command(goPath, append([]string{"run", arg}, extraArgs...)...)
	case ".py":
		pythonPath, err := exec.LookPath("python3")
		if err != nil {
			pythonPath, err = exec.LookPath("python")
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: python not found in PATH\n")
			os.Exit(1)
		}
		cmd = exec.Command(pythonPath, append([]string{arg}, extraArgs...)...)
	case ".js", ".mjs", ".cjs":
		nodePath, err := exec.LookPath("node")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: node not found in PATH\n")
			os.Exit(1)
		}
		cmd = exec.Command(nodePath, append([]string{arg}, extraArgs...)...)
	case ".so", ".dylib":
		handleRunPluginSO(arg)
		return
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported file extension %q (use .go, .py, .js)\n", ext)
		os.Exit(1)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir, _ = os.Getwd()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
