package webfast_test

import (
	"context"
	"net"
	"testing"

	"github.com/fluxorio/fluxor/pkg/lite/core"
	"github.com/fluxorio/fluxor/pkg/lite/fx"
	"github.com/fluxorio/fluxor/pkg/lite/webfast"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestCacheMiddleware_ETag304(t *testing.T) {
	coreCtx := core.NewFluxorContext(context.Background(), core.NewBus(), core.NewWorkerPool(1, 1024), "test")

	r := webfast.NewRouter()
	r.Bind(coreCtx)

	r.GET("/ping", func(c *fx.FastContext) error {
		return c.Text(200, "pong")
	}, webfast.Cache(webfast.CacheConfig{
		CacheControl: "public, max-age=60, immutable",
		ETag:         `"ping-v1"`,
	}))

	ln := fasthttputil.NewInmemoryListener()
	srv := &fasthttp.Server{Handler: r.Handler()}
	go func() { _ = srv.Serve(ln) }()
	defer func() {
		_ = ln.Close()
		_ = srv.Shutdown()
	}()

	client := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) { return ln.Dial() },
	}

	do := func(ifNoneMatch string) (int, string, string, []byte) {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(resp)

		req.Header.SetMethod("GET")
		req.SetRequestURI("http://test/ping")
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}

		if err := client.Do(req, resp); err != nil {
			t.Fatalf("request failed: %v", err)
		}
		return resp.StatusCode(),
			string(resp.Header.Peek("Cache-Control")),
			string(resp.Header.Peek("ETag")),
			append([]byte(nil), resp.Body()...)
	}

	code, cc, etag, body := do("")
	if code != 200 {
		t.Fatalf("status=%d, want 200", code)
	}
	if cc == "" || etag == "" {
		t.Fatalf("expected cache headers set, got cache-control=%q etag=%q", cc, etag)
	}
	if string(body) != "pong" {
		t.Fatalf("body=%q, want %q", string(body), "pong")
	}

	code, _, etag2, body2 := do(`"ping-v1"`)
	if code != 304 {
		t.Fatalf("status=%d, want 304", code)
	}
	if etag2 == "" {
		t.Fatalf("expected ETag on 304 response")
	}
	if len(body2) != 0 {
		t.Fatalf("expected empty body for 304, got %q", string(body2))
	}
}
