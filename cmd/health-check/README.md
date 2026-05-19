# Healthcheck - Standalone Health Check Tool

A standalone tool for checking the health of HTTP endpoints. This is a focused version of the health check functionality from `devopscli`.

## Installation

Build from source:

```bash
go build -o bin/healthcheck ./cmd/healthcheck
```

Or install globally:

```bash
go install ./cmd/healthcheck
```

## Usage

```bash
healthcheck -url <url> [options]
```

**Options:**
- `-url` (required): URL to check
- `-timeout`: Timeout duration (default: 5s)
- `-format`: Output format - `json` or `text` (default: json)

**Examples:**

```bash
# Check health endpoint (JSON output)
healthcheck -url http://localhost:8080/health

# Check with custom timeout and text output
healthcheck -url http://localhost:8080/health -timeout 10s -format text

# Check remote service
healthcheck -url https://api.example.com/health
```

## Exit Codes

- `0`: Service is healthy or degraded
- `1`: Service is unhealthy or error occurred

## Use Cases

### Docker Health Checks

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD healthcheck -url http://localhost:8080/health -timeout 2s || exit 1
```

### CI/CD Pipelines

```bash
#!/bin/bash
if healthcheck -url http://localhost:8080/health; then
    echo "Service is healthy, proceeding with deployment"
    # Continue deployment
else
    echo "Service health check failed, aborting deployment"
    exit 1
fi
```

### Monitoring Scripts

```bash
#!/bin/bash
for url in \
    "http://service1:8080/health" \
    "http://service2:8080/health" \
    "http://service3:8080/health"
do
    healthcheck -url "$url" -format json >> health_checks.log
done
```

## Differences from devopscli

- `healthcheck` is a standalone, focused tool for health checks only
- `devopscli health` is part of a larger CLI tool with multiple commands
- Both use the same underlying implementation and provide the same functionality
- Choose `healthcheck` for simple scripts and Docker health checks
- Choose `devopscli` for workflows that need multiple DevOps operations

