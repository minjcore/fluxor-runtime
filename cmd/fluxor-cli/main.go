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
	"path/filepath"
	"text/template"
)

const version = "0.1.0"

func main() {
	newCmd := flag.NewFlagSet("new", flag.ExitOnError)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		// Check if this is "new workflow" command
		if len(os.Args) >= 3 && os.Args[2] == "workflow" {
			workflowCmd := flag.NewFlagSet("workflow", flag.ExitOnError)
			workflowCmd.Parse(os.Args[3:])
			if workflowCmd.NArg() == 0 {
				fmt.Fprintf(os.Stderr, "Error: workflow name is required\n\n")
				fmt.Fprintf(os.Stderr, "Usage: fluxor-cli new workflow <workflow-name>\n")
				os.Exit(1)
			}
			workflowName := workflowCmd.Arg(0)
			if err := createNewWorkflow(workflowName); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ Successfully created workflow '%s.json'\n", workflowName)
			fmt.Printf("\nYou can now:\n")
			fmt.Printf("  - Edit the workflow file: %s.json\n", workflowName)
			fmt.Printf("  - Register it with a WorkflowVerticle\n")
			fmt.Printf("  - Execute it via the workflow engine API\n\n")
		} else if len(os.Args) >= 3 && os.Args[2] == "service" {
			// New service command
			serviceCmd := flag.NewFlagSet("service", flag.ExitOnError)
			serviceCmd.Parse(os.Args[3:])
			if serviceCmd.NArg() == 0 {
				fmt.Fprintf(os.Stderr, "Error: service name is required\n\n")
				fmt.Fprintf(os.Stderr, "Usage: fluxor-cli new service <service-name>\n")
				os.Exit(1)
			}
			serviceName := serviceCmd.Arg(0)
			if err := createNewService(serviceName); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ Successfully created Fluxor service '%s'\n", serviceName)
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  cd %s\n", serviceName)
			fmt.Printf("  go mod tidy\n")
			fmt.Printf("  # Edit application.properties to configure your service\n")
			fmt.Printf("  go run .\n\n")
		} else {
			// Existing app creation logic
			newCmd.Parse(os.Args[2:])
			if newCmd.NArg() == 0 {
				fmt.Fprintf(os.Stderr, "Error: app name is required\n\n")
				fmt.Fprintf(os.Stderr, "Usage: fluxor-cli new <appname>\n")
				os.Exit(1)
			}
			appName := newCmd.Arg(0)
			if err := createNewApp(appName); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ Successfully created Fluxor application '%s'\n", appName)
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  cd %s\n", appName)
			fmt.Printf("  go mod tidy\n")
			fmt.Printf("  go run .\n\n")
		}
	case "run":
		handleRun()
	case "start":
		handleStart()
	case "stop":
		handleStop()
	case "restart":
		// Check if -target flag is present (VPS restart) or local restart
		if len(os.Args) >= 3 && hasTargetFlag(os.Args[2:]) {
			handleRestartService()
		} else {
			// Local restart (existing functionality)
			handleRestart()
		}
	case "list", "ls":
		handleList()
	case "logs":
		handleLogs()
	case "delete", "del", "rm":
		handleDelete()
	case "status":
		handleStatus()
	case "undeploy":
		handleUndeploy()
	case "deploy":
		handleDeploy()
	case "run-plugin":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: fluxor-cli run-plugin <path/to/plugin.so>\n")
			fmt.Fprintf(os.Stderr, "  Plugin must export func NewVerticle() core.Verticle (see pkg/entrypoint.PluginVerticleSymbol).\n\n")
			os.Exit(1)
		}
		handleRunPluginSO(os.Args[2])
	case "state", "status-remote":
		handleState()
	case "list-services", "services":
		handleListServices()
	case "service-logs":
		handleServiceLogs()
	case "up":
		if len(os.Args) >= 3 && os.Args[2] == "staticsite" {
			handleUpStaticsite()
		} else {
			fmt.Fprintf(os.Stderr, "Usage: fluxor-cli up staticsite [-dir=<dir>] [-port=<port>]\n")
			os.Exit(1)
		}
	case "version", "-v", "--version":
		fmt.Printf("fluxor-cli version %s\n", version)
	case "cpp-example":
		runCppExample()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Fluxor CLI - Create and manage Fluxor applications\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  fluxor-cli new <appname>           Create a new Fluxor application\n")
	fmt.Fprintf(os.Stderr, "  fluxor-cli new service <name>       Create a new service with Eureka integration\n")
	fmt.Fprintf(os.Stderr, "  fluxor-cli new workflow <name>     Create a new workflow JSON file\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli run <file>              Run a script in foreground (e.g. run main.go, run app.py)\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli run-plugin <plugin.so>  Load Fluxor verticle from Go plugin (linux/darwin)\n")
	fmt.Fprintf(os.Stderr, "  fluxor-cli start <name> [dir]      Start an application (PM2-like)\n")
	fmt.Fprintf(os.Stderr, "  fluxor-cli stop <name>             Stop a running application\n")
	fmt.Fprintf(os.Stderr, "  fluxor-cli restart <name>           Restart an application\n")
	fmt.Fprintf(os.Stderr, "  fluxor-cli list                    List all managed applications\n")
	fmt.Fprintf(os.Stderr, "  fluxor-cli logs <name> [--lines N] Show logs for an application\n")
	fmt.Fprintf(os.Stderr, "  fluxor-cli status <name>           Show status of a local app (PM2-like)\n")
	fmt.Fprintf(os.Stderr, "  fluxor-cli delete <name>           Delete an application from list\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli deploy -target <target> [-go-app|-node-app|-nginx|-docker-compose|-certbot] [flags]  Deploy applications\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli deploy <plugin.so>        Run verticle from Go plugin (or: run-plugin, run <file.so>)\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli restart -target <target> [-go-app|-nginx] [flags]  Restart services\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli undeploy -target <target> [-nginx|-go-app] [flags]  Undeploy services\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli state -target <target> [-json]  Query application state on remote server\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli list-services -target <target>  List all systemd services on VPS (pipe to grep to filter)\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli service-logs -target <target> [-service <name>] [-lines N]  Show journalctl logs for service\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli up staticsite [-dir=.] [-port=8080]  Serve static site locally\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli version                 Show version\n")
		fmt.Fprintf(os.Stderr, "  fluxor-cli cpp-example            Run Go calling C++ example\n\n")
}

func createNewApp(appName string) error {
	// Validate app name
	if appName == "" {
		return fmt.Errorf("app name cannot be empty")
	}

	// Check if directory already exists
	if _, err := os.Stat(appName); err == nil {
		return fmt.Errorf("directory '%s' already exists", appName)
	}

	// Create directory
	if err := os.MkdirAll(appName, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Prepare template data
	data := TemplateData{
		AppName: appName,
		Name:    appName,
	}

	// Load and execute templates
	loader := NewTemplateLoader(TemplateTypeApp)
	if err := loader.ExecuteAllTemplates(appName, data); err != nil {
		return fmt.Errorf("failed to execute templates: %w", err)
	}

	return nil
}

// WorkflowDefinition represents a workflow JSON structure
type WorkflowDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Nodes       []NodeDefinition       `json:"nodes"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
}

// NodeDefinition represents a node in the workflow
type NodeDefinition struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
	Next      []string               `json:"next,omitempty"`
	OnError   []string               `json:"onError,omitempty"`
	TrueNext  []string               `json:"trueNext,omitempty"`
	FalseNext []string               `json:"falseNext,omitempty"`
}

func createNewWorkflow(workflowName string) error {
	// Validate workflow name
	if workflowName == "" {
		return fmt.Errorf("workflow name cannot be empty")
	}

	// Create filename with .json extension
	filename := workflowName + ".json"

	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("file '%s' already exists", filename)
	}

	// Create workflow definition
	workflow := WorkflowDefinition{
		ID:   workflowName,
		Name: workflowName,
		Nodes: []NodeDefinition{
			{
				ID:   "start",
				Type: "manual",
				Next: []string{"process"},
			},
			{
				ID:   "process",
				Type: "noop",
				Next: []string{},
			},
		},
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	// Write to file
	// Use restrictive permissions (0600) to prevent unauthorized access to workflow files
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	return nil
}

func createMainGo(appName string) error {
	tmpl := `package main

import (
	"log"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
	"github.com/fluxorio/fluxor/pkg/web"
)

func main() {
	// Create MainVerticle with config
	app, err := entrypoint.NewMainVerticle("config.json")
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Deploy API verticle
	_, err = app.DeployVerticle(NewApiVerticle())
	if err != nil {
		log.Fatalf("Failed to deploy API verticle: %v", err)
	}

	// Start application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}

// ApiVerticle handles HTTP endpoints
type ApiVerticle struct {
	server *web.FastHTTPServer
}

func NewApiVerticle() *ApiVerticle {
	return &ApiVerticle{}
}

func (v *ApiVerticle) Start(ctx core.FluxorContext) error {
	log.Println("API Verticle started")

	// Get HTTP address from config
	addr := ":8080"
	if val, ok := ctx.Config()["http_addr"].(string); ok && val != "" {
		addr = val
	}

	// Create FastHTTPServer using context's GoCMD
	cfg := web.DefaultFastHTTPServerConfig(addr)
	v.server = web.NewFastHTTPServer(ctx.GoCMD(), cfg)

	// Setup routes
	router := v.server.FastRouter()

	// Health check endpoint
	router.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{
			"status":  "UP",
			"service": "{{.AppName}}",
		})
	})

	// Example API endpoint
	router.GETFast("/api/hello", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{
			"message": "Hello from Fluxor!",
		})
	})

	// Start server
	go func() {
		log.Printf("HTTP server starting on %s", addr)
		if err := v.server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Println("API endpoints registered:")
	log.Println("  GET /health - Health check")
	log.Println("  GET /api/hello - Hello endpoint")

	return nil
}

func (v *ApiVerticle) Stop(ctx core.FluxorContext) error {
	log.Println("API Verticle stopped")
	if v.server != nil {
		return v.server.Stop()
	}
	return nil
}
`

	t, err := template.New("main.go").Parse(tmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(appName, "main.go"))
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, map[string]string{
		"AppName": appName,
	})
}

func createConfigJson(appName string) error {
	content := `{
  "http_addr": ":8080"
}
`
	// Use restrictive permissions (0600) to prevent unauthorized access to config files
	return os.WriteFile(filepath.Join(appName, "config.json"), []byte(content), 0600)
}

func createGoMod(appName string) error {
	content := `module ` + appName + `

go 1.24.0

require github.com/fluxorio/fluxor v0.0.0
`
	// Use restrictive permissions (0600) to prevent unauthorized access
	return os.WriteFile(filepath.Join(appName, "go.mod"), []byte(content), 0600)
}

func createReadme(appName string) error {
	content := `# ` + appName + ` - Fluxor Application

A simple Fluxor application starter template.

## Getting Started

1. **Install dependencies:**
   ` + "```bash" + `
   go mod tidy
   ` + "```" + `

2. **Run the application:**
   ` + "```bash" + `
   go run .
   ` + "```" + `

3. **Test the endpoints:**
   ` + "```bash" + `
   curl http://localhost:8080/health
   curl http://localhost:8080/api/hello
   ` + "```" + `

## Project Structure

` + "```" + `
` + appName + `/
├── main.go          # Application entry point
├── config.json      # Configuration file
├── go.mod          # Go module file
└── README.md       # This file
` + "```" + `

## Next Steps

- Add more verticles in separate files (see ` + "`verticles/`" + ` directory pattern)
- Create event contracts in ` + "`contracts/`" + ` directory
- Add database connections using ` + "`pkg/dbruntime`" + `
- Add middleware for authentication, logging, etc.

## Resources

- [Fluxor Documentation](https://github.com/fluxorio/fluxor/blob/main/DOCUMENTATION.md)
- [Primary Pattern Guide](https://github.com/fluxorio/fluxor/blob/main/docs/PRIMARY_PATTERN.md)
- [Examples](https://github.com/fluxorio/fluxor/tree/main/examples)
`
	// Use restrictive permissions (0600) to prevent unauthorized access
	return os.WriteFile(filepath.Join(appName, "README.md"), []byte(content), 0600)
}

func createNewService(serviceName string) error {
	// Validate service name
	if serviceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Check if directory already exists
	if _, err := os.Stat(serviceName); err == nil {
		return fmt.Errorf("directory '%s' already exists", serviceName)
	}

	// Create directory
	if err := os.MkdirAll(serviceName, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Prepare template data
	data := TemplateData{
		ServiceName:     toPascalCase(serviceName),
		ServiceNameLower: serviceName,
		Name:            serviceName,
	}

	// Load and execute templates
	loader := NewTemplateLoader(TemplateTypeService)
	if err := loader.ExecuteAllTemplates(serviceName, data); err != nil {
		return fmt.Errorf("failed to execute templates: %w", err)
	}

	return nil
}

func createServiceMainGo(serviceName string) error {
	tmpl := `package main

import (
	"context"
	"log"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
)

func main() {
	// Load configuration from application.properties
	var cfg ServiceConfig
	if err := config.LoadProperties("application.properties", &cfg); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create MainVerticle with config
	app, err := entrypoint.NewMainVerticle("application.properties")
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Deploy service verticle
	_, err = app.DeployVerticle(New{{.ServiceName}}Verticle(&cfg))
	if err != nil {
		log.Fatalf("Failed to deploy service verticle: %v", err)
	}

	// Start application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	// Wait for interrupt
	select {}
}

// ServiceConfig represents the service configuration loaded from application.properties
type ServiceConfig struct {
	Server struct {
		Addr string ` + "`config:\"addr\"`" + `
		Port int    ` + "`config:\"port\"`" + `
	} ` + "`config:\"server\"`" + `

	Eureka struct {
		RegistryURL string ` + "`config:\"registry.url\"`" + `
		Enabled     bool   ` + "`config:\"enabled\"`" + `
	} ` + "`config:\"eureka\"`" + `

	Service struct {
		Name string ` + "`config:\"name\"`" + `
		Host string ` + "`config:\"host\"`" + `
		Port int    ` + "`config:\"port\"`" + `
	} ` + "`config:\"service\"`" + `
}
`

	t, err := template.New("main.go").Parse(tmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(serviceName, "main.go"))
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, map[string]string{
		"ServiceName": toPascalCase(serviceName),
	})
}

func createServiceVerticle(serviceName string) error {
	tmpl := `package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/eureka"
	"github.com/fluxorio/fluxor/pkg/web"
)

// {{.ServiceName}}Verticle is the main service verticle
type {{.ServiceName}}Verticle struct {
	*core.BaseVerticle
	config *ServiceConfig
	server *web.FastHTTPServer
	client *eureka.Client
}

// New{{.ServiceName}}Verticle creates a new service verticle
func New{{.ServiceName}}Verticle(config *ServiceConfig) *{{.ServiceName}}Verticle {
	return &{{.ServiceName}}Verticle{
		BaseVerticle: core.NewBaseVerticle("{{.ServiceNameLower}}"),
		config:       config,
	}
}

func (v *{{.ServiceName}}Verticle) Start(ctx core.FluxorContext) error {
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	log.Println("{{.ServiceName}} service starting...")

	// Get server address from config
	addr := fmt.Sprintf(":%d", v.config.Server.Port)
	if v.config.Server.Addr != "" {
		addr = v.config.Server.Addr
	}

	// Create FastHTTPServer
	cfg := web.DefaultFastHTTPServerConfig(addr)
	v.server = web.NewFastHTTPServer(ctx.GoCMD(), cfg)

	// Setup routes
	router := v.server.FastRouter()

	// Health check endpoint
	router.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{
			"status":  "UP",
			"service": v.config.Service.Name,
		})
	})

	// Example API endpoint
	router.GETFast("/api/hello", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{
			"message": fmt.Sprintf("Hello from %s!", v.config.Service.Name),
		})
	})

	// Register with Eureka if enabled
	if v.config.Eureka.Enabled {
		if err := v.registerWithEureka(ctx); err != nil {
			log.Printf("Warning: Failed to register with Eureka: %v", err)
		}
	}

	// Start server
	go func() {
		log.Printf("HTTP server starting on %s", addr)
		if err := v.server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Println("{{.ServiceName}} service started successfully")
	return nil
}

func (v *{{.ServiceName}}Verticle) Stop(ctx core.FluxorContext) error {
	log.Println("{{.ServiceName}} service stopping...")

	// Unregister from Eureka
	if v.client != nil {
		v.client.StopHeartbeat()
		unregisterCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = v.client.Unregister(unregisterCtx)
	}

	// Stop HTTP server
	if v.server != nil {
		if err := v.server.Stop(); err != nil {
			log.Printf("Error stopping server: %v", err)
		}
	}

	return v.BaseVerticle.Stop(ctx)
}

func (v *{{.ServiceName}}Verticle) registerWithEureka(ctx core.FluxorContext) error {
	// Create service instance
	instance := &eureka.ServiceInstance{
		ServiceName: v.config.Service.Name,
		Host:        v.config.Service.Host,
		Port:        v.config.Service.Port,
		Status:      eureka.InstanceStatusUp,
		Metadata: map[string]string{
			"version": "1.0.0",
		},
		HealthCheckURL: fmt.Sprintf("http://%s:%d/health", v.config.Service.Host, v.config.Service.Port),
	}

	// Create Eureka client
	clientConfig := eureka.DefaultClientConfig(v.config.Eureka.RegistryURL, instance)
	v.client = eureka.NewClient(clientConfig)

	// Register
	registerCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := v.client.Register(registerCtx); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	// Start heartbeat
	if err := v.client.StartHeartbeat(ctx.GoCMD().Context()); err != nil {
		return fmt.Errorf("failed to start heartbeat: %w", err)
	}

	log.Printf("Registered with Eureka: %s", v.config.Eureka.RegistryURL)
	return nil
}
`

	t, err := template.New("verticle.go").Parse(tmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(serviceName, "verticle.go"))
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, map[string]string{
		"ServiceName":     toPascalCase(serviceName),
		"ServiceNameLower": serviceName,
	})
}

func createApplicationProperties(serviceName string) error {
	content := `# Server Configuration
server.addr=:8080
server.port=8080

# Service Configuration
service.name={{.ServiceName}}
service.host=localhost
service.port=8080

# Eureka Service Registry Configuration
eureka.enabled=true
eureka.registry.url=http://localhost:8761
`
	t, err := template.New("application.properties").Parse(content)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(serviceName, "application.properties"))
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, map[string]string{
		"ServiceName": serviceName,
	})
}

func createServiceGoMod(serviceName string) error {
	content := `module ` + serviceName + `

go 1.24.0

require github.com/fluxorio/fluxor v0.0.0
`
	return os.WriteFile(filepath.Join(serviceName, "go.mod"), []byte(content), 0600)
}

func createServiceReadme(serviceName string) error {
	content := "# " + serviceName + " - Fluxor Service\n\n" +
		"A Fluxor service with Eureka service registry integration.\n\n" +
		"## Configuration\n\n" +
		"Edit `application.properties` to configure your service:\n\n" +
		"```properties\n" +
		"# Server Configuration\n" +
		"server.addr=:8080\n" +
		"server.port=8080\n\n" +
		"# Service Configuration\n" +
		"service.name=" + serviceName + "\n" +
		"service.host=localhost\n" +
		"service.port=8080\n\n" +
		"# Eureka Service Registry Configuration\n" +
		"eureka.enabled=true\n" +
		"eureka.registry.url=http://localhost:8761\n" +
		"```\n\n" +
		"## Getting Started\n\n" +
		"1. **Install dependencies:**\n" +
		"   ```bash\n" +
		"   go mod tidy\n" +
		"   ```\n\n" +
		"2. **Start Eureka registry server** (if using Eureka):\n" +
		"   ```bash\n" +
		"   # In another terminal, start the Eureka server\n" +
		"   # See pkg/eureka/README.md for instructions\n" +
		"   ```\n\n" +
		"3. **Run the service:**\n" +
		"   ```bash\n" +
		"   go run .\n" +
		"   ```\n\n" +
		"4. **Test the endpoints:**\n" +
		"   ```bash\n" +
		"   curl http://localhost:8080/health\n" +
		"   curl http://localhost:8080/api/hello\n" +
		"   ```\n\n" +
		"## Project Structure\n\n" +
		"```\n" +
		serviceName + "/\n" +
		"├── main.go                    # Application entry point\n" +
		"├── verticle.go                # Service verticle implementation\n" +
		"├── application.properties     # Configuration file\n" +
		"├── go.mod                     # Go module file\n" +
		"└── README.md                  # This file\n" +
		"```\n\n" +
		"## Features\n\n" +
		"- ✅ **Properties-based Configuration**: Uses `application.properties` for configuration\n" +
		"- ✅ **Eureka Integration**: Automatic service registration and discovery\n" +
		"- ✅ **Health Checks**: Built-in health check endpoint\n" +
		"- ✅ **HTTP Server**: FastHTTP server with routing\n" +
		"- ✅ **Graceful Shutdown**: Proper cleanup on shutdown\n\n" +
		"## Eureka Service Registry\n\n" +
		"This service automatically registers with Eureka when `eureka.enabled=true`.\n\n" +
		"### Service Discovery\n\n" +
		"To discover other services, use the Eureka client:\n\n" +
		"```go\n" +
		"import \"github.com/fluxorio/fluxor/pkg/eureka\"\n\n" +
		"client := eureka.NewClient(eureka.DefaultClientConfig(\"http://localhost:8761\", instance))\n" +
		"instances, err := client.Discover(ctx, \"other-service\")\n" +
		"```\n\n" +
		"## Next Steps\n\n" +
		"- Add more API endpoints in `verticle.go`\n" +
		"- Configure database connections\n" +
		"- Add authentication/authorization middleware\n" +
		"- Add logging and metrics\n" +
		"- Create additional verticles for different concerns\n\n" +
		"## Resources\n\n" +
		"- [Fluxor Documentation](https://github.com/fluxorio/fluxor)\n" +
		"- [Eureka Service Registry](../pkg/eureka/README.md)\n" +
		"- [Configuration Guide](../pkg/config/README.md)\n"
	return os.WriteFile(filepath.Join(serviceName, "README.md"), []byte(content), 0600)
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	// Simple conversion: capitalize first letter and handle common patterns
	result := ""
	capitalize := true
	for _, r := range s {
		if r == '-' || r == '_' || r == ' ' {
			capitalize = true
			continue
		}
		if capitalize {
			if r >= 'a' && r <= 'z' {
				result += string(r - 32) // Convert to uppercase
			} else {
				result += string(r)
			}
			capitalize = false
		} else {
			result += string(r)
		}
	}
	if len(result) > 0 && result[0] >= 'a' && result[0] <= 'z' {
		result = string(result[0]-32) + result[1:]
	}
	return result
}

// Process management handlers are in handlers_process.go
// Deploy handlers are in handlers_deploy.go
