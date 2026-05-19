#!/bin/bash
# Vulnerability scanning script for cron jobs
# This script runs govulncheck and logs the results

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Find the root project directory (where go.mod is located)
# Start from script directory and walk up until we find go.mod
ROOT_DIR="$SCRIPT_DIR"
while [ "$ROOT_DIR" != "/" ] && [ ! -f "$ROOT_DIR/go.mod" ]; do
    ROOT_DIR="$(dirname "$ROOT_DIR")"
done

if [ ! -f "$ROOT_DIR/go.mod" ]; then
    echo "Error: Could not find go.mod file. Please run from the project root or a subdirectory."
    exit 1
fi

# Change to root project directory for scanning
cd "$ROOT_DIR"

# Set up environment for Go binaries
export PATH="${PATH}:$(go env GOPATH)/bin"
export GOBIN="$(go env GOBIN 2>/dev/null || go env GOPATH)/bin"

# Create logs directory if it doesn't exist
LOG_DIR="${SCRIPT_DIR}/logs"
mkdir -p "$LOG_DIR"

# Log file with timestamp
LOG_FILE="${LOG_DIR}/vulnscan-$(date +%Y%m%d-%H%M%S).log"
ERROR_LOG="${LOG_DIR}/vulnscan-errors.log"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" | tee -a "$LOG_FILE" >> "$ERROR_LOG"
}

# Start scanning
log "================================================"
log "Starting vulnerability scan with govulncheck"
log "Script location: $SCRIPT_DIR"
log "Root project directory: $ROOT_DIR"
log "Working directory: $(pwd)"
log "================================================"

# Check if govulncheck is installed
GOVULNCHECK="${GOBIN}/govulncheck"
if [ ! -f "$GOVULNCHECK" ] && ! command -v govulncheck > /dev/null; then
    log_error "govulncheck not found. Installing..."
    go install golang.org/x/vuln/cmd/govulncheck@latest || {
        log_error "Failed to install govulncheck"
        exit 1
    }
    log "govulncheck installed successfully"
fi

# Run the scan on the entire root project
log "Running govulncheck on root project ($ROOT_DIR)..."
if [ -f "$GOVULNCHECK" ]; then
    if "$GOVULNCHECK" ./... >> "$LOG_FILE" 2>&1; then
        log "✅ Vulnerability scan completed successfully"
        log "No vulnerabilities found (or scan completed without errors)"
    else
        EXIT_CODE=$?
        log_error "Vulnerability scan found issues (exit code: $EXIT_CODE)"
        log_error "Check $LOG_FILE for details"
        exit $EXIT_CODE
    fi
elif command -v govulncheck > /dev/null; then
    if govulncheck ./... >> "$LOG_FILE" 2>&1; then
        log "✅ Vulnerability scan completed successfully"
        log "No vulnerabilities found (or scan completed without errors)"
    else
        EXIT_CODE=$?
        log_error "Vulnerability scan found issues (exit code: $EXIT_CODE)"
        log_error "Check $LOG_FILE for details"
        exit $EXIT_CODE
    fi
else
    log_error "govulncheck not found in PATH or GOBIN"
    exit 1
fi

log "Scan results saved to: $LOG_FILE"
log "================================================"

# Keep only the last 30 log files
find "$LOG_DIR" -name "vulnscan-*.log" -type f -mtime +30 -delete 2>/dev/null || true

exit 0
