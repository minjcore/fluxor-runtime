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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/devops"
)

func runHealthCommand(args []string) error {
	healthCmd := flag.NewFlagSet("health", flag.ExitOnError)
	url := healthCmd.String("url", "", "URL to check (required)")
	timeout := healthCmd.Duration("timeout", 5*time.Second, "Timeout duration")
	format := healthCmd.String("format", "json", "Output format: json, text")
	healthCmd.Parse(args)

	if *url == "" {
		fmt.Fprintf(os.Stderr, "Error: -url is required\n\n")
		healthCmd.Usage()
		fmt.Fprintf(os.Stderr, "\nUsage: devopscli health -url <url> [options]\n")
		return fmt.Errorf("url is required")
	}

	return checkHealth(*url, *timeout, *format)
}

func checkHealth(url string, timeout time.Duration, format string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Perform HTTP health check
	start := time.Now()
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		check := devops.NewHealthCheck(
			devops.HealthStatusUnhealthy,
			fmt.Sprintf("Failed to create request: %v", err),
		)
		return outputHealthCheck(check, format, time.Since(start))
	}

	resp, err := client.Do(req)
	if err != nil {
		check := devops.NewHealthCheck(
			devops.HealthStatusUnhealthy,
			fmt.Sprintf("HTTP request failed: %v", err),
		)
		return outputHealthCheck(check, format, time.Since(start))
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	// Determine health status based on HTTP status code
	var status devops.HealthStatus
	var message string

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		status = devops.HealthStatusHealthy
		message = "Service is healthy"
	} else if resp.StatusCode >= 300 && resp.StatusCode < 500 {
		status = devops.HealthStatusDegraded
		message = fmt.Sprintf("Service returned status %d", resp.StatusCode)
	} else {
		status = devops.HealthStatusUnhealthy
		message = fmt.Sprintf("Service returned status %d", resp.StatusCode)
	}

	check := devops.NewHealthCheck(status, message).
		WithDetails("url", url).
		WithDetails("status_code", resp.StatusCode).
		WithDetails("duration_ms", duration.Milliseconds())

	// Try to parse response body as JSON for additional details
	if resp.ContentLength > 0 && resp.ContentLength < 1024*1024 { // Max 1MB
		var bodyData map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&bodyData); err == nil {
			check.WithDetails("response", bodyData)
		}
	}

	return outputHealthCheck(check, format, duration)
}

func outputHealthCheck(check *devops.HealthCheck, format string, duration time.Duration) error {
	switch strings.ToLower(format) {
	case "text":
		return outputTextFormat(check, duration)
	case "json":
		fallthrough
	default:
		return outputJSONFormat(check)
	}
}

func outputJSONFormat(check *devops.HealthCheck) error {
	data := map[string]interface{}{
		"status":    string(check.Status),
		"message":   check.Message,
		"timestamp": check.Timestamp.Format(time.RFC3339),
		"details":   check.Details,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func outputTextFormat(check *devops.HealthCheck, duration time.Duration) error {
	var statusIcon string
	switch check.Status {
	case devops.HealthStatusHealthy:
		statusIcon = "✅"
	case devops.HealthStatusDegraded:
		statusIcon = "⚠️"
	case devops.HealthStatusUnhealthy:
		statusIcon = "❌"
	default:
		statusIcon = "❓"
	}

	fmt.Printf("%s Status: %s\n", statusIcon, strings.ToUpper(string(check.Status)))
	fmt.Printf("Message: %s\n", check.Message)
	fmt.Printf("Duration: %v\n", duration.Round(time.Millisecond))
	fmt.Printf("Timestamp: %s\n", check.Timestamp.Format(time.RFC3339))

	if len(check.Details) > 0 {
		fmt.Println("\nDetails:")
		for key, value := range check.Details {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	return nil
}
