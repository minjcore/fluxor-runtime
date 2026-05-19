// Copyright (c) 2024-2028 Fluxor Framework
// All rights reserved.

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fluxorio/fluxor/pkg/entrypoint"
)

// handleRunPluginSO runs a verticle loaded from a Go plugin (.so / .dylib).
func handleRunPluginSO(pluginPath string) {
	if err := entrypoint.RunFromPluginSO(pluginPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// deployStartsWithPluginPath returns true when "deploy" is followed by a plugin path
// (not a flag), so we run the plugin host instead of devops deploy.
func deployStartsWithPluginPath(args []string) (pluginPath string, ok bool) {
	if len(args) < 1 {
		return "", false
	}
	p := args[0]
	if p == "" || p[0] == '-' {
		return "", false
	}
	low := strings.ToLower(p)
	if strings.HasSuffix(low, ".so") || strings.HasSuffix(low, ".dylib") {
		return p, true
	}
	return "", false
}
