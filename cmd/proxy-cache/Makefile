.PHONY: build run test clean install-govulncheck govulncheck scan

# Build the proxy cache binary
build:
	@echo "📦 Building proxycache..."
	@go build -o proxycache .
	@echo "✅ Build complete!"

# Run the proxy cache
run: build
	@echo "🚀 Starting proxycache on port 8080..."
	@./proxycache -port 8080 -cache ./proxycache -v

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -f proxycache
	@rm -rf ./proxycache
	@echo "✅ Clean complete!"

# Get the path to Go binaries
GOBIN := $(shell go env GOBIN 2>/dev/null)
ifeq ($(GOBIN),)
	GOBIN := $(shell go env GOPATH)/bin
endif
GOVULNCHECK := $(GOBIN)/govulncheck

# Install govulncheck if not already installed
install-govulncheck:
	@echo "🔧 Installing govulncheck..."
	@if [ -f "$(GOVULNCHECK)" ] || command -v govulncheck > /dev/null; then \
		echo "✅ govulncheck is already installed"; \
		"$(GOVULNCHECK)" -version 2>/dev/null || govulncheck -version; \
	else \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		echo "✅ govulncheck installed successfully"; \
	fi

# Scan for vulnerabilities using govulncheck
govulncheck: install-govulncheck
	@echo "🔍 Scanning for vulnerabilities with govulncheck..."
	@if [ -f "$(GOVULNCHECK)" ]; then \
		"$(GOVULNCHECK)" ./...; \
	elif command -v govulncheck > /dev/null; then \
		govulncheck ./...; \
	else \
		echo "❌ Error: govulncheck not found. Please ensure GOPATH/bin is in your PATH."; \
		exit 1; \
	fi
	@echo "✅ Vulnerability scan complete!"

# Alias for govulncheck
scan: govulncheck

.PHONY: cron-install cron-remove cron-list cron-info

# Get the absolute path to the scan script
SCAN_SCRIPT := $(shell cd "$(shell dirname "$(abspath $(lastword $(MAKEFILE_LIST)))")" && pwd)/scan-vulns.sh

# Install cron job for scheduled vulnerability scanning (daily at 2 AM)
cron-install:
	@echo "📅 Installing cron job for scheduled vulnerability scanning..."
	@if [ ! -f "$(SCAN_SCRIPT)" ]; then \
		echo "❌ Error: scan-vulns.sh not found at $(SCAN_SCRIPT)"; \
		exit 1; \
	fi
	@(crontab -l 2>/dev/null | grep -v "$(SCAN_SCRIPT)" || true; \
		echo "0 2 * * * $(SCAN_SCRIPT) > /dev/null 2>&1") | crontab -
	@echo "✅ Cron job installed successfully!"
	@echo "   Schedule: Daily at 2:00 AM"
	@echo "   Script: $(SCAN_SCRIPT)"
	@echo "   Scan scope: Root project (entire repository)"
	@echo ""
	@echo "To view logs, check: $(dir $(SCAN_SCRIPT))logs/"

# Remove cron job for vulnerability scanning
cron-remove:
	@echo "🗑️  Removing cron job for vulnerability scanning..."
	@crontab -l 2>/dev/null | grep -v "$(SCAN_SCRIPT)" | crontab - 2>/dev/null || true
	@echo "✅ Cron job removed successfully!"

# List current cron jobs (filter for this project)
cron-list:
	@echo "📋 Current cron jobs for vulnerability scanning:"
	@crontab -l 2>/dev/null | grep "$(SCAN_SCRIPT)" || echo "   No cron jobs found for this project"

# Show information about cron setup
cron-info:
	@echo "📅 Vulnerability Scanning Cron Job Information"
	@echo "==============================================="
	@echo "Script location: $(SCAN_SCRIPT)"
	@echo "Log directory: $(dir $(SCAN_SCRIPT))logs/"
	@echo "Scan scope: Root project (scans entire repository from go.mod root)"
	@echo ""
	@echo "Current cron jobs:"
	@crontab -l 2>/dev/null | grep "$(SCAN_SCRIPT)" || echo "   None installed"
	@echo ""
	@echo "To install daily scan at 2 AM: make cron-install"
	@echo "To remove: make cron-remove"
	@echo "To manually run: ./scan-vulns.sh"
