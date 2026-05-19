package openapi

import (
	"fmt"
	"strings"
)

// RouteInfo contains information about a route for OpenAPI generation
type RouteInfo struct {
	Method      string
	Path        string
	Summary     string
	Description string
	OperationID string
	Tags        []string
	Parameters  []Parameter
	RequestBody *RequestBody
	Responses   map[string]Response
	Security    []SecurityRequirement
	Deprecated  bool
}

// Builder helps build OpenAPI specifications from route information
type Builder struct {
	spec    *Spec
	routes  []RouteInfo
}

// NewBuilder creates a new OpenAPI builder
func NewBuilder(info Info) *Builder {
	return &Builder{
		spec:   NewSpec(info),
		routes: make([]RouteInfo, 0),
	}
}

// AddRoute adds a route to the builder
func (b *Builder) AddRoute(route RouteInfo) {
	b.routes = append(b.routes, route)
}

// Build generates the OpenAPI specification from added routes
func (b *Builder) Build() *Spec {
	// Group routes by path
	pathMap := make(map[string]*PathItem)

	for _, route := range b.routes {
		pathItem, exists := pathMap[route.Path]
		if !exists {
			pathItem = &PathItem{}
			pathMap[route.Path] = pathItem
		}

		operation := &Operation{
			Summary:     route.Summary,
			Description: route.Description,
			OperationID: route.OperationID,
			Tags:        route.Tags,
			Parameters:  route.Parameters,
			RequestBody: route.RequestBody,
			Responses:   route.Responses,
			Security:    route.Security,
			Deprecated:  route.Deprecated,
		}

		// Set operation on path item based on method
		method := strings.ToUpper(route.Method)
		switch method {
		case "GET":
			pathItem.GET = operation
		case "POST":
			pathItem.POST = operation
		case "PUT":
			pathItem.PUT = operation
		case "DELETE":
			pathItem.DELETE = operation
		case "PATCH":
			pathItem.PATCH = operation
		case "HEAD":
			pathItem.HEAD = operation
		case "OPTIONS":
			pathItem.OPTIONS = operation
		case "TRACE":
			pathItem.TRACE = operation
		}

		// Extract path parameters from path pattern
		pathParams := extractPathParameters(route.Path)
		if len(pathParams) > 0 {
			// Add path parameters if not already present
			for _, paramName := range pathParams {
				found := false
				for _, existingParam := range pathItem.Parameters {
					if existingParam.Name == paramName && existingParam.In == "path" {
						found = true
						break
					}
				}
				if !found {
					pathItem.Parameters = append(pathItem.Parameters, Parameter{
						Name:        paramName,
						In:          "path",
						Required:    true,
						Description: fmt.Sprintf("Path parameter: %s", paramName),
						Schema: &Schema{
							Type: "string",
						},
					})
				}
			}
		}
	}

	// Add paths to spec
	for path, item := range pathMap {
		b.spec.AddPath(path, *item)
	}

	return b.spec
}

// extractPathParameters extracts parameter names from path patterns like "/users/:id"
func extractPathParameters(path string) []string {
	parts := strings.Split(path, "/")
	params := make([]string, 0)

	for _, part := range parts {
		if len(part) > 0 && (part[0] == ':' || part[0] == '{') {
			// Remove : or { }
			param := strings.TrimPrefix(part, ":")
			param = strings.TrimPrefix(param, "{")
			param = strings.TrimSuffix(param, "}")
			if param != "" {
				params = append(params, param)
			}
		}
	}

	return params
}

// Helper functions for creating common schemas

// StringSchema creates a string schema
func StringSchema() *Schema {
	return &Schema{Type: "string"}
}

// IntegerSchema creates an integer schema
func IntegerSchema() *Schema {
	return &Schema{Type: "integer", Format: "int64"}
}

// NumberSchema creates a number schema
func NumberSchema() *Schema {
	return &Schema{Type: "number"}
}

// BooleanSchema creates a boolean schema
func BooleanSchema() *Schema {
	return &Schema{Type: "boolean"}
}

// ArraySchema creates an array schema
func ArraySchema(items *Schema) *Schema {
	return &Schema{
		Type:  "array",
		Items: items,
	}
}

// ObjectSchema creates an object schema
func ObjectSchema(properties map[string]*Schema, required []string) *Schema {
	return &Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

// RefSchema creates a reference schema
func RefSchema(ref string) *Schema {
	return &Schema{Ref: ref}
}

// JSONResponse creates a JSON response
func JSONResponse(description string, schema *Schema) Response {
	return Response{
		Description: description,
		Content: map[string]MediaType{
			"application/json": {
				Schema: schema,
			},
		},
	}
}

// JSONRequestBody creates a JSON request body
func JSONRequestBody(schema *Schema, required bool) *RequestBody {
	return &RequestBody{
		Description: "",
		Content: map[string]MediaType{
			"application/json": {
				Schema: schema,
			},
		},
		Required: required,
	}
}

// ErrorResponse creates a standard error response
func ErrorResponse(description string) Response {
	return JSONResponse(description, ObjectSchema(
		map[string]*Schema{
			"error":   StringSchema(),
			"message": StringSchema(),
		},
		[]string{"error", "message"},
	))
}

// BearerAuthScheme creates a Bearer token security scheme
func BearerAuthScheme(description string) SecurityScheme {
	return SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  description,
	}
}

// APIKeyAuthScheme creates an API key security scheme
func APIKeyAuthScheme(name, in, description string) SecurityScheme {
	return SecurityScheme{
		Type:        "apiKey",
		Name:        name,
		In:          in, // header, query, cookie
		Description: description,
	}
}
