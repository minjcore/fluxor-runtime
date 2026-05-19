# Fluxor Enterprise Example

A production-ready example demonstrating all enterprise features of Fluxor framework.

## Features Demonstrated

### ✅ P0 Enterprise Features (All Implemented)

1. **Structured Logging with Context**
   - JSON structured logging
   - Request ID tracking
   - Contextual fields (user_id, request_id, etc.)

2. **OpenTelemetry Distributed Tracing**
   - Jaeger/Zipkin integration
   - Automatic span creation
   - Context propagation across services

3. **Authentication & Authorization**
   - JWT authentication middleware
   - RBAC (Role-Based Access Control)
   - API key authentication
   - OAuth2 support

4. **Database Connection Pooling**
   - HikariCP-equivalent pooling
   - Connection lifecycle management
   - Health monitoring

5. **Configuration Management**
   - YAML/JSON config files
   - Environment variable overrides
   - Validation and type safety

6. **Enhanced Health Checks**
   - Liveness endpoints
   - Readiness endpoints
   - Dependency health aggregation
   - Database health checks

7. **Security Middleware**
   - CORS configuration
   - Security headers (HSTS, CSP, X-Frame-Options)
   - Rate limiting (IP-based)
   - Request sanitization

8. **Prometheus Metrics Export**
   - Custom metrics API
   - `/metrics` endpoint
   - Grafana-ready format

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    API Gateway Layer                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐│
│  │   CORS   │→ │ Security │→ │   Rate   │→ │  Auth   ││
│  │          │  │ Headers  │  │ Limiting │  │   JWT   ││
│  └──────────┘  └──────────┘  └──────────┘  └─────────┘│
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│                  Business Logic Layer                    │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐│
│  │     User     │  │   Product    │  │     Order     ││
│  │   Verticle   │  │   Verticle   │  │   Verticle    ││
│  └──────────────┘  └──────────────┘  └───────────────┘│
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│                    Data Access Layer                     │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐│
│  │  Database    │  │    Redis     │  │   Message     ││
│  │  Connection  │  │    Cache     │  │     Queue     ││
│  │     Pool     │  │              │  │   (Kafka)     ││
│  └──────────────┘  └──────────────┘  └───────────────┘│
└─────────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Configuration

Edit `config.yaml` or use environment variables:

```bash
# Server configuration
export PORT=8080
export MAX_CCU=5000
export UTILIZATION_PERCENT=67

# Database configuration
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=fluxor
export DB_USER=fluxor
export DB_PASSWORD=password

# Authentication
export JWT_SECRET=your-secret-key

# Observability
export JAEGER_ENDPOINT=http://localhost:14268/api/traces
export PROMETHEUS_PORT=9090
```

### 2. Run the Application

```bash
# Build
go build -o enterprise-app

# Run with config file
./enterprise-app

# Or with environment variables
CONFIG_PATH=config.yaml ./enterprise-app
```

### 3. Run with Docker

```bash
# Build Docker image
docker build -t fluxor-enterprise .

# Run with docker-compose (includes Jaeger, Prometheus, PostgreSQL)
docker-compose up -d
```

## API Endpoints

### Public Endpoints

- `GET /` - Welcome message with feature list
- `GET /health` - Basic health check
- `GET /ready` - Readiness probe (checks dependencies)
- `GET /health/detailed` - Detailed health status with all checks
- `POST /api/auth/login` - Login and get JWT token
- `POST /api/auth/register` - Register new user

### Authenticated Endpoints (Requires JWT)

- `GET /api/users` - List all users
- `POST /api/users` - Create new user
- `GET /api/users/:id` - Get user by ID

### Admin Endpoints (Requires JWT + Admin Role)

- `GET /api/admin/metrics` - Server metrics (CCU, backpressure, etc.)
- `GET /api/admin/stats` - Database connection pool statistics

### Observability Endpoints

- `GET /metrics` - Prometheus metrics (exposed on port 9090)

## Example Usage

### 1. Get Authentication Token

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password"}'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 86400,
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 2. Call Authenticated Endpoint

```bash
curl -X GET http://localhost:8080/api/users \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 3. Check Health Status

```bash
curl http://localhost:8080/health/detailed
```

Response:
```json
{
  "status": {"healthy": true},
  "checks": {
    "database": {
      "status": "healthy",
      "latency_ms": 5
    },
    "external_api": {
      "status": "healthy",
      "latency_ms": 120
    }
  }
}
```

### 4. View Metrics

```bash
# Server metrics
curl http://localhost:8080/api/admin/metrics \
  -H "Authorization: Bearer <admin-token>"

# Prometheus metrics
curl http://localhost:9090/metrics
```

## Middleware Chain

The application uses an Express-like middleware chain:

```go
Middleware Chain:
1. Recovery       → Catches panics and returns 500
2. Logging        → Logs all requests with structured fields
3. CORS           → Handles CORS preflight and headers
4. Security       → Adds security headers (HSTS, CSP, etc.)
5. Rate Limiting  → IP-based rate limiting
6. Compression    → Gzip/Brotli compression
7. OpenTelemetry  → Distributed tracing spans
8. JWT Auth       → Validates JWT tokens (protected routes only)
9. RBAC           → Role-based access control (admin routes only)
```

## Observability Stack

### Jaeger (Distributed Tracing)

Access Jaeger UI: http://localhost:16686

Features:
- Request tracing across services
- Span visualization
- Performance analysis
- Dependency graphs

### Prometheus (Metrics)

Access Prometheus: http://localhost:9090

Metrics exported:
- HTTP request duration
- Request count by status code
- Active connections
- Queue utilization
- CCU (Concurrent Connected Users)
- Database connection pool stats

### Grafana (Visualization)

Access Grafana: http://localhost:3000

Pre-configured dashboards:
- API performance overview
- Backpressure monitoring
- Database connection pool
- Error rates and latency

## Load Testing

### Using k6

```bash
k6 run load-test.js
```

### Expected Performance

With default configuration (67% utilization, 5000 max CCU):

- Normal capacity: 3,350 CCU
- Request throughput: ~100,000 RPS
- P95 latency: < 10ms
- P99 latency: < 50ms
- Backpressure activation: > 3,350 CCU

## Docker Compose Stack

```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
    depends_on:
      - postgres
      - jaeger
      - prometheus
    environment:
      - DB_HOST=postgres
      - JAEGER_ENDPOINT=http://jaeger:14268/api/traces

  postgres:
    image: postgres:15
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_DB=fluxor
      - POSTGRES_USER=fluxor
      - POSTGRES_PASSWORD=password

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # UI
      - "14268:14268"  # Collector

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    depends_on:
      - prometheus
```

## Production Checklist

- [ ] Change JWT secret to a strong random value
- [ ] Configure production database credentials
- [ ] Set up TLS/HTTPS certificates
- [ ] Configure CORS allowed origins
- [ ] Set appropriate rate limits
- [ ] Enable distributed tracing
- [ ] Set up log aggregation (ELK, Splunk, CloudWatch)
- [ ] Configure health check endpoints for K8s probes
- [ ] Set up monitoring alerts
- [ ] Configure graceful shutdown timeout
- [ ] Enable database connection pooling
- [ ] Set resource limits (CPU, memory)

## Security Best Practices

1. **Environment Variables**: Never commit secrets to Git
2. **JWT Secret**: Use at least 32 random bytes
3. **Database**: Use connection pooling with appropriate limits
4. **Rate Limiting**: Tune based on expected traffic
5. **CORS**: Restrict to known origins only
6. **Security Headers**: Enable all recommended headers
7. **TLS**: Always use HTTPS in production
8. **Input Validation**: Validate and sanitize all inputs
9. **Monitoring**: Set up alerts for suspicious activity
10. **Updates**: Keep dependencies up to date

## Troubleshooting

### High Queue Utilization

```bash
# Check metrics
curl http://localhost:8080/api/admin/metrics

# Increase workers or max CCU
export MAX_CCU=10000
export UTILIZATION_PERCENT=60
```

### Database Connection Issues

```bash
# Check database health
curl http://localhost:8080/health/detailed

# Check pool stats
curl http://localhost:8080/api/admin/stats
```

### Tracing Not Working

```bash
# Verify Jaeger is running
curl http://localhost:14268/api/traces

# Check application logs for tracer initialization
```

## License

MIT License - See LICENSE file for details
