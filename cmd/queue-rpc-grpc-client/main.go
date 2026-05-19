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
	"fmt"
	"os"
	"time"

	pb "github.com/fluxorio/fluxor/proto/fluxor/queue"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Get server address from environment or use default
	address := getEnv("GRPC_ADDRESS", "localhost:50051")

	// Connect to gRPC server
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("Failed to connect to gRPC server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Create client
	client := pb.NewQueueRPCClient(conn)

	// Test cache keys
	testKeys := []string{"user:123", "config:app", "data:test", "not:found"}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Printf("Connecting to gRPC server at %s...\n", address)
	fmt.Println("Making test RPC calls...")

	for _, cacheKey := range testKeys {
		fmt.Printf("Making RPC call for cache key: %s\n", cacheKey)

		req := &pb.GetCacheRequest{
			CacheKey: cacheKey,
			Data:     map[string]string{"action": "get_info"},
		}

		response, err := client.GetCache(ctx, req)
		if err != nil {
			fmt.Printf("❌ RPC call failed for %s: %v\n\n", cacheKey, err)
			continue
		}

		if response.Success {
			fmt.Printf("✅ RPC Success for %s: %s\n\n", cacheKey, string(response.Data))
		} else {
			fmt.Printf("❌ RPC Error for %s: %s\n\n", cacheKey, response.Error)
		}

		// Wait between calls
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("Test RPC calls completed")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
