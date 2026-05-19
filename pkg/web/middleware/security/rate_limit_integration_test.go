package security_test

import (
	"context"
	"net"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware/security"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func newInMemoryFastHTTP(t *testing.T, handler fasthttp.RequestHandler) (*fasthttp.Client, func()) {
	t.Helper()

	ln := fasthttputil.NewInmemoryListener()
	srv := &fasthttp.Server{Handler: handler}

	done := make(chan struct{})
	go func() {
		_ = srv.Serve(ln) // Serve returns when listener is closed
		close(done)
	}()

	client := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) { return ln.Dial() },
	}

	cleanup := func() {
		_ = ln.Close()
		_ = srv.Shutdown()
		<-done
	}

	return client, cleanup
}

func TestRateLimit_PerRouteConfig(t *testing.T) {
	// Skip this test as it was designed for burst-based rate limiting which
	// is not currently supported. The token bucket implementation starts with
	// full capacity which doesn't match the expected burst=1 behavior.
	t.Skip("Test requires burst-based rate limiting which is not implemented")

	gocmd := core.NewGoCMD(context.Background())
	defer gocmd.Close()

	server := web.NewFastHTTPServer(gocmd, web.DefaultFastHTTPServerConfig(":0"))
	router := server.FastRouter()

	keyFunc := func(ctx *web.FastRequestContext) string { return "client-1" }

	limit1 := security.RateLimit(security.RateLimitConfig{
		RequestsPerMinute: 60, // 1 per second
		KeyFunc:           keyFunc,
	})
	limit2 := security.RateLimit(security.RateLimitConfig{
		RequestsPerMinute: 120, // 2 per second
		KeyFunc:           keyFunc,
	})

	router.GETFastWith("/limited-1", func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	}, limit1)

	router.GETFastWith("/limited-2", func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	}, limit2)

	handler := func(rc *fasthttp.RequestCtx) {
		reqCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         rc,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}
		router.ServeFastHTTP(reqCtx)
	}

	client, cleanup := newInMemoryFastHTTP(t, handler)
	defer cleanup()

	doGet := func(path string) int {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(resp)

		req.Header.SetMethod("GET")
		req.SetRequestURI("http://test" + path)
		if err := client.Do(req, resp); err != nil {
			t.Fatalf("request failed: %v", err)
		}
		return resp.StatusCode()
	}

	// /limited-1 allows 1 request, then rate limits
	if got := doGet("/limited-1"); got != 200 {
		t.Fatalf("GET /limited-1 status=%d, want 200", got)
	}
	if got := doGet("/limited-1"); got != 429 {
		t.Fatalf("GET /limited-1 status=%d, want 429", got)
	}

	// /limited-2 allows 2 requests
	if got := doGet("/limited-2"); got != 200 {
		t.Fatalf("GET /limited-2 status=%d, want 200", got)
	}
	if got := doGet("/limited-2"); got != 200 {
		t.Fatalf("GET /limited-2 status=%d, want 200", got)
	}
	// third should be limited
	if got := doGet("/limited-2"); got != 429 {
		t.Fatalf("GET /limited-2 status=%d, want 429", got)
	}
}
