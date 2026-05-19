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
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "health":
		err = runHealthCommand(os.Args[2:])
	case "metrics":
		err = runMetricsCommand(os.Args[2:])
	case "deploy":
		err = runDeployCommand(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("devopscli version %s\n", version)
		return
	default:
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "DevOps CLI - DevOps utilities for Fluxor applications\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  devopscli health -url <url>        Check health of an HTTP endpoint\n")
	fmt.Fprintf(os.Stderr, "  devopscli metrics -endpoint <url>  Collect metrics from an endpoint\n")
	fmt.Fprintf(os.Stderr, "  devopscli deploy -target <target>  Deploy application\n")
	fmt.Fprintf(os.Stderr, "  devopscli version                  Show version\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  devopscli health -url http://localhost:8080/health\n")
	fmt.Fprintf(os.Stderr, "  devopscli health -url http://localhost:8080/health -timeout 10s -format text\n")
	fmt.Fprintf(os.Stderr, "  devopscli metrics -endpoint http://localhost:8080/metrics\n")
	fmt.Fprintf(os.Stderr, "  devopscli deploy -target production -version 1.0.0\n\n")
}
