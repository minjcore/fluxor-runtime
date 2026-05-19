//go:build redis
// +build redis

package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// GoRedisClient wraps go-redis client to implement cache.RedisClient interface
type GoRedisClient struct {
	client *redis.Client
}

// NewGoRedisClient creates a new Redis client wrapper
func NewGoRedisClient(addr, password string, db int) (*GoRedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &GoRedisClient{client: client}, nil
}

// Get retrieves a value by key
func (r *GoRedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set stores a value by key with expiration
func (r *GoRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Del removes one or more keys
func (r *GoRedisClient) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// FlushDB removes all keys from the current database
func (r *GoRedisClient) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

// Exists checks if one or more keys exist
func (r *GoRedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// TTL returns the remaining TTL for a key
func (r *GoRedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// Close closes the Redis connection
func (r *GoRedisClient) Close() error {
	return r.client.Close()
}

// Example usage with Redis cache:
//
// To use Redis cache, uncomment the following code in main():
//
// case "redis":
//     redisClient, err := NewGoRedisClient(*redisAddr, "", 0)
//     if err != nil {
//         log.Fatalf("Failed to connect to Redis: %v", err)
//     }
//     defer redisClient.Close()
//
//     c = cache.NewRedisCache(redisClient, "pkg:")
//     log.Printf("Using Redis cache at %s", *redisAddr)
//
// And add to go.mod:
// require github.com/redis/go-redis/v9 v9.0.0
