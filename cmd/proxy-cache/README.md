# ProxyCache - High-Performance Package Caching Proxy

A blazing-fast caching proxy server for package managers built on the fluxor-cache architecture. Supports Go modules, npm, pip, and other package managers with intelligent caching strategies.

## Features

- 🚀 **High Performance**: Multi-tier caching (memory → disk) for optimal speed
- 💾 **Multiple Cache Backends**: Memory, Redis, or disk-based caching
- 🔄 **LRU Eviction**: Automatic cleanup with configurable size limits
- 📊 **Built-in Metrics**: Real-time statistics and health endpoints
- 🛡️ **Fail-Fast Design**: Robust error handling and validation
- ⚡ **Graceful Shutdown**: Clean termination with request draining
- 🔧 **Flexible Configuration**: CLI flags or JSON config file

## Quick Start

### Using Go

```bash
# Install
go install github.com/fluxorio/fluxor/cmd/proxycache@latest

# Run with defaults
proxycache

# Run with custom settings
proxycache -port 8080 -cache ./mycache -upstream https://proxy.golang.org
```

### From Source

```bash
# Clone the repository
cd cmd/proxycache

# Run directly
go run . -port 8080 -cache ./mycache

# Build binary
go build -o proxycache .
./proxycache
```

## Configuration

### Command Line Flags

```bash
proxycache \
  -port 8080 \
  -cache ./proxycache \
  -cache-type memory \
  -upstream https://proxy.golang.org \
  -ttl 24h \
  -max-size 10737418240 \
  -redis localhost:6379 \
  -v
```

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | 8080 | Port to listen on |
| `-cache` | ./proxycache | Cache directory path |
| `-cache-type` | memory | Cache backend: memory, redis, or disk |
| `-upstream` | https://proxy.golang.org | Upstream proxy URL |
| `-ttl` | 24h | Cache TTL duration |
| `-max-size` | 10GB | Maximum cache size in bytes |
| `-redis` | localhost:6379 | Redis address (if using redis) |
| `-v` | false | Enable verbose logging |

### JSON Configuration File

Create `config.json`:

```json
{
  "port": 8080,
  "host": "0.0.0.0",
  "verbose": false,
  "cache_dir": "./proxycache",
  "cache_type": "memory",
  "cache_ttl": 86400000000000,
  "max_cache_size": 10737418240,
  "redis_addr": "localhost:6379",
  "redis_password": "",
  "redis_db": 0,
  "upstream": "https://proxy.golang.org",
  "upstream_timeout": 30000000000,
  "allowed_origins": ["*"],
  "api_key": "",
  "max_concurrent": 100,
  "rate_limit": 1000,
  "cleanup_interval": 600000000000,
  "max_file_age": 604800000000000
}
```

## Usage Examples

### Go Module Proxy

```bash
# Start the proxy
proxycache -port 8080 -upstream https://proxy.golang.org

# Configure Go to use the proxy
export GOPROXY=http://localhost:8080,direct

# Download modules (cached on subsequent requests)
go get github.com/gin-gonic/gin@latest
go get github.com/gorilla/mux@v1.8.0
```

### NPM Registry Proxy

```bash
# Start with NPM upstream
proxycache -port 8080 -upstream https://registry.npmjs.org

# Configure npm
npm config set registry http://localhost:8080

# Install packages
npm install express
npm install react
```

### Python PyPI Proxy

```bash
# Start with PyPI upstream
proxycache -port 8080 -upstream https://pypi.org/simple

# Configure pip
pip config set global.index-url http://localhost:8080

# Install packages
pip install requests
pip install django
```

### Docker with Redis Cache

```bash
# Start Redis
docker run -d --name redis -p 6379:6379 redis:alpine

# Start proxy with Redis backend
proxycache -cache-type redis -redis localhost:6379 -port 8080
```

## API Endpoints

### Proxy Endpoint

```bash
# All requests to / are proxied and cached
curl http://localhost:8080/github.com/gin-gonic/gin/@v/v1.9.1.info
```

### Statistics

```bash
# Get cache statistics
curl http://localhost:8080/_stats
```

Response:
```json
{
  "hits": 1543,
  "misses": 234,
  "errors": 2,
  "total_requests": 1777,
  "hit_rate": 86.83,
  "cache_bytes": 1073741824,
  "disk_cache_size": 536870912,
  "disk_cache_files": 458
}
```

### Health Check

```bash
# Health check endpoint
curl http://localhost:8080/_health
```

Response:
```json
{
  "status": "healthy",
  "cache_root": "./proxycache",
  "upstream": "https://proxy.golang.org"
}
```

## Architecture

### Multi-Tier Caching

```
Request → Memory Cache → Disk Cache → Upstream
            ↓ Hit           ↓ Hit        ↓ Fetch
         Response        Response      Cache & Response
```

1. **Memory Cache**: Ultra-fast in-memory cache for frequently accessed packages
2. **Disk Cache**: Persistent storage with LRU eviction
3. **Upstream**: Original package registry

### Cache Key Strategy

Files are cached using SHA-256 hash with two-level directory structure:

```
proxycache/
  ├── ab/
  │   └── cd/
  │       └── abcdef123456... (cached file)
  └── ef/
      └── gh/
          └── efgh789012... (cached file)
```

## Performance

### Benchmarks

Tested on MacBook Pro M1 with 16GB RAM:

| Scenario | Requests/sec | Avg Latency | Cache Hit Rate |
|----------|--------------|-------------|----------------|
| Memory Cache Hit | 50,000+ | 0.2ms | 95%+ |
| Disk Cache Hit | 10,000+ | 2ms | 90%+ |
| Cache Miss | 100 | 300ms | N/A |

### Load Testing

```bash
# Run k6 load test
cd ../../loadtest
k6 run --vus 100 --duration 30s load_test.js
```

## Production Deployment

### Systemd Service

Create `/etc/systemd/system/proxycache.service`:

```ini
[Unit]
Description=ProxyCache Package Caching Proxy
After=network.target

[Service]
Type=simple
User=proxycache
WorkingDirectory=/opt/proxycache
ExecStart=/usr/local/bin/proxycache -port 8080 -cache /var/cache/proxycache
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable proxycache
sudo systemctl start proxycache
sudo systemctl status proxycache
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o proxycache ./cmd/proxycache

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/proxycache .
EXPOSE 8080
CMD ["./proxycache"]
```

Build and run:
```bash
docker build -t proxycache .
docker run -d -p 8080:8080 -v /var/cache/proxycache:/cache proxycache \
  -cache /cache -port 8080
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: proxycache
spec:
  replicas: 3
  selector:
    matchLabels:
      app: proxycache
  template:
    metadata:
      labels:
        app: proxycache
    spec:
      containers:
      - name: proxycache
        image: proxycache:latest
        ports:
        - containerPort: 8080
        env:
        - name: CACHE_TYPE
          value: "redis"
        - name: REDIS_ADDR
          value: "redis:6379"
        volumeMounts:
        - name: cache
          mountPath: /cache
      volumes:
      - name: cache
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: proxycache
spec:
  selector:
    app: proxycache
  ports:
  - port: 80
    targetPort: 8080
```

## Monitoring

### Prometheus Metrics

Export cache metrics to Prometheus:

```go
// Add to main.go
import "github.com/prometheus/client_golang/prometheus/promhttp"

mux.Handle("/metrics", promhttp.Handler())
```

### Grafana Dashboard

Import the sample dashboard from `../../observability/grafana-dashboard.json`

Key metrics:
- Cache hit rate
- Request latency
- Cache size
- Error rate

## Troubleshooting

### High Cache Miss Rate

```bash
# Check cache configuration
curl http://localhost:8080/_stats

# Increase TTL
proxycache -ttl 48h

# Increase max cache size
proxycache -max-size 21474836480  # 20GB
```

### Disk Space Issues

```bash
# Monitor cache size
du -sh ./proxycache

# Reduce max cache size
proxycache -max-size 5368709120  # 5GB

# Manual cleanup
rm -rf ./proxycache/*
```

### Connection Issues

```bash
# Test upstream connectivity
curl -I https://proxy.golang.org

# Check health endpoint
curl http://localhost:8080/_health

# Enable verbose logging
proxycache -v
```

## Development

### Running Tests

```bash
cd cmd/proxycache
go test -v ./...
```

### Building from Source

```bash
# Build for current platform
go build -o proxycache .

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o proxycache-linux .

# Build for macOS
GOOS=darwin GOARCH=arm64 go build -o proxycache-mac .
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE.md for details

## Related Projects

- [fluxor-cache](https://github.com/quadgate/fluxor-cache) - Original inspiration
- [Athens](https://github.com/gomods/athens) - Go module proxy
- [Verdaccio](https://verdaccio.org/) - npm proxy

## Support

- GitHub Issues: https://github.com/fluxorio/fluxor/issues
- Documentation: https://fluxor.io/docs
- Community: https://discord.gg/fluxor
