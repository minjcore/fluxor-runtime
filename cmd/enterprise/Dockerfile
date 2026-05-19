# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o enterprise ./cmd/enterprise/main.go

# Final stage
FROM alpine:latest

# OCI labels for GitHub Container Registry
# Update org.opencontainers.image.source with your actual repository URL
LABEL org.opencontainers.image.source="https://github.com/fluxorio/fluxor-homework"
LABEL org.opencontainers.image.description="Fluxor Enterprise - production-ready event-driven runtime with observability"
LABEL org.opencontainers.image.licenses="Proprietary"

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/enterprise .

# Copy config file (optional)
COPY --from=builder /app/cmd/enterprise/config.yaml ./config.yaml

# Expose ports
EXPOSE 8080
EXPOSE 9090

# Set environment variables
ENV PORT=8080 \
    ENVIRONMENT=production

# Run the application
CMD ["./enterprise"]
