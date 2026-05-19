package health

import (
	"context"
	"database/sql"
	"time"

	"github.com/fluxorio/fluxor/pkg/dbruntime"
)

// DatabaseCheck creates a health check for a database pool
func DatabaseCheck(pool *dbruntime.Pool) Checker {
	return func(ctx context.Context) error {
		if pool == nil {
			return &Error{Message: "database pool is nil"}
		}

		// Use a short timeout for health checks
		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if err := pool.Ping(checkCtx); err != nil {
			return &Error{Message: "database ping failed: " + err.Error()}
		}

		return nil
	}
}

// DatabaseComponentCheck creates a health check for a database component
func DatabaseComponentCheck(component *dbruntime.DatabaseComponent) Checker {
	return func(ctx context.Context) error {
		if component == nil {
			return &Error{Message: "database component is nil"}
		}

		pool := component.Pool()
		if pool == nil {
			return &Error{Message: "database pool is nil"}
		}

		return DatabaseCheck(pool)(ctx)
	}
}

// SQLDBCheck creates a health check for a standard sql.DB
func SQLDBCheck(db *sql.DB) Checker {
	return func(ctx context.Context) error {
		if db == nil {
			return &Error{Message: "database is nil"}
		}

		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if err := db.PingContext(checkCtx); err != nil {
			return &Error{Message: "database ping failed: " + err.Error()}
		}

		return nil
	}
}

// Error represents a health check error
type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
