package openapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestValidationError_Error(t *testing.T) {
	t.Run("with field", func(t *testing.T) {
		err := &ValidationError{
			Field:   "path.id",
			Message: "required path parameter is missing",
		}
		msg := err.Error()
		if !strings.Contains(msg, "path.id") {
			t.Errorf("ValidationError.Error() = %q, should contain 'path.id'", msg)
		}
		if !strings.Contains(msg, "required path parameter is missing") {
			t.Errorf("ValidationError.Error() = %q, should contain message", msg)
		}
	})

	t.Run("without field", func(t *testing.T) {
		err := &ValidationError{
			Message: "validation error occurred",
		}
		msg := err.Error()
		if !strings.Contains(msg, "validation error") {
			t.Errorf("ValidationError.Error() = %q, should contain 'validation error'", msg)
		}
	})
}

func TestValidate_PanicOnNilSpec(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Validate() should panic with nil spec")
		}
		if !strings.Contains(r.(string), "Spec is required") {
			t.Errorf("Validate() panic message = %q, should contain 'Spec is required'", r)
		}
	}()

	Validate(ValidationConfig{
		Spec: nil,
	})
}

func TestValidate_SkipPaths(t *testing.T) {
	spec := NewSpec(Info{Title: "Test", Version: "1.0.0"})
	spec.AddPath("/test", PathItem{
		GET: &Operation{
			Responses: map[string]Response{
				"200": JSONResponse("Success", StringSchema()),
			},
		},
	})

	middleware := Validate(ValidationConfig{
		Spec:            spec,
		ValidateRequest: true,
		SkipPaths:       []string{"/health", "/metrics"},
	})

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Test skipped path
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/health")
	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	next := func(ctx *web.FastRequestContext) error {
		return nil
	}

	handler := middleware(next)
	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Validate() should skip /health path, got error: %v", err)
	}
}

func TestValidate_NoSpecForRoute(t *testing.T) {
	spec := NewSpec(Info{Title: "Test", Version: "1.0.0"})
	spec.AddPath("/test", PathItem{
		GET: &Operation{
			Responses: map[string]Response{
				"200": JSONResponse("Success", StringSchema()),
			},
		},
	})

	middleware := Validate(ValidationConfig{
		Spec:            spec,
		ValidateRequest: true,
	})

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Test path not in spec
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/unknown")
	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	next := func(ctx *web.FastRequestContext) error {
		return nil
	}

	handler := middleware(next)
	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Validate() should skip unknown paths, got error: %v", err)
	}
}

func TestFindOperation(t *testing.T) {
	spec := NewSpec(Info{Title: "Test", Version: "1.0.0"})
	spec.AddPath("/users", PathItem{
		GET: &Operation{
			Summary: "Get users",
			Responses: map[string]Response{
				"200": JSONResponse("Success", StringSchema()),
			},
		},
	})
	spec.AddPath("/users/{id}", PathItem{
		GET: &Operation{
			Summary: "Get user",
			Responses: map[string]Response{
				"200": JSONResponse("Success", StringSchema()),
			},
		},
	})

	t.Run("exact match", func(t *testing.T) {
		pathItem, operation := findOperation(spec, "GET", "/users")
		if pathItem == nil {
			t.Fatal("findOperation() pathItem is nil")
		}
		if operation == nil {
			t.Fatal("findOperation() operation is nil")
		}
		if operation.Summary != "Get users" {
			t.Errorf("findOperation() Summary = %q, want 'Get users'", operation.Summary)
		}
	})

	t.Run("pattern match", func(t *testing.T) {
		pathItem, operation := findOperation(spec, "GET", "/users/123")
		if pathItem == nil {
			t.Fatal("findOperation() pathItem is nil for pattern match")
		}
		if operation == nil {
			t.Fatal("findOperation() operation is nil for pattern match")
		}
		if operation.Summary != "Get user" {
			t.Errorf("findOperation() Summary = %q, want 'Get user'", operation.Summary)
		}
	})

	t.Run("no match", func(t *testing.T) {
		pathItem, operation := findOperation(spec, "GET", "/posts")
		if pathItem != nil || operation != nil {
			t.Error("findOperation() should return nil for non-matching path")
		}
	})

	t.Run("with query string", func(t *testing.T) {
		pathItem, operation := findOperation(spec, "GET", "/users?id=123")
		if pathItem == nil || operation == nil {
			t.Error("findOperation() should handle query strings")
		}
	})
}

func TestMatchesPath(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{
			name:    "exact match",
			pattern: "/users",
			path:    "/users",
			want:    true,
		},
		{
			name:    "pattern match with {id}",
			pattern: "/users/{id}",
			path:    "/users/123",
			want:    true,
		},
		{
			name:    "pattern match with multiple params",
			pattern: "/users/{userId}/posts/{postId}",
			path:    "/users/123/posts/456",
			want:    true,
		},
		{
			name:    "different length",
			pattern: "/users/{id}",
			path:    "/users/123/posts",
			want:    false,
		},
		{
			name:    "different static parts",
			pattern: "/users/{id}",
			path:    "/posts/123",
			want:    false,
		},
		{
			name:    "no match",
			pattern: "/users",
			path:    "/posts",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesPath(tt.pattern, tt.path)
			if got != tt.want {
				t.Errorf("matchesPath(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestGetOperation(t *testing.T) {
	pathItem := &PathItem{
		GET:    &Operation{Summary: "GET"},
		POST:   &Operation{Summary: "POST"},
		PUT:    &Operation{Summary: "PUT"},
		DELETE: &Operation{Summary: "DELETE"},
		PATCH:  &Operation{Summary: "PATCH"},
		HEAD:   &Operation{Summary: "HEAD"},
		OPTIONS: &Operation{Summary: "OPTIONS"},
		TRACE:  &Operation{Summary: "TRACE"},
	}

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			_, operation := getOperation(pathItem, method)
			if operation == nil {
				t.Errorf("getOperation() for %s returned nil", method)
			}
		})
	}

	t.Run("unknown method", func(t *testing.T) {
		_, operation := getOperation(pathItem, "UNKNOWN")
		if operation != nil {
			t.Error("getOperation() for unknown method should return nil")
		}
	})
}

func TestValidatePathParameters(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	pathItem := &PathItem{
		Parameters: []Parameter{
			{
				Name:     "id",
				In:       "path",
				Required: true,
				Schema:   IntegerSchema(),
			},
		},
	}
	operation := &Operation{}

	t.Run("missing required parameter", func(t *testing.T) {
		reqCtx := &fasthttp.RequestCtx{}
		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := validatePathParameters(fastCtx, pathItem, operation)
		if err == nil {
			t.Error("validatePathParameters() should return error for missing required parameter")
		}
		if !strings.Contains(err.Error(), "required path parameter is missing") {
			t.Errorf("validatePathParameters() error = %q, should mention missing parameter", err.Error())
		}
	})

	t.Run("valid parameter", func(t *testing.T) {
		reqCtx := &fasthttp.RequestCtx{}
		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             map[string]string{"id": "123"},
		}

		err := validatePathParameters(fastCtx, pathItem, operation)
		if err != nil {
			t.Errorf("validatePathParameters() error = %v, want nil", err)
		}
	})
}

func TestValidateQueryParameters(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	operation := &Operation{
		Parameters: []Parameter{
			{
				Name:     "page",
				In:       "query",
				Required: true,
				Schema:   IntegerSchema(),
			},
			{
				Name:     "limit",
				In:       "query",
				Required: false,
				Schema:   IntegerSchema(),
			},
		},
	}

	t.Run("missing required query parameter", func(t *testing.T) {
		reqCtx := &fasthttp.RequestCtx{}
		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := validateQueryParameters(fastCtx, operation)
		if err == nil {
			t.Error("validateQueryParameters() should return error for missing required parameter")
		}
	})

	t.Run("valid query parameters", func(t *testing.T) {
		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.SetRequestURI("/test?page=1&limit=10")
		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := validateQueryParameters(fastCtx, operation)
		if err != nil {
			t.Errorf("validateQueryParameters() error = %v, want nil", err)
		}
	})
}

func TestValidateHeaders(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	operation := &Operation{
		Parameters: []Parameter{
			{
				Name:     "Authorization",
				In:       "header",
				Required: true,
				Schema:   StringSchema(),
			},
		},
	}

	t.Run("missing required header", func(t *testing.T) {
		reqCtx := &fasthttp.RequestCtx{}
		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := validateHeaders(fastCtx, operation)
		if err == nil {
			t.Error("validateHeaders() should return error for missing required header")
		}
	})

	t.Run("valid header", func(t *testing.T) {
		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.Header.Set("Authorization", "Bearer token")
		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := validateHeaders(fastCtx, operation)
		if err != nil {
			t.Errorf("validateHeaders() error = %v, want nil", err)
		}
	})
}

func TestValidateRequestBody(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	userSchema := ObjectSchema(map[string]*Schema{
		"name":  StringSchema(),
		"email": StringSchema(),
	}, []string{"name", "email"})

	operation := &Operation{
		RequestBody: JSONRequestBody(userSchema, true),
	}

	t.Run("missing required body", func(t *testing.T) {
		reqCtx := &fasthttp.RequestCtx{}
		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := validateRequestBody(fastCtx, operation)
		if err == nil {
			t.Error("validateRequestBody() should return error for missing required body")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.Header.SetContentType("application/json")
		reqCtx.Request.SetBodyString("invalid json")
		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := validateRequestBody(fastCtx, operation)
		if err == nil {
			t.Error("validateRequestBody() should return error for invalid JSON")
		}
	})

	t.Run("valid body", func(t *testing.T) {
		body := map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		}
		bodyJSON, _ := json.Marshal(body)

		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.Header.SetContentType("application/json")
		reqCtx.Request.SetBody(bodyJSON)
		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := validateRequestBody(fastCtx, operation)
		if err != nil {
			t.Errorf("validateRequestBody() error = %v, want nil", err)
		}
	})

	t.Run("missing required field", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "John Doe",
			// email is missing
		}
		bodyJSON, _ := json.Marshal(body)

		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.Header.SetContentType("application/json")
		reqCtx.Request.SetBody(bodyJSON)
		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := validateRequestBody(fastCtx, operation)
		if err == nil {
			t.Error("validateRequestBody() should return error for missing required field")
		}
	})
}

func TestValidateValue(t *testing.T) {
	t.Run("string type", func(t *testing.T) {
		schema := StringSchema()
		err := validateValue("test", schema, "field")
		if err != nil {
			t.Errorf("validateValue() error = %v, want nil", err)
		}

		err = validateValue(123, schema, "field")
		if err == nil {
			t.Error("validateValue() should return error for wrong type")
		}
	})

	t.Run("string minLength", func(t *testing.T) {
		minLen := 5
		schema := &Schema{
			Type:      "string",
			MinLength: &minLen,
		}
		err := validateValue("test", schema, "field")
		if err == nil {
			t.Error("validateValue() should return error for string shorter than minLength")
		}

		err = validateValue("test123", schema, "field")
		if err != nil {
			t.Errorf("validateValue() error = %v, want nil", err)
		}
	})

	t.Run("string maxLength", func(t *testing.T) {
		maxLen := 5
		schema := &Schema{
			Type:      "string",
			MaxLength: &maxLen,
		}
		err := validateValue("test123", schema, "field")
		if err == nil {
			t.Error("validateValue() should return error for string longer than maxLength")
		}

		err = validateValue("test", schema, "field")
		if err != nil {
			t.Errorf("validateValue() error = %v, want nil", err)
		}
	})

	t.Run("string pattern", func(t *testing.T) {
		schema := &Schema{
			Type:    "string",
			Pattern: "^[a-z]+$",
		}
		err := validateValue("test123", schema, "field")
		if err == nil {
			t.Error("validateValue() should return error for string not matching pattern")
		}

		err = validateValue("test", schema, "field")
		if err != nil {
			t.Errorf("validateValue() error = %v, want nil", err)
		}
	})

	t.Run("integer type", func(t *testing.T) {
		schema := IntegerSchema()
		err := validateValue(123, schema, "field")
		if err != nil {
			t.Errorf("validateValue() error = %v, want nil", err)
		}

		err = validateValue("not a number", schema, "field")
		if err == nil {
			t.Error("validateValue() should return error for non-integer")
		}
	})

	t.Run("number minimum/maximum", func(t *testing.T) {
		min := 10.0
		max := 100.0
		schema := &Schema{
			Type:     "number",
			Minimum:  &min,
			Maximum:  &max,
		}
		err := validateValue(5.0, schema, "field")
		if err == nil {
			t.Error("validateValue() should return error for value below minimum")
		}

		err = validateValue(150.0, schema, "field")
		if err == nil {
			t.Error("validateValue() should return error for value above maximum")
		}

		err = validateValue(50.0, schema, "field")
		if err != nil {
			t.Errorf("validateValue() error = %v, want nil", err)
		}
	})

	t.Run("enum", func(t *testing.T) {
		schema := &Schema{
			Type: "string",
			Enum: []interface{}{"red", "green", "blue"},
		}
		err := validateValue("red", schema, "field")
		if err != nil {
			t.Errorf("validateValue() error = %v, want nil", err)
		}

		err = validateValue("yellow", schema, "field")
		if err == nil {
			t.Error("validateValue() should return error for value not in enum")
		}
	})

	t.Run("object required properties", func(t *testing.T) {
		schema := ObjectSchema(map[string]*Schema{
			"name":  StringSchema(),
			"email": StringSchema(),
		}, []string{"name", "email"})

		obj := map[string]interface{}{
			"name": "John",
		}
		err := validateValue(obj, schema, "field")
		if err == nil {
			t.Error("validateValue() should return error for missing required property")
		}

		obj["email"] = "john@example.com"
		err = validateValue(obj, schema, "field")
		if err != nil {
			t.Errorf("validateValue() error = %v, want nil", err)
		}
	})

	t.Run("array minItems/maxItems", func(t *testing.T) {
		minItems := 2
		maxItems := 5
		schema := &Schema{
			Type:     "array",
			Items:    StringSchema(),
			MinItems: &minItems,
			MaxItems: &maxItems,
		}

		arr := []interface{}{"one"}
		err := validateValue(arr, schema, "field")
		if err == nil {
			t.Error("validateValue() should return error for array with fewer than minItems")
		}

		arr = []interface{}{"one", "two", "three", "four", "five", "six"}
		err = validateValue(arr, schema, "field")
		if err == nil {
			t.Error("validateValue() should return error for array with more than maxItems")
		}

		arr = []interface{}{"one", "two", "three"}
		err = validateValue(arr, schema, "field")
		if err != nil {
			t.Errorf("validateValue() error = %v, want nil", err)
		}
	})
}

func TestValidateType(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectedType string
		wantErr     bool
	}{
		{"string valid", "test", "string", false},
		{"string invalid", 123, "string", true},
		{"integer valid", 123, "integer", false},
		{"integer valid float64", 123.0, "integer", false},
		{"integer valid string", "123", "integer", false},
		{"integer invalid", "abc", "integer", true},
		{"number valid", 123.45, "number", false},
		{"number valid int", 123, "number", false},
		{"number valid string", "123.45", "number", false},
		{"number invalid", "abc", "number", true},
		{"boolean valid", true, "boolean", false},
		{"boolean invalid", "true", "boolean", true},
		{"array valid", []interface{}{1, 2, 3}, "array", false},
		{"array invalid", "not array", "array", true},
		{"object valid", map[string]interface{}{"key": "value"}, "object", false},
		{"object invalid", "not object", "object", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateType(tt.value, tt.expectedType, "field")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
