package appendlog

import "io"

// Offset is a monotonically increasing position within a stream.
type Offset uint64

// Durability specifies when Append is acknowledged.
type Durability int

const (
	// DurabilityMemory acknowledges after the record is accepted into memory.
	DurabilityMemory Durability = iota
	// DurabilityFsync acknowledges after the active segment is fsync'd.
	// (Stronger durability, lower throughput.)
	DurabilityFsync
)

// Record is an append-only payload.
// The store treats Data as immutable.
type Record struct {
	// Offset assigned by the store.
	Offset Offset
	// Data is the raw payload (caller-defined encoding).
	Data []byte
}

// Store is an append-only log store with optional disk persistence.
//
// Contract summary:
// - Append-only: no in-place updates/deletes.
// - Offsets are monotonically increasing per store.
// - Rotation seals immutable segments; new writes go to a new segment.
// - Backpressure: Append must fail-fast when buffers are full.
type Store interface {
	Append(data []byte) (Offset, error)
	Read(from Offset, limit int) ([]Record, error)
	Rotate() error
	Sync() error
	Close() error
	Stats() Stats
}

// Stats exposes basic operational counters.
type Stats struct {
	// Current in-memory queued bytes awaiting flush.
	BufferedBytes int64
	// Total bytes written to disk (best-effort).
	WrittenBytes int64
	// Total number of records appended.
	AppendedRecords int64
	// Total rejected appends due to backpressure.
	RejectedAppends int64
}

// Observer receives events for observability (metrics/logging/tracing).
//
// Contract:
// - Implementations must be fast and must not block indefinitely.
// - Observer calls are best-effort; they must never affect correctness.
// - An observer should be thread-safe.
type Observer interface {
	OnRecover(info RecoverInfo)
	OnAppendEnqueued(info AppendInfo)
	OnAppendPersisted(info PersistInfo)
	OnAppendRejected(info RejectInfo)
	OnRotate(info RotateInfo)
}

type RecoverInfo struct {
	Dir       string
	Segments  int
	MaxOffset Offset
}

type AppendInfo struct {
	Offset Offset
	Bytes  int
}

type PersistInfo struct {
	Offset   Offset
	Bytes    int
	Duration int64 // nanoseconds
	Err      error
}

type RejectInfo struct {
	Bytes  int
	Reason error
}

type RotateInfo struct {
	FromSegment int
	ToSegment   int
	Reason      string
}

// NopObserver is the default observer (no-op).
type NopObserver struct{}

func (NopObserver) OnRecover(RecoverInfo)         {}
func (NopObserver) OnAppendEnqueued(AppendInfo)   {}
func (NopObserver) OnAppendPersisted(PersistInfo) {}
func (NopObserver) OnAppendRejected(RejectInfo)   {}
func (NopObserver) OnRotate(RotateInfo)           {}

// Errors.
var (
	ErrClosed         = io.ErrClosedPipe
	ErrInvalidData    = io.ErrUnexpectedEOF
	ErrBackpressure   = io.ErrShortWrite
	ErrInvalidReadArg = io.ErrNoProgress
)
