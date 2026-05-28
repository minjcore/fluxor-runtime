// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package main

// Config holds the configuration for the Go module proxy
type Config struct {
	Server  ServerConfig  `json:"server"`
	Storage StorageConfig `json:"storage"`
	Auth    AuthConfig    `json:"auth"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Address      string `json:"address"`
	ReadTimeout  string `json:"readTimeout"`
	WriteTimeout string `json:"writeTimeout"`
	MaxQueue     int    `json:"maxQueue"`
	Workers      int    `json:"workers"`
}

// StorageConfig holds storage backend configuration
type StorageConfig struct {
	Type     string   `json:"type"`     // "s3", "filesystem", or "mirrored"
	BasePath string   `json:"basePath"` // for filesystem backend
	S3       S3Config `json:"s3"`
	Mirror   *MirrorConfig `json:"mirror,omitempty"` // optional OSS mirror for filesystem
}

// MirrorConfig configures an S3/OSS mirror for the filesystem backend
type MirrorConfig struct {
	S3 S3Config `json:"s3"`
}

// S3Config holds S3-specific configuration
type S3Config struct {
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	AccessKey       string `json:"accessKey"`
	SecretKey       string `json:"secretKey"`
	ForcePathStyle  bool   `json:"forcePathStyle"`  // For MinIO compatibility
	DisableSSL      bool   `json:"disableSSL"`      // For local development
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled bool       `json:"enabled"`
	Users   []UserAuth `json:"users"`
}

// UserAuth represents a user with basic auth credentials
type UserAuth struct {
	Username string `json:"username"`
	Password string `json:"password"` // bcrypt hashed password
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Address:      ":8080",
			ReadTimeout:  "30s",
			WriteTimeout: "30s",
			MaxQueue:     10000,
			Workers:      100,
		},
		Storage: StorageConfig{
			Type: "s3",
			S3: S3Config{
				Endpoint:       "s3.amazonaws.com",
				Bucket:         "go-modules",
				Region:         "us-east-1",
				ForcePathStyle: false,
				DisableSSL:     false,
			},
		},
		Auth: AuthConfig{
			Enabled: true,
			Users: []UserAuth{
				{Username: "admin", Password: "$2a$10$..."},  // placeholder
			},
		},
	}
}
