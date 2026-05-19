package http

import (
	"context"
	"io"
)

// Client is the main HTTP client interface
type Client interface {
	// Get performs a GET request
	Get(ctx context.Context, url string, headers map[string]string) (*Response, error)

	// Post performs a POST request
	Post(ctx context.Context, url string, body interface{}, headers map[string]string) (*Response, error)

	// Put performs a PUT request
	Put(ctx context.Context, url string, body interface{}, headers map[string]string) (*Response, error)

	// Patch performs a PATCH request
	Patch(ctx context.Context, url string, body interface{}, headers map[string]string) (*Response, error)

	// Delete performs a DELETE request
	Delete(ctx context.Context, url string, headers map[string]string) (*Response, error)

	// Request performs a custom HTTP request
	Request(ctx context.Context, req *Request) (*Response, error)
}

// Request represents an HTTP request
type Request struct {
	// Method is the HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
	Method string

	// URL is the request URL (can be relative to BaseURL or absolute)
	URL string

	// Headers are custom headers for this request
	Headers map[string]string

	// Body is the request body
	// Can be: string, []byte, io.Reader, map[string]interface{}, or nil
	Body interface{}

	// ContentType specifies the content type (default: "application/json")
	ContentType string

	// QueryParams are URL query parameters
	QueryParams map[string]string
}

// Response represents an HTTP response
type Response struct {
	// StatusCode is the HTTP status code
	StatusCode int

	// Status is the HTTP status text
	Status string

	// Headers are the response headers
	Headers map[string]string

	// Body is the response body as bytes
	Body []byte

	// BodyString is the response body as string
	BodyString string

	// JSON unmarshals the response body as JSON
	JSON interface{}
}

// BodyReader returns the response body as an io.Reader
func (r *Response) BodyReader() io.Reader {
	if r.Body == nil {
		return nil
	}
	return &bodyReader{data: r.Body, pos: 0}
}

type bodyReader struct {
	data []byte
	pos  int
}

func (br *bodyReader) Read(p []byte) (n int, err error) {
	if br.pos >= len(br.data) {
		return 0, io.EOF
	}
	n = copy(p, br.data[br.pos:])
	br.pos += n
	return n, nil
}
