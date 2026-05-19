package buckey_storage

import (
	"bytes"
	"context"
	"fmt"
)

// DefaultReplicas is the default replication factor on one server.
const DefaultReplicas = 3

// ErrDivergent is returned when replicas disagree and no majority can be determined.
var ErrDivergent = fmt.Errorf("buckey_storage: replicas divergent, no majority")

// ReplicatedStorage stores each blob N times (replicas) on one server using
// in-memory stores. Compatible with in-memory semantics; use for durability
// on a single node (e.g. tolerate one bad write by reading from another copy).
type ReplicatedStorage struct {
	replicas []*MemoryStorage
	cfg      *Config
}

// NewReplicated creates a blob storage that keeps Replicas copies on one server.
// If cfg.Replicas is 0, DefaultReplicas (3) is used.
func NewReplicated(cfg *Config) *ReplicatedStorage {
	if cfg == nil {
		c := DefaultConfig()
		cfg = &c
	}
	n := cfg.Replicas
	if n <= 0 {
		n = DefaultReplicas
	}
	replicas := make([]*MemoryStorage, n)
	for i := 0; i < n; i++ {
		replicas[i] = NewMemory(cfg)
	}
	return &ReplicatedStorage{replicas: replicas, cfg: cfg}
}

// Put writes data to all replicas.
func (r *ReplicatedStorage) Put(ctx context.Context, key string, data []byte) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	var firstErr error
	for _, rep := range r.replicas {
		if err := rep.Put(ctx, key, data); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// Get reads from replicas; returns first successful result. If all fail, returns last error or ErrNotFound.
func (r *ReplicatedStorage) Get(ctx context.Context, key string) ([]byte, error) {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return nil, err
	}
	var lastErr error
	for _, rep := range r.replicas {
		b, err := rep.Get(ctx, key)
		if err == nil {
			return b, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrNotFound
}

// Delete removes key from all replicas.
func (r *ReplicatedStorage) Delete(ctx context.Context, key string) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	for _, rep := range r.replicas {
		_ = rep.Delete(ctx, key)
	}
	return nil
}

// List returns keys with prefix from the first replica (all replicas share same keys after Put/Delete).
func (r *ReplicatedStorage) List(ctx context.Context, prefix string) ([]string, error) {
	ValidateContext(ctx)
	if len(r.replicas) == 0 {
		return nil, nil
	}
	return r.replicas[0].List(ctx, prefix)
}

// GetQuorum reads from at least minReads replicas and returns the majority value.
// If replicas disagree, the majority value is returned and minority replicas are repaired (overwritten).
// If no majority exists (e.g. 3 different values), returns ErrDivergent.
// minReads must be at least 1; typical: (len(replicas)/2)+1 for quorum.
func (r *ReplicatedStorage) GetQuorum(ctx context.Context, key string, minReads int) ([]byte, error) {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return nil, err
	}
	if minReads < 1 {
		minReads = 1
	}
	var values [][]byte
	for _, rep := range r.replicas {
		b, err := rep.Get(ctx, key)
		if err == nil {
			values = append(values, b)
		}
	}
	if len(values) < minReads {
		if len(values) == 0 {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("buckey_storage: only %d replicas had key, need %d", len(values), minReads)
	}
	// Find majority value (by content): group by bytes.Equal via first occurrence
	type contentKey int
	contentToBlob := make(map[contentKey][]byte)
	blobToContent := make(map[string]contentKey) // key = string(blob) for lookup
	var nextKey contentKey
	var contentCounts map[contentKey]int
	for _, b := range values {
		s := string(b)
		if k, ok := blobToContent[s]; ok {
			contentCounts[k]++
			continue
		}
		k := nextKey
		nextKey++
		blobToContent[s] = k
		contentToBlob[k] = b
		if contentCounts == nil {
			contentCounts = make(map[contentKey]int)
		}
		contentCounts[k] = 1
	}
	var bestKey contentKey
	var bestCount int
	for k, n := range contentCounts {
		if n > bestCount {
			bestCount = n
			bestKey = k
		}
	}
	if bestCount == 0 {
		return nil, ErrDivergent
	}
	majority := contentToBlob[bestKey]
	// Repair: write majority to any replica that had a different value
	for _, rep := range r.replicas {
		b, err := rep.Get(ctx, key)
		if err != nil || !bytes.Equal(b, majority) {
			_ = rep.Put(ctx, key, majority)
		}
	}
	return append([]byte(nil), majority...), nil
}

// RepairKey forces all replicas to the same value by reading majority and writing back.
// Returns the majority value or ErrDivergent if no majority.
func (r *ReplicatedStorage) RepairKey(ctx context.Context, key string) ([]byte, error) {
	return r.GetQuorum(ctx, key, 1)
}
