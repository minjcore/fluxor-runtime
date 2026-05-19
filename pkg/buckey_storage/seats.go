package buckey_storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// ErrNoSeat is returned when no seat is available (limit reached).
var ErrNoSeat = errors.New("buckey_storage: no seat available")

// SeatManager limits concurrent "seats" per scope (e.g. global or per-org).
// Uses BlobStorage to persist holders; safe for single-process use with in-memory or FS/S3.
type SeatManager struct {
	storage BlobStorage
	mu      sync.Mutex
}

// NewSeatManager creates a seat manager backed by the given storage.
func NewSeatManager(storage BlobStorage) *SeatManager {
	return &SeatManager{storage: storage}
}

const seatsKeyPrefix = "seats/"

func seatsKey(scope string) string {
	return seatsKeyPrefix + scope
}

// seatData is the stored value for a scope.
type seatData struct {
	Holders []string `json:"holders"`
}

// Acquire tries to acquire one seat for scope. holderID identifies who holds the seat (e.g. user or session ID).
// limit is the max seats for this scope. Returns ErrNoSeat if already at limit.
func (m *SeatManager) Acquire(ctx context.Context, scope, holderID string, limit int) error {
	ValidateContext(ctx)
	if scope == "" || holderID == "" {
		return fmt.Errorf("buckey_storage: scope and holderID cannot be empty")
	}
	if limit <= 0 {
		return fmt.Errorf("buckey_storage: limit must be positive")
	}
	key := seatsKey(scope)
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := m.storage.Get(ctx, key)
	var sd seatData
	if err == nil {
		_ = json.Unmarshal(data, &sd)
	}
	if sd.Holders == nil {
		sd.Holders = []string{}
	}
	for _, id := range sd.Holders {
		if id == holderID {
			return nil // already holding
		}
	}
	if len(sd.Holders) >= limit {
		return ErrNoSeat
	}
	sd.Holders = append(sd.Holders, holderID)
	raw, _ := json.Marshal(&sd)
	return m.storage.Put(ctx, key, raw)
}

// Release releases one seat for scope held by holderID.
func (m *SeatManager) Release(ctx context.Context, scope, holderID string) error {
	ValidateContext(ctx)
	if scope == "" || holderID == "" {
		return fmt.Errorf("buckey_storage: scope and holderID cannot be empty")
	}
	key := seatsKey(scope)
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := m.storage.Get(ctx, key)
	if err != nil {
		return nil // no data = nothing to release
	}
	var sd seatData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil
	}
	var next []string
	for _, id := range sd.Holders {
		if id != holderID {
			next = append(next, id)
		}
	}
	if len(next) == 0 {
		return m.storage.Delete(ctx, key)
	}
	sd.Holders = next
	raw, _ := json.Marshal(&sd)
	return m.storage.Put(ctx, key, raw)
}

// Usage returns the current number of holders and the limit for scope.
// limit is not stored; caller must pass the configured limit (e.g. 0 = unknown).
func (m *SeatManager) Usage(ctx context.Context, scope string) (used int, err error) {
	ValidateContext(ctx)
	if scope == "" {
		return 0, fmt.Errorf("buckey_storage: scope cannot be empty")
	}
	key := seatsKey(scope)
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := m.storage.Get(ctx, key)
	if err != nil {
		return 0, nil
	}
	var sd seatData
	if err := json.Unmarshal(data, &sd); err != nil {
		return 0, nil
	}
	return len(sd.Holders), nil
}
