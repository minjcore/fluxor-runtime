// Package shell provides local process execution (terminal/command interface) for the Fluxor CLI
// (e.g. "fluxor run app.py", "fluxor ps", "fluxor logs").
//
// Shell = running commands in a terminal or via a process interface; cmd is the executable + args.
// It is the sibling of pkg/devops (deploy to VPS): devops = deploy, shell = run locally.
//
//   - fluxor run app.py / main.go  → ProcessRuntime (this package)
//   - fluxor deploy app.yaml       → DockerRuntime (devops)
//
// Multi-runtime vision: Fluxor Runtime (verticle) | Docker Runtime | Local Process Runtime (shell).
package shell
