package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
)

// ValidationConfig configures OpenAPI validation middleware
type ValidationConfig struct {
	// Spec is the OpenAPI specification to validate against
	Spec *Spec

	// ValidateRequest validates incoming requests
	ValidateRequest bool

	// ValidateResponse validates outgoing responses
	ValidateResponse bool

	// OnError is called when validation fails
	OnError func(ctx *web.FastRequestContext, err *ValidationError) error

	// SkipPaths is a list of paths to skip validation
	SkipPaths []string
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error in %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// Validate creates middleware that validates requests/responses against OpenAPI spec
func Validate(config ValidationConfig) web.FastMiddleware {
	if config.Spec == nil {
		panic("OpenAPI validation: Spec is required")
	}

	// Default error handler
	onError := config.OnError
	if onError == nil {
		onError = func(ctx *web.FastRequestContext, err *ValidationError) error {
			ctx.RequestCtx.SetStatusCode(http.StatusBadRequest)
			ctx.RequestCtx.SetContentType("application/json")
			errorResponse := map[string]interface{}{
				"error":   "validation_error",
				"message": err.Error(),
			}
			if err.Field != "" {
				errorResponse["field"] = err.Field
			}
			jsonData, _ := json.Marshal(errorResponse)
			ctx.RequestCtx.Write(jsonData)
			return nil
		}
	}

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// Check if path should be skipped
			path := string(ctx.Path())
			for _, skipPath := range config.SkipPaths {
				if path == skipPath || strings.HasPrefix(path, skipPath) {
					return next(ctx)
				}
			}

			// Find matching path and operation
			method := string(ctx.Method())
			pathItem, operation := findOperation(config.Spec, method, path)
			if pathItem == nil || operation == nil {
				// No spec found for this route, skip validation
				return next(ctx)
			}

			// Validate request if enabled
			if config.ValidateRequest {
				if err := validateRequest(ctx, pathItem, operation); err != nil {
					return onError(ctx, err)
				}
			}

			// Execute handler
			err := next(ctx)

			// Validate response if enabled
			if err == nil && config.ValidateResponse {
				if respErr := validateResponse(ctx, operation); respErr != nil {
					// Log response validation error but don't fail the request
					// In production, you might want to log this
					_ = respErr
				}
			}

			return err
		}
	}
}

// findOperation finds the matching path item and operation for a request
func findOperation(spec *Spec, method, path string) (*PathItem, *Operation) {
	// Normalize path (remove query string)
	path = strings.Split(path, "?")[0]

	// Try to find exact match first
	if pathItem, ok := spec.Paths[path]; ok {
		return getOperation(&pathItem, method)
	}

	// Try pattern matching (e.g., /users/:id -> /users/{id})
	for specPath, pathItem := range spec.Paths {
		if matchesPath(specPath, path) {
			return getOperation(&pathItem, method)
		}
	}

	return nil, nil
}

// matchesPath checks if a request path matches a spec path pattern
func matchesPath(pattern, path string) bool {
	// Convert pattern like /users/{id} to regex
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i, patternPart := range patternParts {
		pathPart := pathParts[i]
		if strings.HasPrefix(patternPart, "{") && strings.HasSuffix(patternPart, "}") {
			// Parameter placeholder, matches anything
			continue
		}
		if patternPart != pathPart {
			return false
		}
	}

	return true
}

// getOperation gets the operation for a method from a path item
func getOperation(pathItem *PathItem, method string) (*PathItem, *Operation) {
	method = strings.ToUpper(method)
	switch method {
	case "GET":
		return pathItem, pathItem.GET
	case "POST":
		return pathItem, pathItem.POST
	case "PUT":
		return pathItem, pathItem.PUT
	case "DELETE":
		return pathItem, pathItem.DELETE
	case "PATCH":
		return pathItem, pathItem.PATCH
	case "HEAD":
		return pathItem, pathItem.HEAD
	case "OPTIONS":
		return pathItem, pathItem.OPTIONS
	case "TRACE":
		return pathItem, pathItem.TRACE
	default:
		return pathItem, nil
	}
}

// validateRequest validates a request against the OpenAPI spec
func validateRequest(ctx *web.FastRequestContext, pathItem *PathItem, operation *Operation) *ValidationError {
	// Validate path parameters
	if err := validatePathParameters(ctx, pathItem, operation); err != nil {
		return err
	}

	// Validate query parameters
	if err := validateQueryParameters(ctx, operation); err != nil {
		return err
	}

	// Validate headers
	if err := validateHeaders(ctx, operation); err != nil {
		return err
	}

	// Validate request body
	if err := validateRequestBody(ctx, operation); err != nil {
		return err
	}

	return nil
}

// validatePathParameters validates path parameters
func validatePathParameters(ctx *web.FastRequestContext, pathItem *PathItem, operation *Operation) *ValidationError {
	// Combine path item and operation parameters
	params := make(map[string]Parameter)
	for _, param := range pathItem.Parameters {
		if param.In == "path" {
			params[param.Name] = param
		}
	}
	for _, param := range operation.Parameters {
		if param.In == "path" {
			params[param.Name] = param
		}
	}

	// Validate each path parameter
	for name, param := range params {
		value, ok := ctx.Params[name]
		if !ok {
			// Try alternative names (e.g., :id vs {id})
			value, ok = ctx.Params[strings.TrimPrefix(name, ":")]
		}

		if !ok && param.Required {
			return &ValidationError{
				Field:   fmt.Sprintf("path.%s", name),
				Message: "required path parameter is missing",
			}
		}

		if ok && param.Schema != nil {
			if err := validateValue(value, param.Schema, fmt.Sprintf("path.%s", name)); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateQueryParameters validates query parameters
func validateQueryParameters(ctx *web.FastRequestContext, operation *Operation) *ValidationError {
	for _, param := range operation.Parameters {
		if param.In != "query" {
			continue
		}

		value := ctx.Query(param.Name)
		if value == "" {
			if param.Required {
				return &ValidationError{
					Field:   fmt.Sprintf("query.%s", param.Name),
					Message: "required query parameter is missing",
				}
			}
			continue
		}

		if param.Schema != nil {
			if err := validateValue(value, param.Schema, fmt.Sprintf("query.%s", param.Name)); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateHeaders validates headers
func validateHeaders(ctx *web.FastRequestContext, operation *Operation) *ValidationError {
	for _, param := range operation.Parameters {
		if param.In != "header" {
			continue
		}

		value := string(ctx.RequestCtx.Request.Header.Peek(param.Name))
		if value == "" {
			if param.Required {
				return &ValidationError{
					Field:   fmt.Sprintf("header.%s", param.Name),
					Message: "required header is missing",
				}
			}
			continue
		}

		if param.Schema != nil {
			if err := validateValue(value, param.Schema, fmt.Sprintf("header.%s", param.Name)); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateRequestBody validates the request body
func validateRequestBody(ctx *web.FastRequestContext, operation *Operation) *ValidationError {
	if operation.RequestBody == nil {
		return nil
	}

	body := ctx.RequestCtx.PostBody()
	if len(body) == 0 {
		if operation.RequestBody.Required {
			return &ValidationError{
				Field:   "body",
				Message: "request body is required",
			}
		}
		return nil
	}

	// Get content type
	contentType := string(ctx.RequestCtx.Request.Header.ContentType())
	if contentType == "" {
		contentType = "application/json"
	}

	// Find matching media type
	var mediaType *MediaType
	for ct, mt := range operation.RequestBody.Content {
		if strings.Contains(contentType, ct) {
			mediaType = &mt
			break
		}
	}

	if mediaType == nil {
		return &ValidationError{
			Field:   "body",
			Message: fmt.Sprintf("unsupported content type: %s", contentType),
		}
	}

	// Validate JSON body
	if mediaType.Schema != nil && strings.Contains(contentType, "json") {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			return &ValidationError{
				Field:   "body",
				Message: fmt.Sprintf("invalid JSON: %v", err),
			}
		}

		if err := validateValue(jsonData, mediaType.Schema, "body"); err != nil {
			return err
		}
	}

	return nil
}

// validateResponse validates the response against the spec
func validateResponse(ctx *web.FastRequestContext, operation *Operation) *ValidationError {
	statusCode := strconv.Itoa(ctx.RequestCtx.Response.StatusCode())
	response, ok := operation.Responses[statusCode]
	if !ok {
		// Try default response
		response, ok = operation.Responses["default"]
		if !ok {
			// No spec for this status code, skip validation
			return nil
		}
	}

	// Get response body
	body := ctx.RequestCtx.Response.Body()
	if len(body) == 0 {
		return nil
	}

	// Get content type
	contentType := string(ctx.RequestCtx.Response.Header.ContentType())
	if contentType == "" {
		contentType = "application/json"
	}

	// Find matching media type
	var mediaType *MediaType
	for ct, mt := range response.Content {
		if strings.Contains(contentType, ct) {
			mediaType = &mt
			break
		}
	}

	if mediaType == nil {
		// No content type specified in spec, skip validation
		return nil
	}

	// Validate JSON response
	if mediaType.Schema != nil && strings.Contains(contentType, "json") {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			return &ValidationError{
				Field:   "response.body",
				Message: fmt.Sprintf("invalid JSON: %v", err),
			}
		}

		if err := validateValue(jsonData, mediaType.Schema, "response.body"); err != nil {
			return err
		}
	}

	return nil
}

// validateValue validates a value against a schema
func validateValue(value interface{}, schema *Schema, fieldPath string) *ValidationError {
	if schema == nil {
		return nil
	}

	// Handle $ref
	if schema.Ref != "" {
		// For now, skip $ref validation (would need to resolve references)
		return nil
	}

	// Validate type
	if schema.Type != "" {
		if err := validateType(value, schema.Type, fieldPath); err != nil {
			return err
		}
	}

	// Validate string constraints
	if schema.Type == "string" {
		if str, ok := value.(string); ok {
			if schema.MinLength != nil && len(str) < *schema.MinLength {
				return &ValidationError{
					Field:   fieldPath,
					Message: fmt.Sprintf("string length must be at least %d", *schema.MinLength),
					Value:   value,
				}
			}
			if schema.MaxLength != nil && len(str) > *schema.MaxLength {
				return &ValidationError{
					Field:   fieldPath,
					Message: fmt.Sprintf("string length must be at most %d", *schema.MaxLength),
					Value:   value,
				}
			}
			if schema.Pattern != "" {
				matched, _ := regexp.MatchString(schema.Pattern, str)
				if !matched {
					return &ValidationError{
						Field:   fieldPath,
						Message: fmt.Sprintf("string does not match pattern: %s", schema.Pattern),
						Value:   value,
					}
				}
			}
		}
	}

	// Validate number constraints
	if schema.Type == "number" || schema.Type == "integer" {
		var num float64
		switch v := value.(type) {
		case float64:
			num = v
		case int:
			num = float64(v)
		case int64:
			num = float64(v)
		case string:
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return &ValidationError{
					Field:   fieldPath,
					Message: "value is not a valid number",
					Value:   value,
				}
			}
			num = parsed
		default:
			return &ValidationError{
				Field:   fieldPath,
				Message: fmt.Sprintf("expected %s, got %T", schema.Type, value),
				Value:   value,
			}
		}

		if schema.Minimum != nil && num < *schema.Minimum {
			return &ValidationError{
				Field:   fieldPath,
				Message: fmt.Sprintf("value must be at least %f", *schema.Minimum),
				Value:   value,
			}
		}
		if schema.Maximum != nil && num > *schema.Maximum {
			return &ValidationError{
				Field:   fieldPath,
				Message: fmt.Sprintf("value must be at most %f", *schema.Maximum),
				Value:   value,
			}
		}
	}

	// Validate enum
	if len(schema.Enum) > 0 {
		found := false
		for _, enumVal := range schema.Enum {
			if value == enumVal {
				found = true
				break
			}
		}
		if !found {
			return &ValidationError{
				Field:   fieldPath,
				Message: fmt.Sprintf("value must be one of: %v", schema.Enum),
				Value:   value,
			}
		}
	}

	// Validate object properties
	if schema.Type == "object" {
		if obj, ok := value.(map[string]interface{}); ok {
			// Check required properties
			for _, required := range schema.Required {
				if _, exists := obj[required]; !exists {
					return &ValidationError{
						Field:   fmt.Sprintf("%s.%s", fieldPath, required),
						Message: "required property is missing",
					}
				}
			}

			// Validate properties
			if schema.Properties != nil {
				for propName, propSchema := range schema.Properties {
					if propValue, exists := obj[propName]; exists {
						if err := validateValue(propValue, propSchema, fmt.Sprintf("%s.%s", fieldPath, propName)); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// Validate array items
	if schema.Type == "array" {
		if arr, ok := value.([]interface{}); ok {
			if schema.MinItems != nil && len(arr) < *schema.MinItems {
				return &ValidationError{
					Field:   fieldPath,
					Message: fmt.Sprintf("array must have at least %d items", *schema.MinItems),
					Value:   value,
				}
			}
			if schema.MaxItems != nil && len(arr) > *schema.MaxItems {
				return &ValidationError{
					Field:   fieldPath,
					Message: fmt.Sprintf("array must have at most %d items", *schema.MaxItems),
					Value:   value,
				}
			}
			if schema.Items != nil {
				for i, item := range arr {
					if err := validateValue(item, schema.Items, fmt.Sprintf("%s[%d]", fieldPath, i)); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// validateType validates that a value matches the expected type
func validateType(value interface{}, expectedType, fieldPath string) *ValidationError {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return &ValidationError{
				Field:   fieldPath,
				Message: "expected string",
				Value:   value,
			}
		}
	case "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return nil
		case float64:
			// JSON numbers are float64, check if it's actually an integer
			if float64(int(value.(float64))) == value.(float64) {
				return nil
			}
		case string:
			// Try to parse as integer
			if _, err := strconv.ParseInt(value.(string), 10, 64); err == nil {
				return nil
			}
		}
		return &ValidationError{
			Field:   fieldPath,
			Message: "expected integer",
			Value:   value,
		}
	case "number":
		switch value.(type) {
		case float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return nil
		case string:
			if _, err := strconv.ParseFloat(value.(string), 64); err == nil {
				return nil
			}
		}
		return &ValidationError{
			Field:   fieldPath,
			Message: "expected number",
			Value:   value,
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return &ValidationError{
				Field:   fieldPath,
				Message: "expected boolean",
				Value:   value,
			}
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return &ValidationError{
				Field:   fieldPath,
				Message: "expected array",
				Value:   value,
			}
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return &ValidationError{
				Field:   fieldPath,
				Message: "expected object",
				Value:   value,
			}
		}
	}

	return nil
}
