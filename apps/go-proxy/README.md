# GoProxy - Go Module Registry

A Maven-like Go module proxy and registry implementing the GOPROXY protocol with S3 storage, basic authentication, and a web UI for browsing/searching modules.

## Features

- **GOPROXY Protocol Compliant**: Works seamlessly with `go get`, `go mod download`, etc.
- **S3 Storage Backend**: Scalable, cloud-native storage using any S3-compatible service
- **Basic Authentication**: Protect your private modules with username/password auth
- **Web UI Dashboard**: Browse, search, and manage modules through a modern web interface
- **REST API**: Full API for programmatic access and integration
- **Built on Fluxor**: High-performance, event-driven architecture

## Quick Start

### 1. Configure

Edit `config.json` to set up your S3 credentials and authentication:

```json
{
  "server": {
    "address": ":8080"
  },
  "storage": {
    "type": "s3",
    "s3": {
      "endpoint": "s3.amazonaws.com",
      "bucket": "your-go-modules-bucket",
      "region": "us-east-1",
      "accessKey": "YOUR_ACCESS_KEY",
      "secretKey": "YOUR_SECRET_KEY"
    }
  },
  "auth": {
    "enabled": true,
    "users": [
      {
        "username": "admin",
        "password": "$2a$10$..."
      }
    ]
  }
}
```

### 2. Generate Password Hash

Use bcrypt to hash your password:

```go
package main

import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    password := "your-password"
    hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    fmt.Println(string(hash))
}
```

### 3. Run

```bash
cd apps/goproxy
go run .
```

### 4. Configure Go

Set the `GOPROXY` environment variable:

```bash
# Without auth (if auth is disabled)
export GOPROXY=http://localhost:8080,direct

# With basic auth
export GOPROXY=http://username:password@localhost:8080,direct
```

### 5. Use

```bash
# Download modules through the proxy
go get github.com/example/module

# Upload a module (via API)
curl -u admin:password -F "zip=@module.zip" -F "mod=@go.mod" \
  http://localhost:8080/api/v1/modules/github.com%2Fexample%2Fmodule/versions/v1.0.0
```

## Configuration

### Server Options

| Option | Description | Default |
|--------|-------------|---------|
| `address` | HTTP server bind address | `:8080` |
| `readTimeout` | Request read timeout | `30s` |
| `writeTimeout` | Response write timeout | `30s` |
| `maxQueue` | Max queued requests | `10000` |
| `workers` | Number of worker goroutines | `100` |

### S3 Storage Options

| Option | Description | Default |
|--------|-------------|---------|
| `endpoint` | S3 endpoint URL | `s3.amazonaws.com` |
| `bucket` | S3 bucket name | `go-modules` |
| `region` | AWS region | `us-east-1` |
| `accessKey` | AWS access key ID | (from env) |
| `secretKey` | AWS secret access key | (from env) |
| `forcePathStyle` | Use path-style URLs (for MinIO) | `false` |
| `disableSSL` | Disable SSL/TLS | `false` |

### Authentication Options

| Option | Description | Default |
|--------|-------------|---------|
| `enabled` | Enable authentication | `true` |
| `users` | Array of user credentials | `[]` |

## API Reference

### GOPROXY Protocol Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /{module}/@v/list` | List available versions |
| `GET /{module}/@v/{version}.info` | Version metadata (JSON) |
| `GET /{module}/@v/{version}.mod` | go.mod file contents |
| `GET /{module}/@v/{version}.zip` | Module source archive |
| `GET /{module}/@latest` | Latest version info |

### REST API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/modules` | GET | List all modules |
| `/api/v1/modules/search` | GET | Search modules |
| `/api/v1/modules/:module` | GET | Get module details |
| `/api/v1/modules/:module/versions` | GET | List module versions |
| `/api/v1/modules/:module/versions/:version` | GET | Get version details |
| `/api/v1/modules/:module/versions/:version` | POST | Upload new version |
| `/api/v1/modules/:module/versions/:version` | DELETE | Delete version |
| `/api/v1/modules/:module/stats` | GET | Get module statistics |
| `/api/v1/health` | GET | Health check |
| `/api/v1/stats` | GET | Overall statistics |

## Using with MinIO

For local development or self-hosted S3-compatible storage, use MinIO:

```json
{
  "storage": {
    "type": "s3",
    "s3": {
      "endpoint": "http://localhost:9000",
      "bucket": "go-modules",
      "region": "us-east-1",
      "accessKey": "minioadmin",
      "secretKey": "minioadmin",
      "forcePathStyle": true,
      "disableSSL": true
    }
  }
}
```

## S3 Object Layout

The registry stores modules with the following S3 key structure:

```
modules/{escaped_module}/@v/list          # Version list (text, one per line)
modules/{escaped_module}/@v/{version}.info # Version metadata (JSON)
modules/{escaped_module}/@v/{version}.mod  # go.mod file (text)
modules/{escaped_module}/@v/{version}.zip  # Source archive (binary)
metadata/modules.json                       # Module index for search
```

Module paths are escaped according to the Go module specification:
- Uppercase letters are replaced with `!` + lowercase (e.g., `Example` → `!example`)

## Architecture

```
┌─────────────────┐     ┌─────────────────┐
│   go get/mod    │     │   Web Browser   │
└────────┬────────┘     └────────┬────────┘
         │                       │
         ▼                       ▼
┌────────────────────────────────────────┐
│            GoProxy Server              │
│  ┌──────────────────────────────────┐  │
│  │        Basic Auth Middleware     │  │
│  └──────────────────────────────────┘  │
│  ┌──────────────┐  ┌────────────────┐  │
│  │ Proxy Handler│  │   API Handler  │  │
│  │  (GOPROXY)   │  │   (REST API)   │  │
│  └──────┬───────┘  └───────┬────────┘  │
│         │                  │           │
│         ▼                  ▼           │
│  ┌──────────────────────────────────┐  │
│  │         Storage Interface        │  │
│  └──────────────────────────────────┘  │
└────────────────────┬───────────────────┘
                     │
                     ▼
          ┌────────────────────┐
          │    S3 Storage      │
          │  (AWS/MinIO/etc)   │
          └────────────────────┘
```

## Development

### Building

```bash
cd apps/goproxy
go build -o goproxy .
```

### Running Tests

```bash
go test ./...
```

### Project Structure

```
apps/goproxy/
├── main.go              # Entry point
├── config.go            # Configuration structs
├── config.json          # Default configuration
├── verticle.go          # Main Fluxor verticle
├── domain/
│   ├── module.go        # Module entity
│   ├── version.go       # Version entity
│   └── errors.go        # Domain errors
├── storage/
│   ├── interface.go     # Storage interface
│   ├── s3_storage.go    # S3 implementation
│   └── metadata.go      # Metadata index
├── handlers/
│   ├── proxy_handler.go # GOPROXY protocol
│   ├── api_handler.go   # REST API
│   └── auth_middleware.go # Authentication
├── web/
│   ├── index.html       # Web UI
│   └── dashboard.js     # UI JavaScript
└── README.md            # This file
```

## License

MIT License - see LICENSE file for details.
