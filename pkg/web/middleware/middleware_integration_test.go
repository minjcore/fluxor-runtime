package middleware_test

import (
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/web/middleware"
)

func TestLoggingMiddleware(t *testing.T) {
	config := middleware.DefaultLoggingConfig()
	if config.Logger == nil {
		t.Error("DefaultLoggingConfig should provide a logger")
	}

	mw := middleware.Logging(config)
	if mw == nil {
		t.Error("Logging should return middleware")
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	config := middleware.DefaultRecoveryConfig()
	if config.Logger == nil {
		t.Error("DefaultRecoveryConfig should provide a logger")
	}

	mw := middleware.Recovery(config)
	if mw == nil {
		t.Error("Recovery should return middleware")
	}
}

func TestCompressionMiddleware(t *testing.T) {
	config := middleware.DefaultCompressionConfig()
	if len(config.ContentTypes) == 0 {
		t.Error("DefaultCompressionConfig should provide content types")
	}

	mw := middleware.Compression(config)
	if mw == nil {
		t.Error("Compression should return middleware")
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	config := middleware.DefaultTimeoutConfig(5 * time.Second)
	if config.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", config.Timeout)
	}

	mw := middleware.Timeout(config)
	if mw == nil {
		t.Error("Timeout should return middleware")
	}
}
