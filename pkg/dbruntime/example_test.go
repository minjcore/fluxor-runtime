package dbruntime_test

import (
	"context"
	"database/sql"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
)

// ExampleNewPool demonstrates creating a connection pool (HikariCP-like)
func ExampleNewPool() {
	// Create pool configuration (similar to HikariConfig)
	config := dbruntime.DefaultPoolConfig(
		"postgres://user:pass@localhost/dbname",
		"postgres",
	)

	// Create pool (similar to HikariDataSource)
	pool, err := dbruntime.NewPool(config)
	if err != nil {
		// Handle error
		return
	}
	defer pool.Close()

	// Use pool (connections are automatically managed)
	ctx := context.Background()
	rows, err := pool.Query(ctx, "SELECT id, name FROM users")
	if err != nil {
		// Handle error
		return
	}
	defer rows.Close()

	// Process rows
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			// Handle error
			return
		}
		// Use id and name
		_ = id
		_ = name
	}
}

// ExampleDatabaseComponent demonstrates using DatabaseComponent with Premium Pattern
func ExampleDatabaseComponent() {
	// Create database component
	config := dbruntime.DefaultPoolConfig(
		"postgres://user:pass@localhost/dbname",
		"postgres",
	)
	component := dbruntime.NewDatabaseComponent(config)

	// In a verticle's doStart method:
	// Note: In real usage, FluxorContext is provided by the framework
	// This is just an example - actual usage would be in a verticle's doStart method
	// Example:
	// func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
	//     component.SetParent(v.BaseVerticle)
	//     if err := component.Start(ctx); err != nil {
	//         return err
	//     }
	//     // Then use component
	//     rows, err := component.Query(ctx.Context(), "SELECT * FROM users")
	//     ...
	// }

	_ = component
}

// ExampleDatabaseComponent_inService demonstrates using DatabaseComponent in a service
func ExampleDatabaseComponent_inService() {
	// Service with database component
	type UserService struct {
		*core.BaseService
		db *dbruntime.DatabaseComponent
	}

	// Create service
	service := &UserService{
		BaseService: core.NewBaseService("user-service", "user.service"),
		db: dbruntime.NewDatabaseComponent(
			dbruntime.DefaultPoolConfig(
				"postgres://user:pass@localhost/dbname",
				"postgres",
			),
		),
	}

	// In doStart:
	doStart := func(ctx core.FluxorContext) error {
		service.db.SetParent(service.BaseVerticle)
		return service.db.Start(ctx)
	}

	// In doHandleRequest:
	doHandleRequest := func(ctx core.FluxorContext, msg core.Message) error {
		userID := msg.Body().(string)

		var name string
		err := service.db.QueryRow(
			context.Background(), // Use context from FluxorContext: ctx.Context()
			"SELECT name FROM users WHERE id = $1",
			userID,
		).Scan(&name)

		if err != nil {
			if err == sql.ErrNoRows {
				return service.Fail(msg, 404, "User not found")
			}
			return service.Fail(msg, 500, err.Error())
		}

		return service.Reply(msg, map[string]interface{}{
			"id":   userID,
			"name": name,
		})
	}

	_ = doStart
	_ = doHandleRequest
}

// ExamplePool_Stats demonstrates monitoring pool statistics (like HikariPoolMXBean)
func ExamplePool_Stats() {
	config := dbruntime.DefaultPoolConfig(
		"postgres://user:pass@localhost/dbname",
		"postgres",
	)
	pool, _ := dbruntime.NewPool(config)
	defer pool.Close()

	// Get pool statistics
	stats := pool.Stats()

	// Monitor pool health
	_ = stats.OpenConnections   // Current open connections
	_ = stats.InUse             // Connections in use
	_ = stats.Idle              // Idle connections
	_ = stats.WaitCount         // Number of connections waiting
	_ = stats.WaitDuration      // Total time waiting for connections
	_ = stats.MaxIdleClosed     // Connections closed due to MaxIdleConns
	_ = stats.MaxIdleTimeClosed // Connections closed due to MaxIdleTime
	_ = stats.MaxLifetimeClosed // Connections closed due to MaxLifetime
}
