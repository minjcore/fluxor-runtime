#!/bin/bash

# Example: Using ProxyCache for Go Module Caching
# This script demonstrates how to use ProxyCache

set -e

echo "📦 ProxyCache Demo - Go Module Caching"
echo "======================================="
echo ""

# Build the proxy cache
echo "1️⃣  Building ProxyCache..."
cd "$(dirname "$0")"
go build -o proxycache .

# Start the proxy in background
echo "2️⃣  Starting ProxyCache on port 8080..."
./proxycache -port 8080 -cache ./demo-cache -v &
PROXY_PID=$!

# Wait for server to start
sleep 2

# Check health
echo "3️⃣  Checking health..."
curl -s http://localhost:8080/_health

echo ""
echo "✅ ProxyCache is running!"
echo ""
echo "To use it, run:"
echo "  export GOPROXY=http://localhost:8080,direct"
echo ""
echo "Press Ctrl+C to stop the proxy"

# Wait for interrupt
trap "kill $PROXY_PID 2>/dev/null || true; exit 0" INT TERM
wait $PROXY_PID
