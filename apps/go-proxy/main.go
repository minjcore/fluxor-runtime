// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"log"

	"github.com/fluxorio/fluxor/pkg/entrypoint"
)

const (
	Version = "1.0.0"
	Name    = "GoProxy"
	Banner  = `
 ██████╗  ██████╗ ██████╗ ██████╗  ██████╗ ██╗  ██╗██╗   ██╗
██╔════╝ ██╔═══██╗██╔══██╗██╔══██╗██╔═══██╗╚██╗██╔╝╚██╗ ██╔╝
██║  ███╗██║   ██║██████╔╝██████╔╝██║   ██║ ╚███╔╝  ╚████╔╝ 
██║   ██║██║   ██║██╔═══╝ ██╔══██╗██║   ██║ ██╔██╗   ╚██╔╝  
╚██████╔╝╚██████╔╝██║     ██║  ██║╚██████╔╝██╔╝ ██╗   ██║   
 ╚═════╝  ╚═════╝ ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   
`
)

func printBanner() {
	fmt.Println(Banner)
	fmt.Printf("%s v%s - Go Module Proxy & Registry\n", Name, Version)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("• GOPROXY Protocol  • S3 Storage Backend")
	fmt.Println("• Basic Auth        • Web UI Dashboard")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

func main() {
	printBanner()
	log.Printf("Starting %s v%s...", Name, Version)

	// Create Fluxor application
	app, err := entrypoint.NewMainVerticle("")
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Deploy GoProxy verticle
	verticle := NewGoProxyVerticle()
	_, err = app.DeployVerticle(verticle)
	if err != nil {
		log.Fatalf("Failed to deploy %s verticle: %v", Name, err)
	}

	// Start application (blocks until SIGINT/SIGTERM)
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Stop application
	if err := app.Stop(); err != nil {
		log.Printf("Error stopping application: %v", err)
	}

	log.Printf("%s stopped gracefully", Name)
}
