package hibernate

import (
	"context"
	"fmt"
	"sync"

	"github.com/fluxorio/fluxor/pkg/cache"
	"github.com/fluxorio/fluxor/pkg/persistence"
)

// SessionFactoryConfig configures a session factory
type SessionFactoryConfig struct {
	// Repository is the base repository for persistence
	Repository persistence.Repository

	// SecondLevelCache is optional second-level cache
	SecondLevelCache cache.Cache

	// EnableSecondLevelCache enables second-level caching
	EnableSecondLevelCache bool
}

// hibernateSessionFactory implements SessionFactory
type hibernateSessionFactory struct {
	config        SessionFactoryConfig
	mu            sync.RWMutex
	closed        bool
	sessionContext map[context.Context]Session // For GetCurrentSession
}

// NewSessionFactory creates a new Hibernate-like session factory
func NewSessionFactory(config SessionFactoryConfig) SessionFactory {
	return &hibernateSessionFactory{
		config:         config,
		sessionContext: make(map[context.Context]Session),
		closed:         false,
	}
}

// OpenSession opens a new session
func (sf *hibernateSessionFactory) OpenSession(ctx context.Context) (Session, error) {
	sf.mu.RLock()
	if sf.closed {
		sf.mu.RUnlock()
		return nil, &Error{Code: "FACTORY_CLOSED", Message: "session factory is closed"}
	}
	sf.mu.RUnlock()

	var secondLevelCache cache.Cache
	if sf.config.EnableSecondLevelCache && sf.config.SecondLevelCache != nil {
		secondLevelCache = sf.config.SecondLevelCache
	}

	session := NewSession(ctx, sf.config.Repository, secondLevelCache)
	return session, nil
}

// GetCurrentSession gets the current session (if using session context)
func (sf *hibernateSessionFactory) GetCurrentSession(ctx context.Context) (Session, error) {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	if session, exists := sf.sessionContext[ctx]; exists {
		return session, nil
	}

	// Create new session and store in context
	session, err := sf.OpenSession(ctx)
	if err != nil {
		return nil, err
	}

	sf.sessionContext[ctx] = session
	return session, nil
}

// Close closes the session factory
func (sf *hibernateSessionFactory) Close() error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	sf.closed = true

	// Close all sessions
	for _, session := range sf.sessionContext {
		session.Close()
	}
	sf.sessionContext = nil

	return nil
}

// Error represents a Hibernate error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("hibernate error [%s]: %s", e.Code, e.Message)
}
