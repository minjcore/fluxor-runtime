package openapi

import (
	"fmt"
	"net/http"

	"github.com/fluxorio/fluxor/pkg/web"
)

// ServeSpec serves the OpenAPI specification as JSON
func ServeSpec(spec *Spec) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		jsonData, err := spec.ToJSON()
		if err != nil {
			ctx.RequestCtx.SetStatusCode(http.StatusInternalServerError)
			return fmt.Errorf("failed to marshal OpenAPI spec: %w", err)
		}

		ctx.RequestCtx.SetContentType("application/json")
		ctx.RequestCtx.SetStatusCode(http.StatusOK)
		_, err = ctx.RequestCtx.Write(jsonData)
		return err
	}
}

// ServeSwaggerUI serves a Swagger UI HTML page
func ServeSwaggerUI(specURL, title string) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>%s API Documentation</title>
	<link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui.css" />
	<style>
		html {
			box-sizing: border-box;
			overflow: -moz-scrollbars-vertical;
			overflow-y: scroll;
		}
		*, *:before, *:after {
			box-sizing: inherit;
		}
		body {
			margin:0;
			background: #fafafa;
		}
	</style>
</head>
<body>
	<div id="swagger-ui"></div>
	<script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-bundle.js"></script>
	<script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-standalone-preset.js"></script>
	<script>
		window.onload = function() {
			const ui = SwaggerUIBundle({
				url: "%s",
				dom_id: '#swagger-ui',
				deepLinking: true,
				presets: [
					SwaggerUIBundle.presets.apis,
					SwaggerUIStandalonePreset
				],
				plugins: [
					SwaggerUIBundle.plugins.DownloadUrl
				],
				layout: "StandaloneLayout"
			});
		};
	</script>
</body>
</html>`, title, specURL)

		ctx.RequestCtx.SetContentType("text/html")
		ctx.RequestCtx.SetStatusCode(http.StatusOK)
		_, err := ctx.RequestCtx.WriteString(html)
		return err
	}
}

// RegisterOpenAPIRoutes registers OpenAPI spec and Swagger UI routes
func RegisterOpenAPIRoutes(router *web.FastRouter, spec *Spec, opts ...OpenAPIOption) {
	config := &OpenAPIConfig{
		SpecPath:      "/openapi.json",
		UIPath:        "/docs",
		UITitle:       "API Documentation",
		SpecURL:       "/openapi.json",
	}

	for _, opt := range opts {
		opt(config)
	}

	// Serve OpenAPI spec
	router.GETFast(config.SpecPath, ServeSpec(spec))

	registerSwaggerUI(router, config)
}

// RegisterOpenAPIRoutesBytes registers pre-generated Swagger/OpenAPI JSON (e.g. from swag) and Swagger UI.
func RegisterOpenAPIRoutesBytes(router *web.FastRouter, specJSON []byte, opts ...OpenAPIOption) {
	config := &OpenAPIConfig{
		SpecPath: "/openapi.json",
		UIPath:   "/docs",
		UITitle:  "API Documentation",
		SpecURL:  "/openapi.json",
	}

	for _, opt := range opts {
		opt(config)
	}

	router.GETFast(config.SpecPath, ServeSpecBytes(specJSON))
	registerSwaggerUI(router, config)
}

// ServeSpecBytes serves a literal JSON spec (Swagger 2.0 or OpenAPI 3.x).
func ServeSpecBytes(specJSON []byte) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetContentType("application/json")
		ctx.RequestCtx.SetStatusCode(http.StatusOK)
		_, err := ctx.RequestCtx.Write(specJSON)
		return err
	}
}

func registerSwaggerUI(router *web.FastRouter, config *OpenAPIConfig) {
	router.GETFast(config.UIPath, ServeSwaggerUI(config.SpecURL, config.UITitle))
}

// OpenAPIConfig configures OpenAPI route registration
type OpenAPIConfig struct {
	SpecPath string
	UIPath   string
	UITitle  string
	SpecURL  string
}

// OpenAPIOption configures OpenAPI routes
type OpenAPIOption func(*OpenAPIConfig)

// WithSpecPath sets the path for the OpenAPI spec endpoint
func WithSpecPath(path string) OpenAPIOption {
	return func(c *OpenAPIConfig) {
		c.SpecPath = path
		if c.SpecURL == "" {
			c.SpecURL = path
		}
	}
}

// WithUIPath sets the path for the Swagger UI
func WithUIPath(path string) OpenAPIOption {
	return func(c *OpenAPIConfig) {
		c.UIPath = path
	}
}

// WithUITitle sets the title for the Swagger UI
func WithUITitle(title string) OpenAPIOption {
	return func(c *OpenAPIConfig) {
		c.UITitle = title
	}
}

// WithSpecURL sets the URL to the spec used by Swagger UI
func WithSpecURL(url string) OpenAPIOption {
	return func(c *OpenAPIConfig) {
		c.SpecURL = url
	}
}
