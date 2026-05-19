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
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	endpoint := flag.String("endpoint", "", "Metrics endpoint URL (e.g., http://localhost:8080/metrics)")
	format := flag.String("format", "prometheus", "Output format: prometheus, json")
	output := flag.String("output", "", "Output file (default: stdout)")
	interval := flag.Duration("interval", 0, "Collection interval (0 = single collection)")
	flag.Parse()

	if *endpoint == "" {
		fmt.Fprintf(os.Stderr, "Error: -endpoint is required\n\n")
		flag.Usage()
		fmt.Fprintf(os.Stderr, "\nUsage: metrics-collector -endpoint <url> [options]\n")
		os.Exit(1)
	}

	if *interval > 0 {
		// Continuous collection
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()

		fmt.Fprintf(os.Stderr, "Collecting metrics from %s every %v\n", *endpoint, *interval)
		fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop\n\n")

		// First collection immediately
		if err := collectMetrics(*endpoint, *format, *output); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Then collect on interval
		for range ticker.C {
			if err := collectMetrics(*endpoint, *format, *output); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				// Continue collection on error
			}
		}
	} else {
		// Single collection
		if err := collectMetrics(*endpoint, *format, *output); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func collectMetrics(endpoint, format, outputFile string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to fetch metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("metrics endpoint returned status %d", resp.StatusCode)
	}

	// Read response body
	buf := make([]byte, 64*1024) // 64KB buffer
	var data []byte
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Output metrics
	var output *os.File = os.Stdout
	if outputFile != "" {
		// Validate output file path to prevent directory traversal attacks
		if err := validateOutputPath(outputFile); err != nil {
			return fmt.Errorf("invalid output file path: %w", err)
		}

		var err error
		// #nosec G304 -- path is validated by validateOutputPath() to prevent directory traversal attacks
		output, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer output.Close()
	}

	if _, err := output.Write(data); err != nil {
		return fmt.Errorf("failed to write metrics: %w", err)
	}

	if outputFile != "" {
		fmt.Fprintf(os.Stderr, "Metrics saved to %s\n", outputFile)
	}

	return nil
}

// validateOutputPath validates the output file path to prevent directory traversal attacks.
func validateOutputPath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for directory traversal sequences in the original path
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains directory traversal sequence")
	}

	// Resolve absolute path to normalize it
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Clean the path to resolve any remaining ".." or "." components
	cleanPath := filepath.Clean(absPath)

	// After cleaning, if the path still contains "..", it indicates
	// an attempt to traverse outside the filesystem root
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains directory traversal sequence")
	}

	return nil
}
