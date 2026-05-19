package openapi

import (
	"testing"
)

func TestNewBuilder(t *testing.T) {
	info := Info{
		Title:       "Test API",
		Description: "Test API Description",
		Version:     "1.0.0",
	}

	builder := NewBuilder(info)
	if builder == nil {
		t.Fatal("NewBuilder() returned nil")
	}
	if builder.spec == nil {
		t.Error("NewBuilder() spec is nil")
	}
	if builder.spec.Info.Title != "Test API" {
		t.Errorf("NewBuilder() spec.Info.Title = %q, want 'Test API'", builder.spec.Info.Title)
	}
	if len(builder.routes) != 0 {
		t.Errorf("NewBuilder() routes length = %d, want 0", len(builder.routes))
	}
}

func TestBuilder_AddRoute(t *testing.T) {
	builder := NewBuilder(Info{Title: "Test", Version: "1.0.0"})

	route := RouteInfo{
		Method:      "GET",
		Path:        "/users",
		Summary:     "Get users",
		Description: "Get all users",
		OperationID: "getUsers",
		Tags:        []string{"users"},
	}

	builder.AddRoute(route)

	if len(builder.routes) != 1 {
		t.Errorf("AddRoute() routes length = %d, want 1", len(builder.routes))
	}
	if builder.routes[0].Method != "GET" {
		t.Errorf("AddRoute() route.Method = %q, want 'GET'", builder.routes[0].Method)
	}
	if builder.routes[0].Path != "/users" {
		t.Errorf("AddRoute() route.Path = %q, want '/users'", builder.routes[0].Path)
	}
}

func TestBuilder_Build(t *testing.T) {
	builder := NewBuilder(Info{Title: "Test API", Version: "1.0.0"})

	// Add GET route
	builder.AddRoute(RouteInfo{
		Method:      "GET",
		Path:        "/users",
		Summary:     "Get users",
		OperationID: "getUsers",
		Responses: map[string]Response{
			"200": JSONResponse("Success", StringSchema()),
		},
	})

	// Add POST route
	builder.AddRoute(RouteInfo{
		Method:      "POST",
		Path:        "/users",
		Summary:     "Create user",
		OperationID: "createUser",
		Responses: map[string]Response{
			"201": JSONResponse("Created", StringSchema()),
		},
	})

	// Add route with different path
	builder.AddRoute(RouteInfo{
		Method:      "GET",
		Path:        "/posts",
		Summary:     "Get posts",
		OperationID: "getPosts",
		Responses: map[string]Response{
			"200": JSONResponse("Success", StringSchema()),
		},
	})

	spec := builder.Build()

	if spec == nil {
		t.Fatal("Build() returned nil")
	}
	if len(spec.Paths) != 2 {
		t.Errorf("Build() paths count = %d, want 2", len(spec.Paths))
	}

	// Check /users path
	usersPath, exists := spec.Paths["/users"]
	if !exists {
		t.Fatal("Build() /users path not found")
	}
	if usersPath.GET == nil {
		t.Error("Build() /users GET operation is nil")
	}
	if usersPath.POST == nil {
		t.Error("Build() /users POST operation is nil")
	}
	if usersPath.GET.OperationID != "getUsers" {
		t.Errorf("Build() GET OperationID = %q, want 'getUsers'", usersPath.GET.OperationID)
	}
	if usersPath.POST.OperationID != "createUser" {
		t.Errorf("Build() POST OperationID = %q, want 'createUser'", usersPath.POST.OperationID)
	}

	// Check /posts path
	postsPath, exists := spec.Paths["/posts"]
	if !exists {
		t.Fatal("Build() /posts path not found")
	}
	if postsPath.GET == nil {
		t.Error("Build() /posts GET operation is nil")
	}
}

func TestBuilder_Build_AllMethods(t *testing.T) {
	builder := NewBuilder(Info{Title: "Test API", Version: "1.0.0"})

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE"}
	for _, method := range methods {
		builder.AddRoute(RouteInfo{
			Method:      method,
			Path:        "/test",
			OperationID: "test" + method,
			Responses: map[string]Response{
				"200": JSONResponse("Success", StringSchema()),
			},
		})
	}

	spec := builder.Build()
	pathItem := spec.Paths["/test"]

	if pathItem.GET == nil {
		t.Error("Build() GET operation is nil")
	}
	if pathItem.POST == nil {
		t.Error("Build() POST operation is nil")
	}
	if pathItem.PUT == nil {
		t.Error("Build() PUT operation is nil")
	}
	if pathItem.DELETE == nil {
		t.Error("Build() DELETE operation is nil")
	}
	if pathItem.PATCH == nil {
		t.Error("Build() PATCH operation is nil")
	}
	if pathItem.HEAD == nil {
		t.Error("Build() HEAD operation is nil")
	}
	if pathItem.OPTIONS == nil {
		t.Error("Build() OPTIONS operation is nil")
	}
	if pathItem.TRACE == nil {
		t.Error("Build() TRACE operation is nil")
	}
}

func TestBuilder_Build_PathParameters(t *testing.T) {
	builder := NewBuilder(Info{Title: "Test API", Version: "1.0.0"})

	// Add route with :id parameter
	builder.AddRoute(RouteInfo{
		Method:      "GET",
		Path:        "/users/:id",
		Summary:     "Get user by ID",
		OperationID: "getUser",
		Responses: map[string]Response{
			"200": JSONResponse("Success", StringSchema()),
		},
	})

	// Add route with {id} parameter
	builder.AddRoute(RouteInfo{
		Method:      "GET",
		Path:        "/posts/{postId}",
		Summary:     "Get post by ID",
		OperationID: "getPost",
		Responses: map[string]Response{
			"200": JSONResponse("Success", StringSchema()),
		},
	})

	// Add route with multiple parameters
	builder.AddRoute(RouteInfo{
		Method:      "GET",
		Path:        "/users/:userId/posts/:postId",
		Summary:     "Get user post",
		OperationID: "getUserPost",
		Responses: map[string]Response{
			"200": JSONResponse("Success", StringSchema()),
		},
	})

	spec := builder.Build()

	// Check /users/:id path
	usersPath := spec.Paths["/users/:id"]
	if len(usersPath.Parameters) != 1 {
		t.Errorf("Build() /users/:id parameters count = %d, want 1", len(usersPath.Parameters))
	}
	if usersPath.Parameters[0].Name != "id" {
		t.Errorf("Build() parameter name = %q, want 'id'", usersPath.Parameters[0].Name)
	}
	if usersPath.Parameters[0].In != "path" {
		t.Errorf("Build() parameter In = %q, want 'path'", usersPath.Parameters[0].In)
	}
	if !usersPath.Parameters[0].Required {
		t.Error("Build() path parameter should be required")
	}

	// Check /posts/{postId} path
	postsPath := spec.Paths["/posts/{postId}"]
	if len(postsPath.Parameters) != 1 {
		t.Errorf("Build() /posts/{postId} parameters count = %d, want 1", len(postsPath.Parameters))
	}
	if postsPath.Parameters[0].Name != "postId" {
		t.Errorf("Build() parameter name = %q, want 'postId'", postsPath.Parameters[0].Name)
	}

	// Check /users/:userId/posts/:postId path
	userPostsPath := spec.Paths["/users/:userId/posts/:postId"]
	if len(userPostsPath.Parameters) != 2 {
		t.Errorf("Build() /users/:userId/posts/:postId parameters count = %d, want 2", len(userPostsPath.Parameters))
	}
}

func TestExtractPathParameters(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		want     []string
	}{
		{
			name:     "no parameters",
			path:     "/users",
			want:     []string{},
		},
		{
			name:     "single :id parameter",
			path:     "/users/:id",
			want:     []string{"id"},
		},
		{
			name:     "single {id} parameter",
			path:     "/users/{id}",
			want:     []string{"id"},
		},
		{
			name:     "multiple : parameters",
			path:     "/users/:userId/posts/:postId",
			want:     []string{"userId", "postId"},
		},
		{
			name:     "multiple { } parameters",
			path:     "/users/{userId}/posts/{postId}",
			want:     []string{"userId", "postId"},
		},
		{
			name:     "mixed : and { } parameters",
			path:     "/users/:userId/posts/{postId}",
			want:     []string{"userId", "postId"},
		},
		{
			name:     "empty parameter name",
			path:     "/users/:",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPathParameters(tt.path)
			if len(got) != len(tt.want) {
				t.Errorf("extractPathParameters(%q) = %v, want %v", tt.path, got, tt.want)
				return
			}
			for i, param := range got {
				if param != tt.want[i] {
					t.Errorf("extractPathParameters(%q)[%d] = %q, want %q", tt.path, i, param, tt.want[i])
				}
			}
		})
	}
}

func TestStringSchema(t *testing.T) {
	schema := StringSchema()
	if schema == nil {
		t.Fatal("StringSchema() returned nil")
	}
	if schema.Type != "string" {
		t.Errorf("StringSchema() Type = %q, want 'string'", schema.Type)
	}
}

func TestIntegerSchema(t *testing.T) {
	schema := IntegerSchema()
	if schema == nil {
		t.Fatal("IntegerSchema() returned nil")
	}
	if schema.Type != "integer" {
		t.Errorf("IntegerSchema() Type = %q, want 'integer'", schema.Type)
	}
	if schema.Format != "int64" {
		t.Errorf("IntegerSchema() Format = %q, want 'int64'", schema.Format)
	}
}

func TestNumberSchema(t *testing.T) {
	schema := NumberSchema()
	if schema == nil {
		t.Fatal("NumberSchema() returned nil")
	}
	if schema.Type != "number" {
		t.Errorf("NumberSchema() Type = %q, want 'number'", schema.Type)
	}
}

func TestBooleanSchema(t *testing.T) {
	schema := BooleanSchema()
	if schema == nil {
		t.Fatal("BooleanSchema() returned nil")
	}
	if schema.Type != "boolean" {
		t.Errorf("BooleanSchema() Type = %q, want 'boolean'", schema.Type)
	}
}

func TestArraySchema(t *testing.T) {
	itemSchema := StringSchema()
	schema := ArraySchema(itemSchema)
	if schema == nil {
		t.Fatal("ArraySchema() returned nil")
	}
	if schema.Type != "array" {
		t.Errorf("ArraySchema() Type = %q, want 'array'", schema.Type)
	}
	if schema.Items != itemSchema {
		t.Error("ArraySchema() Items not set correctly")
	}
}

func TestObjectSchema(t *testing.T) {
	properties := map[string]*Schema{
		"name": StringSchema(),
		"age":  IntegerSchema(),
	}
	required := []string{"name"}

	schema := ObjectSchema(properties, required)
	if schema == nil {
		t.Fatal("ObjectSchema() returned nil")
	}
	if schema.Type != "object" {
		t.Errorf("ObjectSchema() Type = %q, want 'object'", schema.Type)
	}
	if len(schema.Properties) != 2 {
		t.Errorf("ObjectSchema() Properties count = %d, want 2", len(schema.Properties))
	}
	if len(schema.Required) != 1 {
		t.Errorf("ObjectSchema() Required count = %d, want 1", len(schema.Required))
	}
	if schema.Required[0] != "name" {
		t.Errorf("ObjectSchema() Required[0] = %q, want 'name'", schema.Required[0])
	}
}

func TestRefSchema(t *testing.T) {
	ref := "#/components/schemas/User"
	schema := RefSchema(ref)
	if schema == nil {
		t.Fatal("RefSchema() returned nil")
	}
	if schema.Ref != ref {
		t.Errorf("RefSchema() Ref = %q, want %q", schema.Ref, ref)
	}
}

func TestJSONResponse(t *testing.T) {
	schema := StringSchema()
	response := JSONResponse("Success response", schema)

	if response.Description != "Success response" {
		t.Errorf("JSONResponse() Description = %q, want 'Success response'", response.Description)
	}
	if response.Content == nil {
		t.Fatal("JSONResponse() Content is nil")
	}
	mediaType, exists := response.Content["application/json"]
	if !exists {
		t.Fatal("JSONResponse() application/json media type not found")
	}
	if mediaType.Schema != schema {
		t.Error("JSONResponse() Schema not set correctly")
	}
}

func TestJSONRequestBody(t *testing.T) {
	schema := ObjectSchema(map[string]*Schema{
		"name": StringSchema(),
	}, []string{"name"})

	requestBody := JSONRequestBody(schema, true)
	if requestBody == nil {
		t.Fatal("JSONRequestBody() returned nil")
	}
	if !requestBody.Required {
		t.Error("JSONRequestBody() Required should be true")
	}
	if requestBody.Content == nil {
		t.Fatal("JSONRequestBody() Content is nil")
	}
	mediaType, exists := requestBody.Content["application/json"]
	if !exists {
		t.Fatal("JSONRequestBody() application/json media type not found")
	}
	if mediaType.Schema != schema {
		t.Error("JSONRequestBody() Schema not set correctly")
	}
}

func TestErrorResponse(t *testing.T) {
	response := ErrorResponse("Error occurred")
	if response.Description != "Error occurred" {
		t.Errorf("ErrorResponse() Description = %q, want 'Error occurred'", response.Description)
	}
	if response.Content == nil {
		t.Fatal("ErrorResponse() Content is nil")
	}
	mediaType, exists := response.Content["application/json"]
	if !exists {
		t.Fatal("ErrorResponse() application/json media type not found")
	}
	if mediaType.Schema == nil {
		t.Fatal("ErrorResponse() Schema is nil")
	}
	if mediaType.Schema.Type != "object" {
		t.Errorf("ErrorResponse() Schema.Type = %q, want 'object'", mediaType.Schema.Type)
	}
	if _, exists := mediaType.Schema.Properties["error"]; !exists {
		t.Error("ErrorResponse() Schema missing 'error' property")
	}
	if _, exists := mediaType.Schema.Properties["message"]; !exists {
		t.Error("ErrorResponse() Schema missing 'message' property")
	}
}

func TestBearerAuthScheme(t *testing.T) {
	description := "Bearer token authentication"
	scheme := BearerAuthScheme(description)

	if scheme.Type != "http" {
		t.Errorf("BearerAuthScheme() Type = %q, want 'http'", scheme.Type)
	}
	if scheme.Scheme != "bearer" {
		t.Errorf("BearerAuthScheme() Scheme = %q, want 'bearer'", scheme.Scheme)
	}
	if scheme.BearerFormat != "JWT" {
		t.Errorf("BearerAuthScheme() BearerFormat = %q, want 'JWT'", scheme.BearerFormat)
	}
	if scheme.Description != description {
		t.Errorf("BearerAuthScheme() Description = %q, want %q", scheme.Description, description)
	}
}

func TestAPIKeyAuthScheme(t *testing.T) {
	name := "X-API-Key"
	in := "header"
	description := "API key authentication"
	scheme := APIKeyAuthScheme(name, in, description)

	if scheme.Type != "apiKey" {
		t.Errorf("APIKeyAuthScheme() Type = %q, want 'apiKey'", scheme.Type)
	}
	if scheme.Name != name {
		t.Errorf("APIKeyAuthScheme() Name = %q, want %q", scheme.Name, name)
	}
	if scheme.In != in {
		t.Errorf("APIKeyAuthScheme() In = %q, want %q", scheme.In, in)
	}
	if scheme.Description != description {
		t.Errorf("APIKeyAuthScheme() Description = %q, want %q", scheme.Description, description)
	}
}
