package openapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewSpec(t *testing.T) {
	info := Info{
		Title:       "Test API",
		Description: "Test API Description",
		Version:     "1.0.0",
	}

	spec := NewSpec(info)
	if spec == nil {
		t.Fatal("NewSpec() returned nil")
	}
	if spec.OpenAPI != "3.0.0" {
		t.Errorf("NewSpec() OpenAPI = %q, want '3.0.0'", spec.OpenAPI)
	}
	if spec.Info.Title != "Test API" {
		t.Errorf("NewSpec() Info.Title = %q, want 'Test API'", spec.Info.Title)
	}
	if spec.Paths == nil {
		t.Error("NewSpec() Paths is nil")
	}
	if spec.Components == nil {
		t.Error("NewSpec() Components is nil")
	}
	if spec.Components.Schemas == nil {
		t.Error("NewSpec() Components.Schemas is nil")
	}
	if spec.Components.SecuritySchemes == nil {
		t.Error("NewSpec() Components.SecuritySchemes is nil")
	}
}

func TestSpec_AddPath(t *testing.T) {
	spec := NewSpec(Info{Title: "Test", Version: "1.0.0"})

	pathItem := PathItem{
		Summary: "Test path",
		GET: &Operation{
			Summary: "Get operation",
		},
	}

	spec.AddPath("/test", pathItem)

	if len(spec.Paths) != 1 {
		t.Errorf("AddPath() paths count = %d, want 1", len(spec.Paths))
	}
	addedPath, exists := spec.Paths["/test"]
	if !exists {
		t.Fatal("AddPath() path not found")
	}
	if addedPath.Summary != "Test path" {
		t.Errorf("AddPath() path.Summary = %q, want 'Test path'", addedPath.Summary)
	}
	if addedPath.GET == nil {
		t.Error("AddPath() GET operation is nil")
	}
}

func TestSpec_ToJSON(t *testing.T) {
	spec := NewSpec(Info{
		Title:       "Test API",
		Description: "Test API Description",
		Version:     "1.0.0",
	})

	// Add a path
	spec.AddPath("/test", PathItem{
		GET: &Operation{
			Summary: "Get test",
			Responses: map[string]Response{
				"200": JSONResponse("Success", StringSchema()),
			},
		},
	})

	jsonData, err := spec.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}
	if len(jsonData) == 0 {
		t.Error("ToJSON() returned empty data")
	}

	// Verify it's valid JSON
	var decoded Spec
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("ToJSON() returned invalid JSON: %v", err)
	}
	if decoded.OpenAPI != "3.0.0" {
		t.Errorf("ToJSON() decoded OpenAPI = %q, want '3.0.0'", decoded.OpenAPI)
	}
	if decoded.Info.Title != "Test API" {
		t.Errorf("ToJSON() decoded Info.Title = %q, want 'Test API'", decoded.Info.Title)
	}
}

func TestSpec_ToYAML(t *testing.T) {
	spec := NewSpec(Info{Title: "Test", Version: "1.0.0"})

	_, err := spec.ToYAML()
	if err == nil {
		t.Error("ToYAML() should return error (YAML support not implemented)")
	}
	if err.Error()[:5] != "YAML " {
		t.Errorf("ToYAML() error message should start with 'YAML ', got %q", err.Error())
	}
}

func TestSpec_Complex(t *testing.T) {
	spec := NewSpec(Info{
		Title:       "Complex API",
		Description: "A complex API with multiple paths and operations",
		Version:     "2.0.0",
		Contact: &Contact{
			Name:  "API Support",
			Email: "support@example.com",
		},
		License: &License{
			Name: "MIT",
			URL:  "https://opensource.org/licenses/MIT",
		},
	})

	// Add servers
	spec.Servers = []Server{
		{
			URL:         "https://api.example.com",
			Description: "Production server",
		},
		{
			URL:         "https://staging-api.example.com",
			Description: "Staging server",
		},
	}

	// Add security schemes
	spec.Components.SecuritySchemes["bearerAuth"] = BearerAuthScheme("Bearer token authentication")
	spec.Components.SecuritySchemes["apiKey"] = APIKeyAuthScheme("X-API-Key", "header", "API key authentication")

	// Add schemas
	userSchema := ObjectSchema(map[string]*Schema{
		"id":    IntegerSchema(),
		"name":  StringSchema(),
		"email": StringSchema(),
	}, []string{"id", "name", "email"})

	spec.Components.Schemas["User"] = userSchema

	// Add paths with operations
	spec.AddPath("/users", PathItem{
		GET: &Operation{
			Summary:     "List users",
			OperationID: "listUsers",
			Tags:        []string{"users"},
			Responses: map[string]Response{
				"200": JSONResponse("List of users", ArraySchema(userSchema)),
			},
		},
		POST: &Operation{
			Summary:     "Create user",
			OperationID: "createUser",
			Tags:        []string{"users"},
			RequestBody: JSONRequestBody(userSchema, true),
			Responses: map[string]Response{
				"201": JSONResponse("User created", userSchema),
				"400": ErrorResponse("Invalid input"),
			},
			Security: []SecurityRequirement{
				{"bearerAuth": []string{}},
			},
		},
	})

	spec.AddPath("/users/{id}", PathItem{
		Parameters: []Parameter{
			{
				Name:     "id",
				In:       "path",
				Required: true,
				Schema:   IntegerSchema(),
			},
		},
		GET: &Operation{
			Summary:     "Get user by ID",
			OperationID: "getUser",
			Tags:        []string{"users"},
			Responses: map[string]Response{
				"200": JSONResponse("User found", userSchema),
				"404": ErrorResponse("User not found"),
			},
		},
		PUT: &Operation{
			Summary:     "Update user",
			OperationID: "updateUser",
			Tags:        []string{"users"},
			RequestBody: JSONRequestBody(userSchema, true),
			Responses: map[string]Response{
				"200": JSONResponse("User updated", userSchema),
				"404": ErrorResponse("User not found"),
			},
		},
		DELETE: &Operation{
			Summary:     "Delete user",
			OperationID: "deleteUser",
			Tags:        []string{"users"},
			Responses: map[string]Response{
				"204": Response{Description: "User deleted"},
				"404": ErrorResponse("User not found"),
			},
		},
	})

	// Verify spec structure
	if len(spec.Paths) != 2 {
		t.Errorf("Spec paths count = %d, want 2", len(spec.Paths))
	}
	if len(spec.Servers) != 2 {
		t.Errorf("Spec servers count = %d, want 2", len(spec.Servers))
	}
	if len(spec.Components.SecuritySchemes) != 2 {
		t.Errorf("Spec security schemes count = %d, want 2", len(spec.Components.SecuritySchemes))
	}
	if len(spec.Components.Schemas) != 1 {
		t.Errorf("Spec schemas count = %d, want 1", len(spec.Components.Schemas))
	}

	// Test JSON serialization
	jsonData, err := spec.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "openapi") {
		t.Error("ToJSON() missing 'openapi' field")
	}
	if !strings.Contains(jsonStr, "info") {
		t.Error("ToJSON() missing 'info' field")
	}
	if !strings.Contains(jsonStr, "paths") {
		t.Error("ToJSON() missing 'paths' field")
	}
	if !strings.Contains(jsonStr, "components") {
		t.Error("ToJSON() missing 'components' field")
	}
	if !strings.Contains(jsonStr, "servers") {
		t.Error("ToJSON() missing 'servers' field")
	}
}
