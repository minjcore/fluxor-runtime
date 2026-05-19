// Package csql provides advanced SQL query building and execution utilities
// for the Fluxor persistence package.
//
// The csql package extends the base persistence package with:
//
//   - Fluent Query Builder: Build complex SQL queries with a fluent API
//   - Advanced SQL Features: JOINs, GROUP BY, HAVING, subqueries
//   - Raw SQL Execution: Execute custom SQL queries when needed
//   - Type-Safe Building: Compile-time query construction
//
// Example usage:
//
//	import (
//	    "context"
//	    "github.com/fluxorio/fluxor/pkg/persistence"
//	    "github.com/fluxorio/fluxor/pkg/persistence/csql"
//	)
//
//	config := persistence.DefaultConfig("users", db)
//	repo, _ := csql.NewCSQLRepository(config)
//
//	// Build query
//	qb := csql.NewQueryBuilder("users").
//	    Select("id", "name", "email").
//	    WhereEq("status", "active").
//	    OrderBy("created_at", "DESC").
//	    Limit(10)
//
//	// Execute
//	rows, _ := repo.Execute(ctx, qb)
//	defer rows.Close()
package csql
