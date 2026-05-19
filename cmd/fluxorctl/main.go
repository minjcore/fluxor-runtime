// Copyright (c) 2024-2028 Fluxor Framework
// All rights reserved.

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fluxorio/fluxor/pkg/entrypoint"
)

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	path := os.Args[2]
	if strings.HasPrefix(path, "-") {
		fmt.Fprintf(os.Stderr, "Error: expected path to plugin, got %q\n\n", path)
		printUsage()
		os.Exit(1)
	}
	if !strings.HasSuffix(strings.ToLower(path), ".so") && !strings.HasSuffix(strings.ToLower(path), ".dylib") {
		fmt.Fprintf(os.Stderr, "Error: expected .so or .dylib plugin path, got %q\n\n", path)
		printUsage()
		os.Exit(1)
	}

	switch cmd {
	case "deploy", "run":
		if err := entrypoint.RunFromPluginSO(path); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "fluxorctl — run a Fluxor verticle from a Go plugin (same host contract as fluxor-cli).\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  fluxorctl deploy <path/to/plugin.{so,dylib}>\n")
	fmt.Fprintf(os.Stderr, "  fluxorctl run    <path/to/plugin.{so,dylib}>\n\n")
	fmt.Fprintf(os.Stderr, "The plugin must export: func NewVerticle() core.Verticle\n")
	fmt.Fprintf(os.Stderr, "Build: go build -buildmode=plugin -o myverticle.so <plugin package>\n\n")
	fmt.Fprintf(os.Stderr, "For YAML/VPS deploy, use fluxor-cli deploy -target ...\n")
}
