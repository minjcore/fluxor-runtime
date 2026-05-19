# Managers Signal Example

This example demonstrates the Managers (Execution Control Unit) event signaling system with HTTP server lifecycle events.

## What It Demonstrates

- **Managers as Control Plane**: Managers coordinates and manages components (control plane pattern)
- **Event Signaling**: HTTP server signals Managers when it starts/stops
- **Event Handlers**: Register handlers to observe component lifecycle events
- **Component Registration**: Register components with Managers for coordination

## How It Works

1. **Create GoCMD and Managers**: Sets up the runtime (GoCMD) and control plane (Managers)
2. **Register Event Handlers**: Registers handlers for HTTP server start/stop events
3. **Create HTTP Server**: Uses `Managers.CreateHTTPServer()` which wraps the server to send signals
4. **Start Server**: When `httpServer.Start()` is called, the wrapper signals Managers
5. **Event Triggers**: Managers emits events, triggering registered handlers

## Key Concepts

- **Managers = Control Plane**: Managers coordinates but doesn't start/stop components (runtime does that)
- **Signal Pattern**: Components signal Managers when lifecycle events occur
- **Event Observers**: Register handlers to observe and react to component events
- **Component Wrapper**: `CreateHTTPServer()` wraps the server to automatically signal Managers

## Running the Example

```bash
go run ./cmd/managers-signal
```

The server will start on `http://localhost:8080`. You'll see:

1. Event handlers registered
2. HTTP server created via Managers
3. Server started (triggers start event handler)
4. Event handlers called with component name and event type

## Testing

```bash
# In another terminal, test the endpoints
curl http://localhost:8080/
curl http://localhost:8080/status
```

Press Ctrl+C to stop. The stop event handler will be triggered.

## Output Example

```
Created GoCMD instance
Created Managers instance (control plane)
Registered event handlers with Managers
Created HTTP server via Managers (wrapped for signaling)
Configured HTTP routes
Registered HTTP server with Managers
Wired components via Managers
Starting HTTP server...
→ This will trigger Managers event handlers when server starts
🚀 Managers Event Handler: Component 'http-server' started
   → HTTP server has started successfully!
   → Server is now listening on :8080
📡 General Event Handler: Component 'http-server' event: started

✅ Server is running. Event handlers were triggered!
   Send HTTP requests to http://localhost:8080/
   Press Ctrl+C to stop...
```

