# HelloWork - Simple Verticle Example

A minimal Fluxor binary that demonstrates how to start and stop a verticle.

## Overview

This example shows the basic lifecycle of a Fluxor verticle:
1. Create a GoCMD instance (Fluxor Stream)
2. Deploy a verticle
3. Wait for interrupt signal (Ctrl+C)
4. Undeploy the verticle and shutdown

## Building

```bash
go build -o hellowork.exe ./cmd/hellowork
```

Or from the project root:

```bash
go build -o bin/hellowork ./cmd/hellowork
```

## Running

```bash
./hellowork.exe
```

The program will:
- Deploy a `HelloVerticle` that listens on the event bus
- Send a test message to the verticle
- Wait for Ctrl+C to stop
- Gracefully undeploy the verticle and shutdown

## Example Output

```
Deployed verticle with ID: deployment.xxxxx-xxxxx-xxxxx
Received reply: Hello from HelloVerticle!
HelloVerticle is running. Press Ctrl+C to stop...
^C
Shutting down...
Verticle undeployed successfully
```

## Code Structure

- `HelloVerticle`: A simple verticle that implements `Start()` and `Stop()` methods
- Uses `BaseVerticle` for lifecycle management
- Registers an event bus consumer in `Start()`
- Handles graceful shutdown on interrupt signal

