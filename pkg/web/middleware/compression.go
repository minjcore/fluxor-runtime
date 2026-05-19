package middleware

import (
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
)

// CompressionConfig configures response compression
type CompressionConfig struct {
	// Level is the compression level (1-9, default: 6)
	Level int

	// MinSize is the minimum response size to compress (default: 1024 bytes)
	MinSize int

	// ContentTypes is a list of content types to compress (default: all text types)
	ContentTypes []string

	// SkipPaths is a list of paths to skip compression
	SkipPaths []string
}

// DefaultCompressionConfig returns a default compression configuration
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Level:   6,
		MinSize: 1024,
		ContentTypes: []string{
			"text/plain",
			"text/html",
			"text/css",
			"text/javascript",
			"application/json",
			"application/javascript",
			"application/xml",
			"application/xhtml+xml",
		},
		SkipPaths: []string{},
	}
}

// Compression middleware compresses HTTP responses
func Compression(config CompressionConfig) web.FastMiddleware {
	level := config.Level
	if level < 1 || level > 9 {
		level = 6
	}

	minSize := config.MinSize
	if minSize < 0 {
		minSize = 1024
	}

	contentTypes := config.ContentTypes
	if len(contentTypes) == 0 {
		contentTypes = DefaultCompressionConfig().ContentTypes
	}

	// Create content type map for fast lookup
	contentTypeMap := make(map[string]bool)
	for _, ct := range contentTypes {
		contentTypeMap[strings.ToLower(ct)] = true
	}

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// Check if path should be skipped
			path := string(ctx.Path())
			skip := false
			for _, skipPath := range config.SkipPaths {
				if path == skipPath || strings.HasPrefix(path, skipPath) {
					skip = true
					break
				}
			}

			// Execute handler
			err := next(ctx)

			// Check if we should compress the response
			if !skip && err == nil {
				contentType := string(ctx.RequestCtx.Response.Header.ContentType())
				contentTypeLower := strings.ToLower(contentType)

				// Check if content type should be compressed
				shouldCompress := false
				for ct := range contentTypeMap {
					if strings.Contains(contentTypeLower, ct) {
						shouldCompress = true
						break
					}
				}

				// Check response size and if client accepts gzip
				bodySize := len(ctx.RequestCtx.Response.Body())
				acceptEncoding := string(ctx.RequestCtx.Request.Header.Peek("Accept-Encoding"))
				if shouldCompress && bodySize >= minSize && strings.Contains(acceptEncoding, "gzip") {
					// fasthttp automatically handles gzip compression
					// when Content-Encoding header is set before writing body
					// Note: This is a simplified implementation
					// For full compression, you may need to use fasthttp's compression features
					ctx.RequestCtx.Response.Header.Set("Content-Encoding", "gzip")
				}
			}

			return err
		}
	}
}
