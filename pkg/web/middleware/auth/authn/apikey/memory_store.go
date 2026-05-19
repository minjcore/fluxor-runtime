package apikey

import (
	"context"
	"fmt"
	"sync"
)

// MemoryStore is an in-memory implementation of Store for testing/development
type MemoryStore struct {
	mu      sync.RWMutex
	keys    map[string]*Key
	byHash  map[string]*Key // indexed by LookupHash
	byID    map[string]*Key // indexed by ID
}

// NewMemoryStore creates a new in-memory key store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		keys:   make(map[string]*Key),
		byHash: make(map[string]*Key),
		byID:   make(map[string]*Key),
	}
}

// Create creates a new API key
func (s *MemoryStore) Create(ctx context.Context, key *Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys[key.ID] = key
	s.byID[key.ID] = key
	if key.LookupHash != "" {
		s.byHash[key.LookupHash] = key
	}
	return nil
}

// GetByID retrieves a key by ID
func (s *MemoryStore) GetByID(ctx context.Context, id string) (*Key, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.byID[id]
	if !ok {
		return nil, ErrKeyNotFound
	}

	// Return a copy to prevent external modification
	keyCopy := *key
	return &keyCopy, nil
}

// GetByHash retrieves a key by its hash
func (s *MemoryStore) GetByHash(ctx context.Context, hash string) (*Key, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.byHash[hash]
	if !ok {
		return nil, ErrKeyNotFound
	}

	// Return a copy to prevent external modification
	keyCopy := *key
	return &keyCopy, nil
}

// ListByPrincipal lists all keys for a principal
func (s *MemoryStore) ListByPrincipal(ctx context.Context, principalID string) ([]*Key, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []*Key
	for _, key := range s.byID {
		if key.PrincipalID == principalID {
			keyCopy := *key
			keys = append(keys, &keyCopy)
		}
	}

	return keys, nil
}

// Update updates an existing key
func (s *MemoryStore) Update(ctx context.Context, key *Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byID[key.ID]; !ok {
		return ErrKeyNotFound
	}

	s.keys[key.ID] = key
	s.byID[key.ID] = key
	if key.LookupHash != "" {
		s.byHash[key.LookupHash] = key
	}

	return nil
}

// Delete deletes a key
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, ok := s.byID[id]
	if !ok {
		return ErrKeyNotFound
	}

	delete(s.keys, id)
	delete(s.byID, id)
	if key.LookupHash != "" {
		delete(s.byHash, key.LookupHash)
	}

	return nil
}

// Revoke revokes a key
func (s *MemoryStore) Revoke(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, ok := s.keys[id]
	if !ok {
		return ErrKeyNotFound
	}

	key.Revoked = true
	return nil
}

var (
	// ErrKeyNotFound is returned when a key is not found
	ErrKeyNotFound = fmt.Errorf("key not found")
)
