# Multi-stage build for smaller image
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o proxycache \
    ./cmd/proxycache

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 proxycache && \
    adduser -D -u 1000 -G proxycache proxycache

# Create cache directory
RUN mkdir -p /cache && chown -R proxycache:proxycache /cache

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/proxycache .

# Switch to non-root user
USER proxycache

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/_health || exit 1

# Default command
CMD ["./proxycache", "-cache", "/cache", "-port", "8080"]
