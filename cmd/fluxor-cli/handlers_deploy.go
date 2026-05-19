// Copyright (c) 2024-2028 Fluxor Framework
// All rights reserved.
//
// This source code is proprietary and confidential.
// Unauthorized copying, modification, distribution, or use of this software,
// via any medium is strictly prohibited without the express written permission
// of Fluxor Framework.
//
// This code is provided as an example for demonstration purposes only.
// Redistribution or sharing of this source code is not permitted.
//
// License: Proprietary - All Rights Reserved
// For licensing inquiries, please contact: caokhang91@gmail.com

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/fluxorio/fluxor/pkg/config"
	"github.com/fluxorio/fluxor/pkg/devops"
)

// handleDeploy handles the "deploy" command
func handleDeploy() {
	if p, ok := deployStartsWithPluginPath(os.Args[2:]); ok {
		handleRunPluginSO(p)
		return
	}

	deployCmd := flag.NewFlagSet("deploy", flag.ExitOnError)
	target := deployCmd.String("target", "", "Target name from YAML config (required)")
	configPath := deployCmd.String("config", "", "Path to deploy.yaml file (auto-discovered if not specified)")
	envFile := deployCmd.String("env-file", "", "Path to .env file (defaults to .env.local)")
	goApp := deployCmd.Bool("go-app", false, "Deploy Go application service only")
	nodeApp := deployCmd.Bool("node-app", false, "Deploy Node.js application HTML files only")
	nginx := deployCmd.Bool("nginx", false, "Deploy nginx configuration only")
	dockerCompose := deployCmd.Bool("docker-compose", false, "Deploy Docker Compose application only")
	certbot := deployCmd.Bool("certbot", false, "Run certbot on VPS to obtain/renew SSL (uses target certbot.domains and email)")
	force := deployCmd.Bool("force", false, "Skip confirmation prompts")

	deployCmd.Parse(os.Args[2:])

	if *target == "" {
		fmt.Fprintf(os.Stderr, "Error: -target is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli deploy -target <target> [-go-app|-node-app|-nginx|-docker-compose|-certbot] [-config <path>] [flags]\n")
		fmt.Fprintf(os.Stderr, "       fluxor-cli deploy <plugin.so>   Load verticle from Go plugin (linux/darwin; export NewVerticle)\n")
		os.Exit(1)
	}

	// Check that at least one service flag is specified
	if !*goApp && !*nodeApp && !*nginx && !*dockerCompose && !*certbot {
		fmt.Fprintf(os.Stderr, "Error: must specify at least one flag (-go-app, -node-app, -nginx, -docker-compose, or -certbot)\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli deploy -target <target> [-go-app|-node-app|-nginx|-docker-compose|-certbot] [-config <path>] [flags]\n")
		fmt.Fprintf(os.Stderr, "       fluxor-cli deploy <plugin.so>   Load verticle from Go plugin (linux/darwin; export NewVerticle)\n")
		os.Exit(1)
	}

	// Auto-discover deploy.yaml if not specified
	actualConfigPath := devops.FindDeployConfig(*configPath)
	if *configPath == "" {
		fmt.Printf("Using deploy config: %s\n", actualConfigPath)
	}

	// Load YAML config
	var cfg devops.DeployConfig
	if err := config.LoadYAML(actualConfigPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config from %s: %v\n", actualConfigPath, err)
		os.Exit(1)
	}

	// Get target config
	targetConfig, ok := cfg.Targets[*target]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: target '%s' not found in config\n", *target)
		os.Exit(1)
	}

	// Apply environment variable overrides from .env file
	if err := devops.ApplyEnvOverrides(&targetConfig, *target, *envFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load .env file: %v\n", err)
		// Continue anyway, use values from YAML
	}

	// Execute deploy Go app if requested
	if *goApp {
		opts := devops.DeployGoAppOptions{
			Force: *force,
		}
		if err := devops.DeployGoApp(targetConfig, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Execute deploy Node app if requested
	if *nodeApp {
		opts := devops.DeployNodeAppOptions{
			Force: *force,
		}
		if err := devops.DeployNodeApp(targetConfig, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Execute deploy nginx if requested
	if *nginx {
		opts := devops.DeployNginxOptions{
			Force: *force,
		}
		if err := devops.DeployNginx(targetConfig, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Execute deploy Docker Compose if requested
	if *dockerCompose {
		opts := devops.DeployDockerComposeOptions{
			Force: *force,
		}
		if err := devops.DeployDockerCompose(targetConfig, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Run certbot on VPS if requested (obtain/renew SSL for target certbot.domains)
	if *certbot {
		opts := devops.RunCertbotOptions{
			Force: *force,
		}
		if err := devops.RunCertbot(targetConfig, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// handleUndeploy handles the "undeploy" command
func handleUndeploy() {
	undeployCmd := flag.NewFlagSet("undeploy", flag.ExitOnError)
	target := undeployCmd.String("target", "", "Target name from YAML config (required)")
	configPath := undeployCmd.String("config", "", "Path to deploy.yaml file (auto-discovered if not specified)")
	envFile := undeployCmd.String("env-file", "", "Path to .env file (defaults to .env.local)")
	nginx := undeployCmd.Bool("nginx", false, "Undeploy nginx configuration only")
	goApp := undeployCmd.Bool("go-app", false, "Undeploy Go application service only")
	removeFiles := undeployCmd.Bool("remove-files", false, "Remove deployed application files")
	force := undeployCmd.Bool("force", false, "Skip confirmation prompts")

	undeployCmd.Parse(os.Args[2:])

	if *target == "" {
		fmt.Fprintf(os.Stderr, "Error: -target is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli undeploy -target <target> [-nginx|-go-app] [-config <path>] [flags]\n")
		os.Exit(1)
	}

	// Check that at least one service flag is specified
	if !*nginx && !*goApp {
		fmt.Fprintf(os.Stderr, "Error: must specify at least one service flag (-nginx or -go-app)\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli undeploy -target <target> [-nginx|-go-app] [-config <path>] [flags]\n")
		os.Exit(1)
	}

	// Auto-discover deploy.yaml if not specified
	actualConfigPath := devops.FindDeployConfig(*configPath)
	if *configPath == "" {
		fmt.Printf("Using deploy config: %s\n", actualConfigPath)
	}

	// Load YAML config
	var cfg devops.DeployConfig
	if err := config.LoadYAML(actualConfigPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config from %s: %v\n", actualConfigPath, err)
		os.Exit(1)
	}

	// Get target config
	targetConfig, ok := cfg.Targets[*target]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: target '%s' not found in config\n", *target)
		os.Exit(1)
	}

	// Apply environment variable overrides from .env file
	if err := devops.ApplyEnvOverrides(&targetConfig, *target, *envFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load .env file: %v\n", err)
		// Continue anyway, use values from YAML
	}

	// Execute undeploy nginx if requested
	if *nginx {
		opts := devops.UndeployNginxOptions{
			Force: *force,
		}
		if err := devops.UndeployNginx(targetConfig, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Execute undeploy Go app if requested
	if *goApp {
		opts := devops.UndeployGoAppOptions{
			RemoveFiles: *removeFiles,
			Force:       *force,
		}
		if err := devops.UndeployGoApp(targetConfig, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// handleRestartService handles the "restart -target" command for remote services
func handleRestartService() {
	restartCmd := flag.NewFlagSet("restart", flag.ExitOnError)
	target := restartCmd.String("target", "", "Target name from YAML config (required)")
	configPath := restartCmd.String("config", "", "Path to deploy.yaml file (auto-discovered if not specified)")
	envFile := restartCmd.String("env-file", "", "Path to .env file (defaults to .env.local)")
	goApp := restartCmd.Bool("go-app", false, "Restart Go application service only")
	nginx := restartCmd.Bool("nginx", false, "Restart nginx service only")
	force := restartCmd.Bool("force", false, "Skip confirmation prompts")

	restartCmd.Parse(os.Args[2:])

	if *target == "" {
		fmt.Fprintf(os.Stderr, "Error: -target is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli restart -target <target> [-go-app|-nginx] [-config <path>] [flags]\n")
		os.Exit(1)
	}

	// Check that at least one service flag is specified
	if !*goApp && !*nginx {
		fmt.Fprintf(os.Stderr, "Error: must specify at least one service flag (-go-app or -nginx)\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli restart -target <target> [-go-app|-nginx] [-config <path>] [flags]\n")
		os.Exit(1)
	}

	// Auto-discover deploy.yaml if not specified
	actualConfigPath := devops.FindDeployConfig(*configPath)
	if *configPath == "" {
		fmt.Printf("Using deploy config: %s\n", actualConfigPath)
	}

	// Load YAML config
	var cfg devops.DeployConfig
	if err := config.LoadYAML(actualConfigPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config from %s: %v\n", actualConfigPath, err)
		os.Exit(1)
	}

	// Get target config
	targetConfig, ok := cfg.Targets[*target]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: target '%s' not found in config\n", *target)
		os.Exit(1)
	}

	// Apply environment variable overrides from .env file
	if err := devops.ApplyEnvOverrides(&targetConfig, *target, *envFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load .env file: %v\n", err)
		// Continue anyway, use values from YAML
	}

	// Execute restart Go app if requested
	if *goApp {
		opts := devops.RestartGoAppOptions{
			Force: *force,
		}
		if err := devops.RestartGoApp(targetConfig, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Execute restart nginx if requested
	if *nginx {
		opts := devops.RestartNginxOptions{
			Force: *force,
		}
		if err := devops.RestartNginx(targetConfig, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// handleState handles the "state" command
func handleState() {
	stateCmd := flag.NewFlagSet("state", flag.ExitOnError)
	target := stateCmd.String("target", "", "Target name from YAML config (required)")
	configPath := stateCmd.String("config", "", "Path to deploy.yaml file (auto-discovered if not specified)")
	envFile := stateCmd.String("env-file", "", "Path to .env file (defaults to .env.local)")
	jsonOutput := stateCmd.Bool("json", false, "Output as JSON")

	stateCmd.Parse(os.Args[2:])

	if *target == "" {
		fmt.Fprintf(os.Stderr, "Error: -target is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli state -target <target> [-json] [-config <path>]\n")
		os.Exit(1)
	}

	// Auto-discover deploy.yaml if not specified
	actualConfigPath := devops.FindDeployConfig(*configPath)
	if *configPath == "" {
		fmt.Printf("Using deploy config: %s\n", actualConfigPath)
	}

	// Load YAML config
	var cfg devops.DeployConfig
	if err := config.LoadYAML(actualConfigPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config from %s: %v\n", actualConfigPath, err)
		os.Exit(1)
	}

	// Get target config
	targetConfig, ok := cfg.Targets[*target]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: target '%s' not found in config\n", *target)
		os.Exit(1)
	}

	// Apply environment variable overrides
	if err := devops.ApplyEnvOverrides(&targetConfig, *target, *envFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load .env file: %v\n", err)
	}

	// Query state
	fmt.Printf("Querying state for target: %s (%s)...\n", *target, targetConfig.Host)
	state, err := devops.QueryState(targetConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to query state: %v\n", err)
		os.Exit(1)
	}

	// Save state
	stateManager, err := devops.NewStateManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create state manager: %v\n", err)
	} else {
		if err := stateManager.SaveState(*target, state); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
		}
	}

	// Output state
	if *jsonOutput {
		// JSON output
		jsonData, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to marshal state: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
	} else {
		// Human-readable output
		fmt.Printf("\n📊 Application State: %s\n", *target)
		fmt.Printf("Host: %s\n", state.Host)
		fmt.Printf("Last Updated: %s\n", state.LastUpdated.Format("2006-01-02 15:04:05"))
		
		fmt.Printf("\n🔧 Services:\n")
		for name, service := range state.Services {
			fmt.Printf("  %s (%s): %s\n", name, service.Type, service.Status)
			if service.PID > 0 {
				fmt.Printf("    PID: %d\n", service.PID)
			}
			if service.Error != "" {
				fmt.Printf("    Error: %s\n", service.Error)
			}
		}

		if state.Resources.DiskUsage.Total > 0 {
			fmt.Printf("\n💾 Resources:\n")
			fmt.Printf("  Disk: %.1f%% used (%.1f GB / %.1f GB)\n",
				state.Resources.DiskUsage.Percent,
				float64(state.Resources.DiskUsage.Used)/(1024*1024*1024),
				float64(state.Resources.DiskUsage.Total)/(1024*1024*1024))
			fmt.Printf("  Memory: %.1f%% used (%.1f GB / %.1f GB)\n",
				state.Resources.MemoryUsage.Percent,
				float64(state.Resources.MemoryUsage.Used)/(1024*1024*1024),
				float64(state.Resources.MemoryUsage.Total)/(1024*1024*1024))
		}

		if state.Health.Status != "" {
			fmt.Printf("\n🏥 Health: %s\n", state.Health.Status)
			for _, check := range state.Health.Checks {
				fmt.Printf("  %s: %s\n", check.Name, check.Status)
			}
		}
	}
}

// handleListServices lists all systemd services on the target VPS (systemctl list-units --type=service).
func handleListServices() {
	listCmd := flag.NewFlagSet("list-services", flag.ExitOnError)
	target := listCmd.String("target", "", "Target name from YAML config (required)")
	configPath := listCmd.String("config", "", "Path to deploy.yaml file (auto-discovered if not specified)")
	envFile := listCmd.String("env-file", "", "Path to .env file (defaults to .env.local)")

	listCmd.Parse(os.Args[2:])

	if *target == "" {
		fmt.Fprintf(os.Stderr, "Error: -target is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli list-services -target <target> [-config <path>]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli list-services -target production\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli list-services -target quadgate-io | grep nginx\n\n")
		os.Exit(1)
	}

	actualConfigPath := devops.FindDeployConfig(*configPath)
	if *configPath == "" {
		fmt.Printf("Using deploy config: %s\n", actualConfigPath)
	}

	var cfg devops.DeployConfig
	if err := config.LoadYAML(actualConfigPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config from %s: %v\n", actualConfigPath, err)
		os.Exit(1)
	}

	targetConfig, ok := cfg.Targets[*target]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: target '%s' not found in config\n", *target)
		os.Exit(1)
	}

	if err := devops.ApplyEnvOverrides(&targetConfig, *target, *envFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load .env file: %v\n", err)
	}

	// Only print progress when stdout is a terminal (not piped to grep, etc.)
	if fi, err := os.Stdout.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		fmt.Fprintf(os.Stderr, "Listing systemd services on %s (%s)... (tip: pipe to grep to filter)\n", *target, targetConfig.Host)
	}
	out, err := devops.ListSystemdServices(targetConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(out)
}

// handleServiceLogs fetches journalctl logs for a systemd service on the target VPS.
func handleServiceLogs() {
	logsCmd := flag.NewFlagSet("service-logs", flag.ExitOnError)
	target := logsCmd.String("target", "", "Target name from YAML config (required)")
	configPath := logsCmd.String("config", "", "Path to deploy.yaml file (auto-discovered if not specified)")
	envFile := logsCmd.String("env-file", "", "Path to .env file (defaults to .env.local)")
	service := logsCmd.String("service", "", "Systemd service name (default: target's go_app.service_name)")
	lines := logsCmd.Int("lines", 80, "Number of log lines to show")

	logsCmd.Parse(os.Args[2:])

	if *target == "" {
		fmt.Fprintf(os.Stderr, "Error: -target is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fluxor-cli service-logs -target <target> [-service <name>] [-lines N]\n")
		os.Exit(1)
	}

	actualConfigPath := devops.FindDeployConfig(*configPath)
	if *configPath == "" {
		fmt.Printf("Using deploy config: %s\n", actualConfigPath)
	}

	var cfg devops.DeployConfig
	if err := config.LoadYAML(actualConfigPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config from %s: %v\n", actualConfigPath, err)
		os.Exit(1)
	}

	targetConfig, ok := cfg.Targets[*target]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: target '%s' not found in config\n", *target)
		os.Exit(1)
	}

	if err := devops.ApplyEnvOverrides(&targetConfig, *target, *envFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load .env file: %v\n", err)
	}

	serviceName := *service
	if serviceName == "" && targetConfig.GoApp.ServiceName != "" {
		serviceName = targetConfig.GoApp.ServiceName
	}
	if serviceName == "" {
		fmt.Fprintf(os.Stderr, "Error: -service is required (target has no go_app.service_name)\n")
		os.Exit(1)
	}

	opts := devops.ServiceLogsOptions{Lines: *lines}
	out, err := devops.ServiceLogs(targetConfig, serviceName, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(out)
}

// hasTargetFlag checks if -target flag exists in args
func hasTargetFlag(args []string) bool {
	for i, arg := range args {
		if arg == "-target" && i+1 < len(args) {
			return true
		}
	}
	return false
}
