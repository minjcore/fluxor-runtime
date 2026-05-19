package jsons

import (
	"fmt"
	"time"
)

// User represents a common user structure for benchmarking
type User struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Email    string                 `json:"email"`
	Age      int                    `json:"age"`
	Active   bool                   `json:"active"`
	Created  time.Time              `json:"created_at"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NestedData represents a complex nested structure for benchmarking
type NestedData struct {
	Users []User `json:"users"`
	Count int    `json:"count"`
	Meta  struct {
		Page     int      `json:"page"`
		Pages    int      `json:"pages"`
		Tags     []string `json:"tags"`
		Settings map[string]interface{} `json:"settings"`
	} `json:"meta"`
}

// EventBusMessage represents a typical EventBus message structure
type EventBusMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Headers   map[string]string      `json:"headers"`
}

// APIMessage represents a typical REST API response structure
type APIMessage struct {
	Success bool                   `json:"success"`
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
	Meta    map[string]interface{} `json:"meta"`
}

// ConfigData represents a typical configuration file structure
type ConfigData struct {
	App struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Env     string `json:"env"`
	} `json:"app"`
	Server struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"server"`
	Database map[string]interface{} `json:"database"`
	Features map[string]bool        `json:"features"`
}

// BenchmarkResult represents the result of a benchmark operation
type BenchmarkResult struct {
	Library     string
	Operation   string
	PayloadSize int
	Duration    time.Duration
	AllocBytes  uint64
	AllocCount  uint64
	Throughput  float64 // ops/sec
}

// generateSmallUser generates a small user object (< 100 bytes)
func generateSmallUser() User {
	return User{
		ID:      "12345",
		Name:    "John Doe",
		Email:   "john@example.com",
		Age:     30,
		Active:  true,
		Created: time.Now(),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
}

// generateMediumUsers generates a medium-sized array of users (1KB-100KB)
func generateMediumUsers(count int) []User {
	users := make([]User, count)
	for i := 0; i < count; i++ {
		users[i] = User{
			ID:      fmt.Sprintf("user-%d", i+10000),
			Name:    fmt.Sprintf("User %d", i),
			Email:   fmt.Sprintf("user%d@example.com", i),
			Age:     20 + (i % 50),
			Active:  i%2 == 0,
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"index":    i,
				"category": "test",
				"score":    float64(i) * 1.5,
			},
		}
	}
	return users
}

// generateLargeNestedData generates a large nested structure (100KB-10MB)
func generateLargeNestedData(userCount int) NestedData {
	users := generateMediumUsers(userCount)
	nested := NestedData{
		Users: users,
		Count: userCount,
	}
	nested.Meta.Page = 1
	nested.Meta.Pages = 10
	nested.Meta.Tags = []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
	nested.Meta.Settings = map[string]interface{}{
		"setting1": "value1",
		"setting2": 12345,
		"setting3": true,
		"setting4": []string{"a", "b", "c"},
	}
	return nested
}

// generateEventBusMessage generates a typical EventBus message
func generateEventBusMessage() EventBusMessage {
	return EventBusMessage{
		ID:        "msg-12345",
		Type:      "user.created",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"user_id": "12345",
			"action":  "created",
			"status":  "active",
		},
		Headers: map[string]string{
			"content-type": "application/json",
			"source":       "api",
		},
	}
}

// generateAPIMessage generates a typical REST API response
func generateAPIMessage() APIMessage {
	return APIMessage{
		Success: true,
		Code:    200,
		Message: "Success",
		Data: map[string]interface{}{
			"id":       "12345",
			"name":     "Item Name",
			"quantity": 100,
			"price":    29.99,
		},
		Meta: map[string]interface{}{
			"request_id": "req-12345",
			"timestamp":  time.Now().Unix(),
		},
	}
}

// generateConfigData generates a typical configuration structure
func generateConfigData() ConfigData {
	var config ConfigData
	config.App.Name = "test-app"
	config.App.Version = "1.0.0"
	config.App.Env = "production"
	config.Server.Host = "0.0.0.0"
	config.Server.Port = 8080
	config.Database = map[string]interface{}{
		"host":     "localhost",
		"port":     5432,
		"name":     "testdb",
		"username": "testuser",
		"ssl":      true,
	}
	config.Features = map[string]bool{
		"feature1": true,
		"feature2": false,
		"feature3": true,
	}
	return config
}
