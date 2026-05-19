package health_test

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/web/health"
)

func TestHealthCheckIntegration(t *testing.T) {
	// Register a simple health check
	health.Register("test", func(ctx context.Context) error {
		return nil
	})

	// Register a failing health check
	health.Register("failing", func(ctx context.Context) error {
		return &health.Error{Message: "test failure"}
	})

	// Run health checks
	results := health.Check(context.Background())

	if len(results) != 2 {
		t.Errorf("Expected 2 health checks, got %d", len(results))
	}

	if results["test"].Status != health.StatusUp {
		t.Errorf("test check should be UP, got %s", results["test"].Status)
	}

	if results["failing"].Status != health.StatusDown {
		t.Errorf("failing check should be DOWN, got %s", results["failing"].Status)
	}
}

func TestDatabaseHealthCheck(t *testing.T) {
	// This would require an actual database connection
	// For now, just test that the function exists and can be called
	checker := health.DatabaseCheck(nil)
	if checker == nil {
		t.Error("DatabaseCheck should return a checker function")
	}
}

func TestHTTPHealthCheck(t *testing.T) {
	// This would require an actual HTTP server
	// For now, just test that the function exists
	checker := health.HTTPCheck("http://localhost:8080/health", 5*time.Second)
	if checker == nil {
		t.Error("HTTPCheck should return a checker function")
	}
}
