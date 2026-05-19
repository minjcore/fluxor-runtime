# DevOps CLI

DevOps CLI (`devopscli`) is a command-line tool providing DevOps utilities for Fluxor applications.

## Project Structure

The `devopscli` tool is organized into separate command files:

```
cmd/devopscli/
├── main.go      # Main CLI entry point and routing
├── health.go    # Health check subcommand
├── metrics.go   # Metrics collection subcommand
├── deploy.go    # Deployment subcommand
└── README.md    # This file
```

## Installation

Build from source:

```bash
go build -o bin/devopscli ./cmd/devopscli
```

Or install globally:

```bash
go install ./cmd/devopscli
```

## Commands

### Health Check

Check the health of HTTP endpoints:

```bash
devopscli health -url <url> [options]
```

**Options:**
- `-url` (required): URL to check
- `-timeout`: Timeout duration (default: 5s)
- `-format`: Output format - `json` or `text` (default: json)

**Examples:**

```bash
# Check health endpoint (JSON output)
devopscli health -url http://localhost:8080/health

# Check with custom timeout and text output
devopscli health -url http://localhost:8080/health -timeout 10s -format text

# Check remote service
devopscli health -url https://api.example.com/health
```

**Output Formats:**

**JSON format:**
```json
{
  "status": "healthy",
  "message": "Service is healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "details": {
    "url": "http://localhost:8080/health",
    "status_code": 200,
    "duration_ms": 45,
    "response": {
      "status": "UP",
      "service": "my-service"
    }
  }
}
```

**Text format:**
```
✅ Status: HEALTHY
Message: Service is healthy
Duration: 45ms
Timestamp: 2024-01-15T10:30:00Z

Details:
  url: http://localhost:8080/health
  status_code: 200
  duration_ms: 45
```

**Status Codes:**

- `healthy`: HTTP 2xx - Service is operational
- `degraded`: HTTP 3xx-4xx - Service is operational but with issues
- `unhealthy`: HTTP 5xx or connection errors - Service is not operational

**Exit Codes:**

- `0`: Health check successful (healthy or degraded)
- `1`: Health check failed (unhealthy or error)

### Version

Show version information:

```bash
devopscli version
# or
devopscli -v
# or
devopscli --version
```

## Use Cases

### CI/CD Pipelines

Use in your CI/CD pipelines to verify service health before deployment:

```bash
#!/bin/bash
if devopscli health -url http://localhost:8080/health; then
    echo "Service is healthy, proceeding with deployment"
    # Continue deployment
else
    echo "Service health check failed, aborting deployment"
    exit 1
fi
```

### Monitoring Scripts

Integrate with monitoring systems:

```bash
# Check multiple services
for url in \
    "http://service1:8080/health" \
    "http://service2:8080/health" \
    "http://service3:8080/health"
do
    devopscli health -url "$url" -format json >> health_checks.log
done
```

### Docker Health Checks

Use in Docker health checks:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD devopscli health -url http://localhost:8080/health -timeout 2s || exit 1
```

## Future Features

The DevOps CLI is designed to be extended with additional commands:

- **Metrics**: Collect and export metrics from Fluxor applications
- **Deploy**: Deploy Fluxor applications with blue-green support
- **Monitor**: Real-time monitoring dashboard
- **Logs**: Log aggregation and analysis
- **Status**: Show status of multiple services

## Integration with Fluxor

The DevOps CLI works with any HTTP health endpoint. Fluxor applications can use the `pkg/web/health` package to create standardized health endpoints:

```go
import "github.com/fluxorio/fluxor/pkg/web/health"

router.GETFast("/health", health.Handler())
router.GETFast("/ready", health.ReadyHandler())
```

Then check them with:

```bash
devopscli health -url http://localhost:8080/health
devopscli health -url http://localhost:8080/ready
```

## See Also

- [DevOps Package](../../pkg/devops/README.md) - Core DevOps utilities
- [Health Package](../../pkg/web/health/) - Health check implementation for Fluxor apps
- [Fluxor Documentation](../../DOCUMENTATION.md) - Complete Fluxor documentation

