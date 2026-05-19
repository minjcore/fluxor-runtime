package webhook

import (
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
	"github.com/fluxorio/fluxor/pkg/web"
)

// FastHTTPMiddleware creates a FastHTTP middleware for webhook handling
// This middleware checks if the request path matches a webhook endpoint
// If it matches, it handles the webhook and returns early
// If it doesn't match, it passes to the next handler
func FastHTTPMiddleware(receiver *Receiver) web.FastMiddleware {
	failfast.NotNil(receiver, "receiver")

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			path := string(ctx.Path())

			// Check if this is a webhook endpoint
			_, isWebhook := receiver.GetEndpoint(path)
			if !isWebhook {
				// Not a webhook request, pass to next handler
				return next(ctx)
			}

			// Parse webhook request
			req, err := parseWebhookRequest(ctx, path)
			if err != nil {
				ctx.RequestCtx.SetStatusCode(400)
				ctx.RequestCtx.SetContentType("application/json")
				errorMsg := fmt.Sprintf(`{"error":"parse_error","message":"%s"}`, err.Error())
				ctx.RequestCtx.WriteString(errorMsg)
				return nil // Error already handled
			}

			// Handle webhook request
			if err := receiver.HandleRequest(req); err != nil {
				ctx.RequestCtx.SetStatusCode(400)
				ctx.RequestCtx.SetContentType("application/json")
				errorMsg := fmt.Sprintf(`{"error":"webhook_error","message":"%s"}`, err.Error())
				ctx.RequestCtx.WriteString(errorMsg)
				return nil // Error already handled
			}

			// Return success response
			ctx.RequestCtx.SetStatusCode(200)
			ctx.RequestCtx.SetContentType("application/json")
			ctx.RequestCtx.WriteString(`{"status":"ok"}`)
			return nil
		}
	}
}

// RegisterRoutes registers webhook routes on a FastRouter
func RegisterRoutes(router *web.FastRouter, receiver *Receiver) error {
	failfast.NotNil(router, "router")
	failfast.NotNil(receiver, "receiver")

	// Get all endpoint paths from receiver
	endpointPaths := receiver.GetEndpoints()

	// Create handler function
	handler := func(ctx *web.FastRequestContext) error {
		path := string(ctx.Path())

		// Parse webhook request
		req, err := parseWebhookRequest(ctx, path)
		if err != nil {
			ctx.RequestCtx.SetStatusCode(400)
			ctx.RequestCtx.SetContentType("application/json")
			errorMsg := fmt.Sprintf(`{"error":"parse_error","message":"%s"}`, err.Error())
			ctx.RequestCtx.WriteString(errorMsg)
			return nil
		}

		// Handle webhook request
		if err := receiver.HandleRequest(req); err != nil {
			ctx.RequestCtx.SetStatusCode(400)
			ctx.RequestCtx.SetContentType("application/json")
			errorMsg := fmt.Sprintf(`{"error":"webhook_error","message":"%s"}`, err.Error())
			ctx.RequestCtx.WriteString(errorMsg)
			return nil
		}

		// Return success response
		ctx.RequestCtx.SetStatusCode(200)
		ctx.RequestCtx.SetContentType("application/json")
		ctx.RequestCtx.WriteString(`{"status":"ok"}`)
		return nil
	}

	// Register POST routes for all endpoints
	for _, path := range endpointPaths {
		router.POSTFast(path, handler)
	}

	return nil
}

// parseWebhookRequest parses a webhook request from FastRequestContext
func parseWebhookRequest(ctx *web.FastRequestContext, path string) (*WebhookRequest, error) {
	// Extract headers
	headers := make(map[string]string)
	ctx.RequestCtx.Request.Header.VisitAll(func(key, value []byte) {
		headers[strings.ToLower(string(key))] = string(value)
	})

	// Extract query parameters
	queryParams := make(map[string]string)
	ctx.RequestCtx.QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = string(value)
	})

	// Get request body
	payload := ctx.RequestCtx.PostBody()

	// Normalize headers to original case for signature validation
	originalHeaders := make(map[string]string)
	ctx.RequestCtx.Request.Header.VisitAll(func(key, value []byte) {
		originalHeaders[string(key)] = string(value)
	})

	return &WebhookRequest{
		Path:        path,
		Payload:     payload,
		Headers:     originalHeaders,
		QueryParams: queryParams,
		Method:      string(ctx.Method()),
	}, nil
}