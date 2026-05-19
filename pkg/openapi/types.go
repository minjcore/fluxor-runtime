package openapi

import (
	"encoding/json"
	"fmt"
)

// Spec represents an OpenAPI 3.0 specification
type Spec struct {
	OpenAPI      string                 `json:"openapi" yaml:"openapi"`
	Info         Info                   `json:"info" yaml:"info"`
	Servers      []Server               `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths        map[string]PathItem    `json:"paths" yaml:"paths"`
	Components   *Components            `json:"components,omitempty" yaml:"components,omitempty"`
	Security     []SecurityRequirement  `json:"security,omitempty" yaml:"security,omitempty"`
	Tags         []Tag                  `json:"tags,omitempty" yaml:"tags,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

// Info provides metadata about the API
type Info struct {
	Title          string  `json:"title" yaml:"title"`
	Description    string  `json:"description,omitempty" yaml:"description,omitempty"`
	TermsOfService string  `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	License        *License `json:"license,omitempty" yaml:"license,omitempty"`
	Version        string  `json:"version" yaml:"version"`
}

// Contact information for the API
type Contact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

// License information for the API
type License struct {
	Name string `json:"name" yaml:"name"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

// Server represents a server
type Server struct {
	URL         string                    `json:"url" yaml:"url"`
	Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// ServerVariable represents a server variable
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default     string   `json:"default" yaml:"default"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

// PathItem describes the operations available on a single path
type PathItem struct {
	Ref         string     `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Summary     string     `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	GET         *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	PUT         *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	POST        *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	DELETE      *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	OPTIONS     *Operation `json:"options,omitempty" yaml:"options,omitempty"`
	HEAD        *Operation `json:"head,omitempty" yaml:"head,omitempty"`
	PATCH       *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
	TRACE       *Operation `json:"trace,omitempty" yaml:"trace,omitempty"`
	Servers     []Server   `json:"servers,omitempty" yaml:"servers,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// Operation describes a single API operation
type Operation struct {
	Tags         []string                `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary      string                  `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description  string                  `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID  string                  `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters   []Parameter             `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody  *RequestBody            `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses    map[string]Response     `json:"responses" yaml:"responses"`
	Deprecated   bool                    `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Security     []SecurityRequirement   `json:"security,omitempty" yaml:"security,omitempty"`
	ExternalDocs *ExternalDocumentation  `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

// Parameter describes a single operation parameter
type Parameter struct {
	Name            string      `json:"name" yaml:"name"`
	In              string      `json:"in" yaml:"in"` // query, header, path, cookie
	Description     string      `json:"description,omitempty" yaml:"description,omitempty"`
	Required        bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated      bool        `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	AllowEmptyValue bool        `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	Schema          *Schema     `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example         interface{} `json:"example,omitempty" yaml:"example,omitempty"`
}

// RequestBody describes a single request body
type RequestBody struct {
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]MediaType `json:"content" yaml:"content"`
	Required    bool                 `json:"required,omitempty" yaml:"required,omitempty"`
}

// Response describes a single response
type Response struct {
	Description string               `json:"description" yaml:"description"`
	Headers     map[string]Header    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
	Links       map[string]Link      `json:"links,omitempty" yaml:"links,omitempty"`
}

// Header describes a header
type Header struct {
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated  bool        `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Schema      *Schema     `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty" yaml:"example,omitempty"`
}

// MediaType provides schema and examples for a media type
type MediaType struct {
	Schema   *Schema              `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  interface{}          `json:"example,omitempty" yaml:"example,omitempty"`
	Examples map[string]Example   `json:"examples,omitempty" yaml:"examples,omitempty"`
	Encoding map[string]Encoding  `json:"encoding,omitempty" yaml:"encoding,omitempty"`
}

// Encoding describes encoding information for a media type
type Encoding struct {
	ContentType   string             `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	Headers       map[string]Header  `json:"headers,omitempty" yaml:"headers,omitempty"`
	Style         string             `json:"style,omitempty" yaml:"style,omitempty"`
	Explode       bool               `json:"explode,omitempty" yaml:"explode,omitempty"`
	AllowReserved bool               `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
}

// Example provides example values
type Example struct {
	Summary       string      `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description   string      `json:"description,omitempty" yaml:"description,omitempty"`
	Value         interface{} `json:"value,omitempty" yaml:"value,omitempty"`
	ExternalValue string      `json:"externalValue,omitempty" yaml:"externalValue,omitempty"`
}

// Link represents a possible design-time link for a response
type Link struct {
	OperationRef string                 `json:"operationRef,omitempty" yaml:"operationRef,omitempty"`
	OperationID  string                 `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody  interface{}            `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Description  string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Server       *Server                `json:"server,omitempty" yaml:"server,omitempty"`
}

// Schema represents a JSON Schema
type Schema struct {
	Type                 string             `json:"type,omitempty" yaml:"type,omitempty"`
	Format               string             `json:"format,omitempty" yaml:"format,omitempty"`
	Title                string             `json:"title,omitempty" yaml:"title,omitempty"`
	Description          string             `json:"description,omitempty" yaml:"description,omitempty"`
	Default              interface{}        `json:"default,omitempty" yaml:"default,omitempty"`
	Example              interface{}        `json:"example,omitempty" yaml:"example,omitempty"`
	Enum                 []interface{}      `json:"enum,omitempty" yaml:"enum,omitempty"`
	Const                interface{}        `json:"const,omitempty" yaml:"const,omitempty"`
	MultipleOf           *float64           `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMaximum     *float64           `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	ExclusiveMinimum     *float64           `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength            *int               `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	Pattern              string             `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	MaxItems             *int               `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MinItems             *int               `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	UniqueItems          bool               `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	MaxProperties        *int               `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	MinProperties        *int               `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	Required             []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	AdditionalProperties interface{}        `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	Items                *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	OneOf                []*Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf                []*Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	Not                  *Schema            `json:"not,omitempty" yaml:"not,omitempty"`
	Ref                  string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Nullable             bool               `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	ReadOnly             bool               `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	WriteOnly            bool               `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
	Deprecated           bool               `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}

// Components holds reusable components
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Responses       map[string]Response        `json:"responses,omitempty" yaml:"responses,omitempty"`
	Parameters      map[string]Parameter       `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Examples        map[string]Example         `json:"examples,omitempty" yaml:"examples,omitempty"`
	RequestBodies   map[string]RequestBody     `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
	Headers         map[string]Header          `json:"headers,omitempty" yaml:"headers,omitempty"`
	SecuritySchemes map[string]SecurityScheme  `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
	Links           map[string]Link            `json:"links,omitempty" yaml:"links,omitempty"`
}

// SecurityScheme defines a security scheme
type SecurityScheme struct {
	Type             string            `json:"type" yaml:"type"`
	Description      string            `json:"description,omitempty" yaml:"description,omitempty"`
	Name             string            `json:"name,omitempty" yaml:"name,omitempty"`
	In               string            `json:"in,omitempty" yaml:"in,omitempty"`
	Scheme           string            `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	BearerFormat     string            `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
	Flows            *OAuthFlows       `json:"flows,omitempty" yaml:"flows,omitempty"`
	OpenIDConnectURL string            `json:"openIdConnectUrl,omitempty" yaml:"openIdConnectUrl,omitempty"`
}

// OAuthFlows configuration
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty" yaml:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty" yaml:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty" yaml:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty" yaml:"authorizationCode,omitempty"`
}

// OAuthFlow configuration
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes" yaml:"scopes"`
}

// SecurityRequirement lists required security schemes
type SecurityRequirement map[string][]string

// Tag adds metadata to a single tag
type Tag struct {
	Name         string                 `json:"name" yaml:"name"`
	Description  string                 `json:"description,omitempty" yaml:"description,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

// ExternalDocumentation allows referencing an external resource
type ExternalDocumentation struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string `json:"url" yaml:"url"`
}

// NewSpec creates a new OpenAPI 3.0 specification
func NewSpec(info Info) *Spec {
	return &Spec{
		OpenAPI:    "3.0.0",
		Info:       info,
		Paths:      make(map[string]PathItem),
		Components: &Components{
			Schemas:         make(map[string]*Schema),
			SecuritySchemes: make(map[string]SecurityScheme),
		},
	}
}

// AddPath adds a path to the specification
func (s *Spec) AddPath(path string, item PathItem) {
	s.Paths[path] = item
}

// ToJSON returns the spec as JSON
func (s *Spec) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// ToYAML returns the spec as YAML (requires yaml.v3)
func (s *Spec) ToYAML() ([]byte, error) {
	// For now, return error if yaml support is not available
	// In production, use gopkg.in/yaml.v3
	return nil, fmt.Errorf("YAML support requires yaml.v3 package")
}
