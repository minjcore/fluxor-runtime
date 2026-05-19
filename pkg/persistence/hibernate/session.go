package hibernate

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/fluxorio/fluxor/pkg/cache"
	"github.com/fluxorio/fluxor/pkg/core/failfast"
	"github.com/fluxorio/fluxor/pkg/persistence"
)

// Session represents a Hibernate-like session for entity management
// Similar to Hibernate Session interface
type Session interface {
	// Get retrieves an entity by ID (like session.get())
	Get(ctx context.Context, entityType interface{}, id interface{}) (interface{}, error)

	// Load retrieves an entity by ID (like session.load() - lazy)
	Load(ctx context.Context, entityType interface{}, id interface{}) (interface{}, error)

	// Save persists a new entity (like session.save())
	Save(ctx context.Context, entity interface{}) error

	// Update updates an existing entity (like session.update())
	Update(ctx context.Context, entity interface{}) error

	// SaveOrUpdate saves or updates an entity (like session.saveOrUpdate())
	SaveOrUpdate(ctx context.Context, entity interface{}) error

	// Delete removes an entity (like session.delete())
	Delete(ctx context.Context, entity interface{}) error

	// Flush flushes the session (like session.flush())
	Flush(ctx context.Context) error

	// Clear clears the session cache (like session.clear())
	Clear()

	// Evict evicts an entity from cache (like session.evict())
	Evict(entity interface{})

	// CreateQuery creates an HQL query (like session.createQuery())
	CreateQuery(hql string) Query

	// CreateCriteria creates a Criteria query (like session.createCriteria())
	CreateCriteria(entityType interface{}) Criteria

	// BeginTransaction begins a transaction (like session.beginTransaction())
	BeginTransaction(ctx context.Context) (Transaction, error)

	// Close closes the session (like session.close())
	Close() error
}

// SessionFactory creates and manages sessions
// Similar to Hibernate SessionFactory
type SessionFactory interface {
	// OpenSession opens a new session
	OpenSession(ctx context.Context) (Session, error)

	// GetCurrentSession gets the current session (if using session context)
	GetCurrentSession(ctx context.Context) (Session, error)

	// Close closes the session factory
	Close() error
}

// EntityState represents the state of an entity
type EntityState int

const (
	// Transient - entity not associated with session
	Transient EntityState = iota
	// Persistent - entity associated with session and in database
	Persistent
	// Detached - entity was persistent but session closed
	Detached
	// Removed - entity marked for deletion
	Removed
)

// entityMetadata tracks entity state and metadata
type entityMetadata struct {
	entity     interface{}
	state      EntityState
	original   map[string]interface{} // For dirty checking
	id         interface{}
	entityType reflect.Type
	tableName  string
}

// hibernateSession implements Session interface
type hibernateSession struct {
	ctx            context.Context
	repository     persistence.Repository
	firstLevelCache map[string]*entityMetadata // First-level cache (session cache)
	secondLevelCache cache.Cache                // Second-level cache (optional)
	mu              sync.RWMutex
	dirtyEntities   map[string]interface{} // Entities that need flushing
	closed          bool
}

// NewSession creates a new Hibernate-like session
func NewSession(ctx context.Context, repository persistence.Repository, secondLevelCache cache.Cache) Session {
	return &hibernateSession{
		ctx:              ctx,
		repository:       repository,
		firstLevelCache: make(map[string]*entityMetadata),
		secondLevelCache: secondLevelCache,
		dirtyEntities:   make(map[string]interface{}),
		closed:          false,
	}
}

// Get retrieves an entity by ID (like session.get())
func (s *hibernateSession) Get(ctx context.Context, entityType interface{}, id interface{}) (interface{}, error) {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(entityType, "entityType")
	failfast.NotNil(id, "id")

	if s.closed {
		return nil, fmt.Errorf("session is closed")
	}

	// Check first-level cache
	cacheKey := s.getCacheKey(entityType, id)
	s.mu.RLock()
	if metadata, exists := s.firstLevelCache[cacheKey]; exists {
		s.mu.RUnlock()
		return metadata.entity, nil
	}
	s.mu.RUnlock()

	// Check second-level cache (thread-safe - cache.Get/Set are thread-safe)
	var cachedEntity interface{}
	if s.secondLevelCache != nil {
		if cached, err := s.secondLevelCache.Get(ctx, cacheKey); err == nil {
			var entity interface{}
			if err := json.Unmarshal(cached, &entity); err == nil {
				cachedEntity = entity
			}
		}
	}

	// If found in second-level cache, store in first-level cache and return
	if cachedEntity != nil {
		s.mu.Lock()
		s.firstLevelCache[cacheKey] = &entityMetadata{
			entity:     cachedEntity,
			state:      Persistent,
			id:         id,
			entityType: reflect.TypeOf(entityType),
		}
		s.mu.Unlock()
		return cachedEntity, nil
	}

	// Load from database
	entity, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store in first-level cache
	s.mu.Lock()
	s.firstLevelCache[cacheKey] = &entityMetadata{
		entity:     entity,
		state:      Persistent,
		original:   s.copyEntity(entity),
		id:         id,
		entityType: reflect.TypeOf(entityType),
	}
	s.mu.Unlock()

	// Store in second-level cache (thread-safe - cache.Set is thread-safe)
	if s.secondLevelCache != nil {
		if data, err := json.Marshal(entity); err == nil {
			// Note: cache.Set is thread-safe, no mutex needed
			s.secondLevelCache.Set(ctx, cacheKey, data, 0)
		}
	}

	return entity, nil
}

// Load retrieves an entity by ID (lazy loading - like session.load())
func (s *hibernateSession) Load(ctx context.Context, entityType interface{}, id interface{}) (interface{}, error) {
	// For now, Load is same as Get (lazy loading would require proxy objects)
	return s.Get(ctx, entityType, id)
}

// Save persists a new entity (like session.save())
func (s *hibernateSession) Save(ctx context.Context, entity interface{}) error {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(entity, "entity")

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	// Convert entity to map
	entityMap := s.entityToMap(entity)

	// Save to database
	if err := s.repository.Create(ctx, entityMap); err != nil {
		return err
	}

	// Get ID from result (would need to query back or get from insert result)
	// For now, mark as dirty to be flushed
	cacheKey := s.getEntityCacheKey(entity)
	s.mu.Lock()
	s.dirtyEntities[cacheKey] = entity
	s.firstLevelCache[cacheKey] = &entityMetadata{
		entity:     entity,
		state:      Persistent,
		original:   s.copyEntity(entity),
		entityType: reflect.TypeOf(entity),
	}
	s.mu.Unlock()

	return nil
}

// Update updates an existing entity (like session.update())
func (s *hibernateSession) Update(ctx context.Context, entity interface{}) error {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(entity, "entity")

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	// Get ID from entity
	id := s.getEntityID(entity)
	if id == nil {
		return fmt.Errorf("entity must have an ID field")
	}

	// Convert entity to map
	entityMap := s.entityToMap(entity)

	// Update in database
	if err := s.repository.Update(ctx, id, entityMap); err != nil {
		return err
	}

	// Update cache
	cacheKey := s.getEntityCacheKey(entity)
	s.mu.Lock()
	s.dirtyEntities[cacheKey] = entity
	if metadata, exists := s.firstLevelCache[cacheKey]; exists {
		metadata.state = Persistent
		metadata.original = s.copyEntity(entity)
	}
	s.mu.Unlock()

	// Invalidate second-level cache (thread-safe - cache.Delete is thread-safe)
	if s.secondLevelCache != nil {
		// Note: cache.Delete is thread-safe, no mutex needed
		s.secondLevelCache.Delete(ctx, cacheKey)
	}

	return nil
}

// SaveOrUpdate saves or updates an entity (like session.saveOrUpdate())
func (s *hibernateSession) SaveOrUpdate(ctx context.Context, entity interface{}) error {
	id := s.getEntityID(entity)
	if id == nil || (reflect.ValueOf(id).IsZero()) {
		return s.Save(ctx, entity)
	}
	return s.Update(ctx, entity)
}

// Delete removes an entity (like session.delete())
func (s *hibernateSession) Delete(ctx context.Context, entity interface{}) error {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(entity, "entity")

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	id := s.getEntityID(entity)
	if id == nil {
		return fmt.Errorf("entity must have an ID field")
	}

	// Delete from database
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}

	// Remove from cache
	cacheKey := s.getEntityCacheKey(entity)
	s.mu.Lock()
	delete(s.firstLevelCache, cacheKey)
	delete(s.dirtyEntities, cacheKey)
	s.mu.Unlock()

	// Invalidate second-level cache (thread-safe - cache.Delete is thread-safe)
	if s.secondLevelCache != nil {
		// Note: cache.Delete is thread-safe, no mutex needed
		s.secondLevelCache.Delete(ctx, cacheKey)
	}

	return nil
}

// Flush flushes the session (like session.flush())
func (s *hibernateSession) Flush(ctx context.Context) error {
	if s.closed {
		return fmt.Errorf("session is closed")
	}

	s.mu.RLock()
	dirty := make([]interface{}, 0, len(s.dirtyEntities))
	for _, entity := range s.dirtyEntities {
		dirty = append(dirty, entity)
	}
	s.mu.RUnlock()

	// Flush dirty entities
	for _, entity := range dirty {
		id := s.getEntityID(entity)
		if id != nil && !reflect.ValueOf(id).IsZero() {
			if err := s.Update(ctx, entity); err != nil {
				return err
			}
		} else {
			if err := s.Save(ctx, entity); err != nil {
				return err
			}
		}
	}

	// Clear dirty entities
	s.mu.Lock()
	s.dirtyEntities = make(map[string]interface{})
	s.mu.Unlock()

	return nil
}

// Clear clears the session cache (like session.clear())
func (s *hibernateSession) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.firstLevelCache = make(map[string]*entityMetadata)
	s.dirtyEntities = make(map[string]interface{})
}

// Evict evicts an entity from cache (like session.evict())
func (s *hibernateSession) Evict(entity interface{}) {
	cacheKey := s.getEntityCacheKey(entity)
	s.mu.Lock()
	delete(s.firstLevelCache, cacheKey)
	delete(s.dirtyEntities, cacheKey)
	s.mu.Unlock()
}

// CreateQuery creates an HQL query (like session.createQuery())
func (s *hibernateSession) CreateQuery(hql string) Query {
	return newHQLQuery(s, hql)
}

// CreateCriteria creates a Criteria query (like session.createCriteria())
func (s *hibernateSession) CreateCriteria(entityType interface{}) Criteria {
	return newCriteria(s, entityType)
}

// BeginTransaction begins a transaction (like session.beginTransaction())
func (s *hibernateSession) BeginTransaction(ctx context.Context) (Transaction, error) {
	if s.closed {
		return nil, fmt.Errorf("session is closed")
	}
	tx, err := s.repository.BeginTransaction(ctx)
	if err != nil {
		return nil, err
	}
	return &hibernateTransaction{tx: tx}, nil
}

// Close closes the session (like session.close())
func (s *hibernateSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.firstLevelCache = nil
	s.dirtyEntities = nil
	return nil
}

// Helper methods

func (s *hibernateSession) getCacheKey(entityType interface{}, id interface{}) string {
	return fmt.Sprintf("%s:%v", reflect.TypeOf(entityType).String(), id)
}

func (s *hibernateSession) getEntityCacheKey(entity interface{}) string {
	id := s.getEntityID(entity)
	entityType := reflect.TypeOf(entity)
	return fmt.Sprintf("%s:%v", entityType.String(), id)
}

func (s *hibernateSession) getEntityID(entity interface{}) interface{} {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Try common ID field names
	idFields := []string{"ID", "Id", "id"}
	for _, fieldName := range idFields {
		field := v.FieldByName(fieldName)
		if field.IsValid() && field.CanInterface() {
			return field.Interface()
		}
	}

	return nil
}

func (s *hibernateSession) entityToMap(entity interface{}) map[string]interface{} {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	result := make(map[string]interface{})
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Get field name from tag or use struct field name
		fieldName := fieldType.Name
		if dbTag := fieldType.Tag.Get("db"); dbTag != "" {
			fieldName = dbTag
		}

		if field.IsValid() && field.CanInterface() {
			result[fieldName] = field.Interface()
		}
	}

	return result
}

func (s *hibernateSession) copyEntity(entity interface{}) map[string]interface{} {
	return s.entityToMap(entity)
}
