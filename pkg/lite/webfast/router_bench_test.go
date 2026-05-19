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

func BenchmarkLiteFast_Ping_InMemory(b *testing.B) {
	coreCtx := core.NewFluxorContext(context.Background(), core.NewBus(), core.NewWorkerPool(1, 1024), "bench")

	r := webfast.NewRouter()
	r.Bind(coreCtx)

	r.GET("/ping", func(c *fx.FastContext) error {
		// minimal response (best case for throughput)
		return c.Text(200, "pong")
	})

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

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod("GET")
	req.SetRequestURI("http://test/ping")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp.Reset()
		if err := client.Do(req, resp); err != nil {
			b.Fatal(err)
		}
	}
}
