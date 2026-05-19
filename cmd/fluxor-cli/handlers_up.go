package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

// handleUpStaticsite serves a static site from a directory (e.g. fluxor-cli up staticsite).
func handleUpStaticsite() {
	upCmd := flag.NewFlagSet("up staticsite", flag.ExitOnError)
	dir := upCmd.String("dir", ".", "Directory to serve (default: current directory)")
	port := upCmd.String("port", "8080", "Port to listen on")
	_ = upCmd.Parse(os.Args[3:])

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid dir: %v\n", err)
		os.Exit(1)
	}
	fi, err := os.Stat(absDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: dir not found: %s (%v)\n", absDir, err)
		os.Exit(1)
	}
	if !fi.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: not a directory: %s\n", absDir)
		os.Exit(1)
	}

	addr := ":" + *port
	fs := http.FileServer(http.Dir(absDir))
	http.Handle("/", http.StripPrefix("", fs))

	fmt.Printf("Serving static site at http://localhost%s from %s\n", addr, absDir)
	fmt.Printf("Press Ctrl+C to stop.\n")

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("\nStopping...")
}
