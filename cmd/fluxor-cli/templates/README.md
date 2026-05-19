# Fluxor CLI Templates

This directory contains scaffold templates for different project types.

## Template Structure

```
templates/
├── service/          # Service scaffold with Eureka integration
│   ├── main.go.tmpl
│   ├── verticle.go.tmpl
│   ├── application.properties.tmpl
│   ├── go.mod.tmpl
│   └── README.md.tmpl
├── app/              # Basic application scaffold
│   ├── main.go.tmpl
│   ├── config.json.tmpl
│   ├── go.mod.tmpl
│   └── README.md.tmpl
└── workflow/          # Workflow JSON template (future)
```

## Adding New Templates

1. Create a new directory under `templates/` (e.g., `templates/myscaffold/`)
2. Add template files with `.tmpl` extension
3. Use Go template syntax with variables like `{{.Name}}`
4. Update `templates.go` to register the new scaffold type
5. Add CLI command handler in `main.go`

## Template Variables

Common template variables:
- `{{.ServiceName}}` - Service name (PascalCase)
- `{{.ServiceNameLower}}` - Service name (lowercase)
- `{{.AppName}}` - Application name

## Usage

Templates are loaded and executed by the CLI when creating new projects:

```bash
fluxor-cli new service my-service    # Uses templates/service/
fluxor-cli new myapp                 # Uses templates/app/
```
