#!/bin/bash
set -e

echo "🚀 Starting ProxyCache..."

# Build the application
echo "📦 Building proxycache..."
go build -o proxycache .

# Run with default settings
echo "🎯 Starting proxycache on port 8080..."
./proxycache -port 8080 -cache ./proxycache -v
