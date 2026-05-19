// Package openapi provides OpenAPI 3.0 specification generation and serving.
//
// This package allows you to:
//   - Generate OpenAPI 3.0 specifications from your routes
//   - Serve OpenAPI specifications as JSON
//   - Serve Swagger UI for interactive API documentation
//   - Validate requests and responses against OpenAPI schemas
//
// Example usage:
//
//	// Create an OpenAPI builder
//	builder := openapi.NewBuilder(openapi.Info{
//		Title:       "My API",
//		Description: "My awesome API",
//		Version:     "1.0.0",
//	})
//
//	// Add routes
//	builder.AddRoute(openapi.RouteInfo{
//		Method:      "GET",
//		Path:        "/users/:id",
//		Summary:     "Get user by ID",
//		Description: "Retrieves a user by their unique identifier",
//		OperationID: "getUser",
//		Tags:        []string{"users"},
//		Responses: map[string]openapi.Response{
//			"200": openapi.JSONResponse("User found", userSchema),
//			"404": openapi.ErrorResponse("User not found"),
//		},
//	})
//
//	// Build and register
//	spec := builder.Build()
//	openapi.RegisterOpenAPIRoutes(router, spec)
//
//	// Add validation middleware
//	router.UseFast(openapi.Validate(openapi.ValidationConfig{
//		Spec:            spec,
//		ValidateRequest: true,
//	}))
//
//	// Access Swagger UI at /docs
//	// Access OpenAPI spec at /openapi.json
package openapi
