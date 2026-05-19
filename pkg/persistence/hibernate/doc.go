// Package hibernate provides Hibernate-like ORM functionality for Fluxor.
//
// The hibernate package provides:
//
//   - Session Management: Similar to Hibernate Session
//   - First-Level Cache: Session-scoped entity cache
//   - Second-Level Cache: Optional shared cache (using cache package)
//   - HQL Queries: Hibernate Query Language support
//   - Criteria API: Type-safe query building
//   - Entity Lifecycle: Transient, Persistent, Detached states
//   - Dirty Checking: Automatic change tracking
//   - Flush/Clear: Session management operations
//
// Example usage:
//
//	import (
//	    "context"
//	    "github.com/fluxorio/fluxor/pkg/persistence"
//	    "github.com/fluxorio/fluxor/pkg/persistence/hibernate"
//	    "github.com/fluxorio/fluxor/pkg/cache"
//	)
//
//	// Create repository
//	config := persistence.DefaultConfig("users", db)
//	repo, _ := persistence.NewSQLRepository(config)
//
//	// Create session factory
//	factory := hibernate.NewSessionFactory(hibernate.SessionFactoryConfig{
//	    Repository:            repo,
//	    EnableSecondLevelCache: true,
//	    SecondLevelCache:      cache.NewMemoryCache(),
//	})
//
//	// Open session
//	ctx := context.Background()
//	session, _ := factory.OpenSession(ctx)
//	defer session.Close()
//
//	// Save entity (like session.save())
//	user := &User{Name: "John", Email: "john@example.com"}
//	session.Save(ctx, user)
//
//	// Get entity (like session.get())
//	found, _ := session.Get(ctx, &User{}, 1)
//
//	// HQL query (like session.createQuery())
//	query := session.CreateQuery("FROM User WHERE status = :status")
//	query.SetParameter("status", "active")
//	users, _ := query.List(ctx)
//
//	// Criteria API (like session.createCriteria())
//	criteria := session.CreateCriteria(&User{})
//	criteria.Add(hibernate.Eq("status", "active"))
//	criteria.AddOrder(hibernate.Asc("name"))
//	results, _ := criteria.List(ctx)
package hibernate
