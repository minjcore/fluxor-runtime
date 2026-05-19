# Serve static site locally
fluxor-cli up staticsite              # Serve current dir on :8080
fluxor-cli up staticsite -dir=dist -port=3000

# Deploy everything
fluxor-cli deploy -target production -go-app -node-app -nginx -force

# Or deploy individually
fluxor-cli deploy -target production -go-app -force    # Go server
fluxor-cli deploy -target production -node-app -force  # HTML + assets
fluxor-cli deploy -target production -nginx -force     # Nginx config
# Fluxor CLI

Command-line tool for creating and managing Fluxor applications (PM2-like process manager).

## Installation

Build from source:
```bash
go build -o bin/fluxor-cli ./cmd/fluxor-cli
```

Or install globally:
```bash
go install ./cmd/fluxor-cli
```

## Usage

### Create a new application

```bash
fluxor-cli new myapp
```

This will create a new Fluxor application with:
- `main.go` - Application entry point with API verticle
- `config.json` - Configuration file
- `go.mod` - Go module file
- `README.md` - Documentation

### Process Management (PM2-like)

Fluxor CLI includes PM2-like process management capabilities:

#### Start an application

Start an application in the background (daemon mode):

```bash
# Start from current directory
fluxor-cli start myapp

# Start from specific directory
fluxor-cli start myapp /path/to/app
```

The application will run in the background and logs will be saved to `~/.fluxor-cli/myapp.log`.

#### List all applications

```bash
fluxor-cli list
# or
fluxor-cli ls
```

Shows all managed applications with their status, PID, and working directory.

#### Stop an application

```bash
fluxor-cli stop myapp
```

Stops a running application gracefully.

#### Restart an application

```bash
fluxor-cli restart myapp
```

Restarts an application (stops and starts again).

#### View logs

```bash
# Show all logs
fluxor-cli logs myapp

# Show last 100 lines
fluxor-cli logs myapp --lines 100
```

#### Check status

```bash
fluxor-cli status myapp
```

Shows detailed information about an application including:
- Status (online/stopped/error)
- PID
- Working directory
- Command
- Restart count
- Log file locations

#### Delete an application

```bash
fluxor-cli delete myapp
# or
fluxor-cli del myapp
# or
fluxor-cli rm myapp
```

Removes an application from the process list (stops it first if running).

### Show version

```bash
fluxor-cli version
```

## Complete Workflow Example

1. Create a new application:
   ```bash
   fluxor-cli new myapp
   cd myapp
   go mod tidy
   ```

2. Start the application:
   ```bash
   fluxor-cli start myapp .
   ```

3. Check status:
   ```bash
   fluxor-cli status myapp
   ```

4. View logs:
   ```bash
   fluxor-cli logs myapp
   ```

5. List all applications:
   ```bash
   fluxor-cli list
   ```

6. Stop the application:
   ```bash
   fluxor-cli stop myapp
   ```

## Data Storage

Process information is stored in `~/.fluxor-cli/`:
- `processes.json` - Process metadata
- `*.log` - Application logs
- `*.error.log` - Error logs

## Features

- ✅ Background process management (daemon mode)
- ✅ Process monitoring and auto-cleanup
- ✅ Log file management
- ✅ Process status tracking
- ✅ Restart functionality
- ✅ Multiple application management

