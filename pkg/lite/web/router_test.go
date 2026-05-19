package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fluxorio/fluxor/pkg/lite/core"
	"github.com/fluxorio/fluxor/pkg/lite/fx"
	"github.com/fluxorio/fluxor/pkg/lite/web"
)

func TestRouter_Handle_NotFound(t *testing.T) {
	r := web.NewRouter()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/missing", nil)
	rec := httptest.NewRecorder()
	ctx := fx.NewContext(rec, req, core.NewFluxorContext(req.Context(), core.NewBus(), core.NewWorkerPool(1, 10), "test"))

	if err := r.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if rec.Code != 404 {
		t.Fatalf("status=%d, want 404", rec.Code)
	}
}

func TestRouter_Handle_PathParams(t *testing.T) {
	r := web.NewRouter()
	r.GET("/users/:id", func(c *fx.Context) error {
		return c.Text(200, c.Param("id"))
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/123", nil)
	rec := httptest.NewRecorder()
	ctx := fx.NewContext(rec, req, core.NewFluxorContext(req.Context(), core.NewBus(), core.NewWorkerPool(1, 10), "test"))

	if err := r.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if rec.Code != 200 {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "123" {
		t.Fatalf("body=%q, want %q", got, "123")
	}
}

func TestRouter_MiddlewareOrder(t *testing.T) {
	r := web.NewRouter()

	var order []string
	r.Use(func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *fx.Context) error {
			order = append(order, "global-before")
			err := next(c)
			order = append(order, "global-after")
			return err
		}
	})

	r.GET("/x", func(c *fx.Context) error {
		order = append(order, "handler")
		return c.Text(200, "ok")
	}, func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *fx.Context) error {
			order = append(order, "route-before")
			err := next(c)
			order = append(order, "route-after")
			return err
		}
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	rec := httptest.NewRecorder()
	ctx := fx.NewContext(rec, req, core.NewFluxorContext(req.Context(), core.NewBus(), core.NewWorkerPool(1, 10), "test"))

	if err := r.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	want := []string{"global-before", "route-before", "handler", "route-after", "global-after"}
	if len(order) != len(want) {
		t.Fatalf("order=%v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order=%v, want %v", order, want)
		}
	}
}
