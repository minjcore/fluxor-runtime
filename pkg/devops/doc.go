// Package devops provides deployment and operations for Fluxor (VPS, Docker Compose, SSH, state).
//
// Multi-runtime vision: this package implements the deploy/remote side. The sibling package
// pkg/shell implements local process run (terminal/command interface: fluxor run app.py, fluxor ps, fluxor logs).
// Together they support a uv/PM2-style UX:
//
//   - fluxor run app.py / main.go  → pkg/shell.ProcessRuntime (local processes)
//   - fluxor deploy app.yaml      → DockerRuntime (this package; Docker Compose on VPS)
//
// Future runtimes (Kubernetes, Docker Swarm, local Docker) can be added as additional
// implementations of the same deploy/run concepts.
package devops
