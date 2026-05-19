package appendlog

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// FSStoreConfig configures the file-backed append-only store.
type FSStoreConfig struct {
	Dir string

	// MaxSegmentBytes triggers rotation when the active segment reaches this size.
	MaxSegmentBytes int64

	// MaxBufferedBytes bounds in-memory buffering. When exceeded, Append fails-fast.
	MaxBufferedBytes int64

	// Durability controls when Append is acknowledged.
	Durability Durability

	// Observer receives best-effort events for observability.
	// If nil, a no-op observer is used.
	Observer Observer
}

// DefaultFSStoreConfig returns a conservative default config.
func DefaultFSStoreConfig(dir string) FSStoreConfig {
	return FSStoreConfig{
		Dir:              dir,
		MaxSegmentBytes:  64 << 20, // 64MB
		MaxBufferedBytes: 8 << 20,  // 8MB
		Durability:       DurabilityMemory,
		Observer:         NopObserver{},
	}
}

// NewFSStore creates an append-only store backed by segment files in dir.
func NewFSStore(cfg FSStoreConfig) (Store, error) {
	if strings.TrimSpace(cfg.Dir) == "" {
		return nil, fmt.Errorf("dir is required")
	}

	// Validate directory path to prevent directory traversal attacks
	if err := validateDirPath(cfg.Dir); err != nil {
		return nil, fmt.Errorf("invalid directory path: %w", err)
	}

	if cfg.MaxSegmentBytes <= 0 {
		cfg.MaxSegmentBytes = 64 << 20
	}
	if cfg.MaxBufferedBytes <= 0 {
		cfg.MaxBufferedBytes = 8 << 20
	}
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, err
	}

	s := &fsStore{
		cfg: cfg,
	}
	if s.cfg.Observer == nil {
		s.cfg.Observer = NopObserver{}
	}
	if err := s.openOrRecover(); err != nil {
		_ = s.Close()
		return nil, err
	}

	// Start background flusher.
	s.flushWg.Add(1)
	go s.flushLoop()

	return s, nil
}

type appendReq struct {
	offset Offset
	data   []byte
	ackCh  chan error
}

// fsStore implements Store with:
// - in-memory-first append path (buffered channel)
// - background flush to disk segment files
// - rotation by size
type fsStore struct {
	cfg FSStoreConfig

	mu     sync.RWMutex
	closed bool

	nextOffset uint64 // atomic

	// active segment state (protected by mu)
	activeID   int
	activeFile *os.File
	activeBuf  *bufio.Writer
	activeSize int64

	// in-memory queue
	appendCh chan appendReq
	flushWg  sync.WaitGroup

	// stats
	bufferedBytes   int64
	writtenBytes    int64
	appendedRecords int64
	rejectedAppends int64
}

func (s *fsStore) openOrRecover() error {
	// Discover existing segments and compute next offset + activeID.
	segments, err := listSegments(s.cfg.Dir)
	if err != nil {
		return err
	}
	var maxOffset Offset
	var maxID int
	for _, seg := range segments {
		if seg.id > maxID {
			maxID = seg.id
		}
		off, err := scanSegmentMaxOffset(seg.path)
		if err != nil {
			return err
		}
		if off > maxOffset {
			maxOffset = off
		}
	}
	atomic.StoreUint64(&s.nextOffset, uint64(maxOffset+1))
	s.cfg.Observer.OnRecover(RecoverInfo{
		Dir:       s.cfg.Dir,
		Segments:  len(segments),
		MaxOffset: maxOffset,
	})

	// Open active segment (existing last, or new).
	s.activeID = maxID
	if s.activeID == 0 && len(segments) == 0 {
		s.activeID = 1
	}
	path := segmentPath(s.cfg.Dir, s.activeID)
	// #nosec G304 -- path is constructed from validated cfg.Dir (validated in NewFSStore)
	// and integer segment ID using filepath.Join, preventing directory traversal
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	s.activeFile = f
	s.activeSize = st.Size()
	s.activeBuf = bufio.NewWriterSize(f, 256<<10) // 256KB buffer

	// queue capacity: enough to cover max buffered bytes with small records
	s.appendCh = make(chan appendReq, 1024)
	return nil
}

func (s *fsStore) Append(data []byte) (Offset, error) {
	if len(data) == 0 {
		s.cfg.Observer.OnAppendRejected(RejectInfo{Bytes: 0, Reason: ErrInvalidData})
		return 0, ErrInvalidData
	}

	// Check closed state and capture channel atomically
	s.mu.RLock()
	closed := s.closed
	ch := s.appendCh
	s.mu.RUnlock()

	if closed || ch == nil {
		s.cfg.Observer.OnAppendRejected(RejectInfo{Bytes: len(data), Reason: ErrClosed})
		return 0, ErrClosed
	}

	// Fail-fast backpressure based on total buffered bytes.
	size := int64(len(data))
	for {
		cur := atomic.LoadInt64(&s.bufferedBytes)
		if cur+size > s.cfg.MaxBufferedBytes {
			atomic.AddInt64(&s.rejectedAppends, 1)
			s.cfg.Observer.OnAppendRejected(RejectInfo{Bytes: len(data), Reason: ErrBackpressure})
			return 0, ErrBackpressure
		}
		if atomic.CompareAndSwapInt64(&s.bufferedBytes, cur, cur+size) {
			break
		}
	}

	offset := Offset(atomic.AddUint64(&s.nextOffset, 1) - 1)
	ackCh := make(chan error, 1)

	req := appendReq{
		offset: offset,
		data:   append([]byte(nil), data...), // copy
		ackCh:  ackCh,
	}

	// Non-blocking enqueue: if channel is full, fail-fast.
	// Use captured channel to avoid race with Close()
	select {
	case ch <- req:
		atomic.AddInt64(&s.appendedRecords, 1)
		s.cfg.Observer.OnAppendEnqueued(AppendInfo{Offset: offset, Bytes: len(req.data)})
	default:
		atomic.AddInt64(&s.rejectedAppends, 1)
		atomic.AddInt64(&s.bufferedBytes, -size)
		s.cfg.Observer.OnAppendRejected(RejectInfo{Bytes: len(data), Reason: ErrBackpressure})
		return 0, ErrBackpressure
	}

	// Ack policy.
	if s.cfg.Durability == DurabilityMemory {
		return offset, nil
	}
	return offset, <-ackCh
}

func (s *fsStore) Read(from Offset, limit int) ([]Record, error) {
	if limit <= 0 {
		return nil, ErrInvalidReadArg
	}

	// Snapshot directory segments (read-only path).
	segs, err := listSegments(s.cfg.Dir)
	if err != nil {
		return nil, err
	}

	out := make([]Record, 0, min(limit, 128))
	for _, seg := range segs {
		recs, err := readSegmentRange(seg.path, from, limit-len(out))
		if err != nil {
			return nil, err
		}
		out = append(out, recs...)
		if len(out) >= limit {
			break
		}
		if len(recs) > 0 {
			from = recs[len(recs)-1].Offset + 1
		}
	}
	return out, nil
}

func (s *fsStore) Rotate() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	from := s.activeID
	if err := s.rotateLocked(); err != nil {
		return err
	}
	s.cfg.Observer.OnRotate(RotateInfo{
		FromSegment: from,
		ToSegment:   s.activeID,
		Reason:      "manual",
	})
	return nil
}

func (s *fsStore) rotateLocked() error {
	if s.activeBuf != nil {
		_ = s.activeBuf.Flush()
	}
	if s.cfg.Durability == DurabilityFsync && s.activeFile != nil {
		_ = s.activeFile.Sync()
	}
	if s.activeFile != nil {
		_ = s.activeFile.Close()
	}

	s.activeID++
	path := segmentPath(s.cfg.Dir, s.activeID)
	// #nosec G304 -- path is constructed from validated cfg.Dir (validated in NewFSStore)
	// and integer segment ID using filepath.Join, preventing directory traversal
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	s.activeFile = f
	s.activeBuf = bufio.NewWriterSize(f, 256<<10)
	s.activeSize = 0
	return nil
}

func (s *fsStore) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.activeBuf != nil {
		if err := s.activeBuf.Flush(); err != nil {
			return err
		}
	}
	if s.activeFile != nil {
		return s.activeFile.Sync()
	}
	return nil
}

func (s *fsStore) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	ch := s.appendCh
	s.appendCh = nil
	s.mu.Unlock()

	// Close channel to signal flushLoop to stop
	if ch != nil {
		close(ch)
	}

	// Wait for flushLoop to finish processing remaining requests
	s.flushWg.Wait()

	// Now safe to flush and close files (no concurrent access)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeBuf != nil {
		_ = s.activeBuf.Flush()
	}
	if s.activeFile != nil {
		_ = s.activeFile.Close()
	}
	return nil
}

func (s *fsStore) Stats() Stats {
	return Stats{
		BufferedBytes:   atomic.LoadInt64(&s.bufferedBytes),
		WrittenBytes:    atomic.LoadInt64(&s.writtenBytes),
		AppendedRecords: atomic.LoadInt64(&s.appendedRecords),
		RejectedAppends: atomic.LoadInt64(&s.rejectedAppends),
	}
}

func (s *fsStore) flushLoop() {
	defer s.flushWg.Done()

	// Capture appendCh once to avoid race condition with Close()
	// We hold the reference to the original channel even if Close() sets s.appendCh to nil
	s.mu.RLock()
	ch := s.appendCh
	s.mu.RUnlock()

	if ch == nil {
		return
	}

	// Drain append requests and persist them in order.
	// Loop will exit when channel is closed by Close()
	for req := range ch {
		start := time.Now()
		err := s.appendToDisk(req.offset, req.data)
		s.cfg.Observer.OnAppendPersisted(PersistInfo{
			Offset:   req.offset,
			Bytes:    len(req.data),
			Duration: time.Since(start).Nanoseconds(),
			Err:      err,
		})
		atomic.AddInt64(&s.bufferedBytes, -int64(len(req.data)))
		if s.cfg.Durability == DurabilityFsync {
			// Send ack after persisting (non-blocking with buffered channel)
			select {
			case req.ackCh <- err:
			default:
				// Channel full or closed - best effort
			}
		}
	}
}

func (s *fsStore) appendToDisk(offset Offset, data []byte) error {
	// Record format (little endian):
	// [offset u64][len u32][data bytes]
	// Validate data size to prevent integer overflow when converting to uint32
	if len(data) > math.MaxUint32 {
		return fmt.Errorf("data size %d exceeds maximum uint32 value %d: %w", len(data), math.MaxUint32, ErrInvalidData)
	}
	var hdr [12]byte
	binary.LittleEndian.PutUint64(hdr[0:8], uint64(offset))
	binary.LittleEndian.PutUint32(hdr[8:12], uint32(len(data)))

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}

	// Rotate if needed.
	if s.activeSize+int64(len(hdr))+int64(len(data)) > s.cfg.MaxSegmentBytes {
		from := s.activeID
		if err := s.rotateLocked(); err != nil {
			return err
		}
		s.cfg.Observer.OnRotate(RotateInfo{
			FromSegment: from,
			ToSegment:   s.activeID,
			Reason:      "max_segment_bytes",
		})
	}

	if _, err := s.activeBuf.Write(hdr[:]); err != nil {
		return err
	}
	if _, err := s.activeBuf.Write(data); err != nil {
		return err
	}
	s.activeSize += int64(len(hdr) + len(data))
	atomic.AddInt64(&s.writtenBytes, int64(len(hdr)+len(data)))

	// Flush in background frequently to reduce loss window.
	// (Still "in-memory-first": we are buffering in bufio.)
	if s.activeBuf.Buffered() > 256<<10 {
		if err := s.activeBuf.Flush(); err != nil {
			return err
		}
	}
	if s.cfg.Durability == DurabilityFsync {
		if err := s.activeBuf.Flush(); err != nil {
			return err
		}
		return s.activeFile.Sync()
	}
	return nil
}

type segInfo struct {
	id   int
	path string
}

func segmentPath(dir string, id int) string {
	return filepath.Join(dir, fmt.Sprintf("%06d.log", id))
}

func listSegments(dir string) ([]segInfo, error) {
	// #nosec G304 -- dir comes from validated cfg.Dir (validated in NewFSStore via validateDirPath)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var segs []segInfo
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".log") {
			continue
		}
		base := strings.TrimSuffix(name, ".log")
		id, err := strconv.Atoi(base)
		if err != nil {
			continue
		}
		segs = append(segs, segInfo{id: id, path: filepath.Join(dir, name)})
	}
	sort.Slice(segs, func(i, j int) bool { return segs[i].id < segs[j].id })
	return segs, nil
}

func scanSegmentMaxOffset(path string) (Offset, error) {
	// #nosec G304 -- path is constructed from validated cfg.Dir and filename from directory listing
	// using filepath.Join, preventing directory traversal
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var max Offset
	for {
		var hdr [12]byte
		_, err := io.ReadFull(f, hdr[:])
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return max, nil
			}
			return 0, err
		}
		off := Offset(binary.LittleEndian.Uint64(hdr[0:8]))
		n := binary.LittleEndian.Uint32(hdr[8:12])
		if _, err := io.CopyN(io.Discard, f, int64(n)); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return max, nil
			}
			return 0, err
		}
		if off > max {
			max = off
		}
	}
}

func readSegmentRange(path string, from Offset, limit int) ([]Record, error) {
	if limit <= 0 {
		return nil, nil
	}
	// #nosec G304 -- path is constructed from validated cfg.Dir and filename from directory listing
	// using filepath.Join, preventing directory traversal
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := make([]Record, 0, min(limit, 128))
	for len(out) < limit {
		var hdr [12]byte
		_, err := io.ReadFull(f, hdr[:])
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return out, nil
			}
			return nil, err
		}
		off := Offset(binary.LittleEndian.Uint64(hdr[0:8]))
		n := binary.LittleEndian.Uint32(hdr[8:12])

		data := make([]byte, n)
		if _, err := io.ReadFull(f, data); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return out, nil
			}
			return nil, err
		}
		if off < from {
			continue
		}
		out = append(out, Record{Offset: off, Data: data})
	}
	return out, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// validateDirPath validates the directory path to prevent directory traversal attacks.
func validateDirPath(dir string) error {
	if dir == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for directory traversal sequences in the original path
	if strings.Contains(dir, "..") {
		return fmt.Errorf("path contains directory traversal sequence")
	}

	// Resolve absolute path to normalize it
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Clean the path to resolve any remaining ".." or "." components
	cleanPath := filepath.Clean(absPath)

	// After cleaning, if the path still contains "..", it indicates
	// an attempt to traverse outside the filesystem root
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains directory traversal sequence")
	}

	return nil
}

// Compile-time interface assertion.
var _ Store = (*fsStore)(nil)
