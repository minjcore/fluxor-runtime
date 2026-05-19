package auth_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func newInMemoryFastHTTP(t *testing.T, handler fasthttp.RequestHandler) (*fasthttp.Client, func()) {
	t.Helper()

	ln := fasthttputil.NewInmemoryListener()
	srv := &fasthttp.Server{Handler: handler}

	done := make(chan struct{})
	go func() {
		_ = srv.Serve(ln)
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

func TestJWTAuthMiddleware_Integration(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	defer gocmd.Close()

	server := web.NewFastHTTPServer(gocmd, web.DefaultFastHTTPServerConfig(":0"))
	router := server.FastRouter()

	secret := "test-secret"
	cfg := auth.DefaultJWTConfig(secret)

	// Protected route
	router.GETFastWith("/private", func(ctx *web.FastRequestContext) error {
		userID, err := auth.GetUserID(ctx, cfg.ClaimsKey)
		if err != nil {
			return ctx.JSON(500, map[string]any{"error": "missing_user"})
		}
		return ctx.JSON(200, map[string]any{"user_id": userID})
	}, auth.JWT(cfg))

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

	doGet := func(token string) (int, string) {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(resp)

		req.Header.SetMethod("GET")
		req.SetRequestURI("http://test/private")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		if err := client.Do(req, resp); err != nil {
			t.Fatalf("request failed: %v", err)
		}
		return resp.StatusCode(), string(resp.Body())
	}

	// Missing token -> 401
	if status, _ := doGet(""); status != 401 {
		t.Fatalf("missing token status=%d, want 401", status)
	}

	// Invalid token -> 401
	if status, _ := doGet("not-a-jwt"); status != 401 {
		t.Fatalf("invalid token status=%d, want 401", status)
	}

	// Valid token -> 200
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	status, body := doGet(tokenStr)
	if status != 200 {
		t.Fatalf("valid token status=%d, want 200 (body=%s)", status, body)
	}
	if body == "" {
		t.Fatalf("expected non-empty body")
	}
}
