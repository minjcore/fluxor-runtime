// Copyright (c) 2024-2028 Fluxor Framework
// All rights reserved.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fluxorio/fluxor/pkg/entrypoint"
)

const usage = `fluxorctl — manage Fluxor verticles (plugin or subprocess).

Standalone (starts a new process):
  fluxorctl deploy <plugin.{so,dylib}>
  fluxorctl run    <plugin.{so,dylib}>

Plugin hot-deploy into a running process:
  fluxorctl deploy   --target <socket> <plugin.{so,dylib}>
  fluxorctl undeploy --target <socket> <deployment-id>

Subprocess hot-deploy into a running process:
  fluxorctl spawn  --target <socket> <binary>
  fluxorctl kill   --target <socket> <subprocess-id>

List all deployments (plugins + subprocesses):
  fluxorctl list --target <socket>

Plugin build: go build -buildmode=plugin -o myverticle.so <package>
Subprocess build: go build -o myverticle <package>  (binary calls entrypoint.RunSubprocess)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	// Parse --target <socket> flag from remaining args.
	target, rest := extractFlag(args, "--target")
	socket, rest := extractFlag(rest, "--socket")

	switch cmd {
	case "deploy", "run":
		if target != "" {
			// Hot-deploy into running process.
			if len(rest) == 0 {
				fatalf("deploy --target: missing plugin path\n")
			}
			path := rest[0]
			if !isPlugin(path) {
				fatalf("expected .so or .dylib plugin path, got %q\n", path)
			}
			resp, err := entrypoint.AdminDial(target, entrypoint.AdminRequest{
				Cmd:  "deploy",
				Path: path,
			})
			if err != nil {
				fatalf("dial %s: %v\n", target, err)
			}
			printResp(resp)
		} else {
			// Standalone: start a new process.
			if len(rest) == 0 {
				fatalf("deploy: missing plugin path\n")
			}
			path := rest[0]
			if !isPlugin(path) {
				fatalf("expected .so or .dylib plugin path, got %q\n", path)
			}
			if err := entrypoint.RunFromPluginSO(path, socket); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}

	case "undeploy":
		if target == "" {
			fatalf("undeploy requires --target <socket>\n")
		}
		if len(rest) == 0 {
			fatalf("undeploy: missing deployment ID\n")
		}
		resp, err := entrypoint.AdminDial(target, entrypoint.AdminRequest{
			Cmd: "undeploy",
			ID:  rest[0],
		})
		if err != nil {
			fatalf("dial %s: %v\n", target, err)
		}
		printResp(resp)

	case "spawn":
		if target == "" {
			fatalf("spawn requires --target <socket>\n")
		}
		if len(rest) == 0 {
			fatalf("spawn: missing binary path\n")
		}
		resp, err := entrypoint.AdminDial(target, entrypoint.AdminRequest{
			Cmd:  "spawn",
			Path: rest[0],
		})
		if err != nil {
			fatalf("dial %s: %v\n", target, err)
		}
		printResp(resp)

	case "kill":
		if target == "" {
			fatalf("kill requires --target <socket>\n")
		}
		if len(rest) == 0 {
			fatalf("kill: missing subprocess ID\n")
		}
		resp, err := entrypoint.AdminDial(target, entrypoint.AdminRequest{
			Cmd: "kill",
			ID:  rest[0],
		})
		if err != nil {
			fatalf("dial %s: %v\n", target, err)
		}
		printResp(resp)

	case "list":
		if target == "" {
			fatalf("list requires --target <socket>\n")
		}
		resp, err := entrypoint.AdminDial(target, entrypoint.AdminRequest{Cmd: "list"})
		if err != nil {
			fatalf("dial %s: %v\n", target, err)
		}
		if !resp.OK {
			fatalf("list failed: %s\n", resp.Err)
		}
		if len(resp.IDs) == 0 {
			fmt.Println("(no deployments)")
		} else {
			for _, id := range resp.IDs {
				fmt.Println(id)
			}
		}

	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}

// extractFlag finds --flag <value> in args and returns (value, remaining args).
func extractFlag(args []string, flag string) (string, []string) {
	var value string
	var rest []string
	for i := 0; i < len(args); i++ {
		if args[i] == flag && i+1 < len(args) {
			value = args[i+1]
			i++
		} else {
			rest = append(rest, args[i])
		}
	}
	return value, rest
}

func isPlugin(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".so") || strings.HasSuffix(lower, ".dylib")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format, args...)
	os.Exit(1)
}

func printResp(resp *entrypoint.AdminResponse) {
	if resp.OK {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Err)
		os.Exit(1)
	}
}
