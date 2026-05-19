# Append-Only Log Package

High-performance append-only log storage with rotation, durability guarantees, and event sourcing support.

## 🎯 Overview

The `appendlog` package provides a durable, append-only log store ideal for:

- ✅ **Event sourcing**: Immutable event streams
- ✅ **Audit logging**: Tamper-proof audit trails
- ✅ **Time-series data**: High-throughput writes
- ✅ **Message queues**: Persistent message storage
- ✅ **WAL (Write-Ahead Log)**: Database transaction logs

### Key Features

- **Append-only**: No in-place updates or deletes
- **Monotonic offsets**: Guaranteed ordering
- **Segment rotation**: Immutable sealed segments
- **Durability control**: Memory or fsync modes
- **Backpressure**: Fail-fast when buffers full
- **Observer pattern**: Real-time event notifications
- **Thread-safe**: Concurrent append and read operations

---

## 📦 Core Concepts

### Offset

Monotonically increasing position within the log:

```go
type Offset uint64

// Examples:
offset1 := Offset(0)    // First record
offset2 := Offset(42)   // 43rd record
offset3 := Offset(1000) // 1001st record
```

### Record

Immutable log entry:

```go
type Record struct {
    Offset Offset  // Assigned by store
    Data   []byte  // Raw payload (caller-defined encoding)
}
```

### Durability

Acknowledgment guarantee:

```go
type Durability int

const (
    DurabilityMemory Durability = iota  // Acknowledge after in-memory
    DurabilityFsync                     // Acknowledge after fsync
)
```

---

## 🚀 Quick Start

### In-Memory Store (Fast, Volatile)

```go
import "github.com/fluxorio/fluxor/pkg/appendlog"

// Create in-memory store
store := appendlog.NewMemoryStore()
defer store.Close()

// Append records
offset1, err := store.Append([]byte("event 1"))
offset2, err := store.Append([]byte("event 2"))
offset3, err := store.Append([]byte("event 3"))

// Read records
records, err := store.Read(offset1, 10)
for _, rec := range records {
    fmt.Printf("Offset %d: %s\n", rec.Offset, string(rec.Data))
}

// Get stats
stats := store.Stats()
fmt.Printf("Total records: %d\n", stats.RecordCount)
```

### Filesystem Store (Durable, Persistent)

```go
// Create filesystem store
config := appendlog.FSStoreConfig{
    Dir:        "/var/log/myapp",
    Durability: appendlog.DurabilityFsync,  // Durable writes
    MaxSegmentSize: 64 * 1024 * 1024,       // 64 MB segments
}

store, err := appendlog.NewFSStore(config)
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// Append (fsync'd to disk)
offset, err := store.Append([]byte("important event"))

// Rotate to new segment
if err := store.Rotate(); err != nil {
    log.Fatal(err)
}

// Force sync
if err := store.Sync(); err != nil {
    log.Fatal(err)
}
```

---

## 📋 Store Interface

```go
type Store interface {
    // Append writes a record and returns its offset
    Append(data []byte) (Offset, error)
    
    // Read retrieves records starting from offset (up to limit)
    Read(from Offset, limit int) ([]Record, error)
    
    // Rotate seals current segment and creates new one
    Rotate() error
    
    // Sync forces flush to disk (for durability)
    Sync() error
    
    // Close closes the store and releases resources
    Close() error
    
    // Stats returns current statistics
    Stats() Stats
}
```

---

## 📊 Statistics

```go
type Stats struct {
    BufferedBytes int64  // In-memory queued bytes
    WrittenBytes  int64  // Total bytes written to disk
    RecordCount   int64  // Total records appended
    SegmentCount  int    // Number of segments (for FS store)
}

// Get stats
stats := store.Stats()
fmt.Printf("Buffered: %d bytes\n", stats.BufferedBytes)
fmt.Printf("Written: %d bytes\n", stats.WrittenBytes)
fmt.Printf("Records: %d\n", stats.RecordCount)
fmt.Printf("Segments: %d\n", stats.SegmentCount)
```

---

## 🔔 Observer Pattern

Subscribe to real-time append events:

```go
type Observer interface {
    OnAppend(offset Offset, data []byte)
}

// Create observer
type MyObserver struct{}

func (o *MyObserver) OnAppend(offset Offset, data []byte) {
    fmt.Printf("New record at offset %d: %s\n", offset, string(data))
}

// Register observer
observer := &MyObserver{}
store.RegisterObserver(observer)
defer store.UnregisterObserver(observer)

// Append triggers observer
offset, _ := store.Append([]byte("event"))
// Output: New record at offset 0: event
```

---

## 🎨 Usage Patterns

### Pattern 1: Event Sourcing

```go
type EventStore struct {
    log appendlog.Store
}

func (es *EventStore) AppendEvent(event Event) (Offset, error) {
    // Serialize event
    data, err := json.Marshal(event)
    if err != nil {
        return 0, err
    }
    
    // Append to log
    return es.log.Append(data)
}

func (es *EventStore) ReplayEvents(from Offset, handler func(Event) error) error {
    const batchSize = 100
    
    for {
        // Read batch
        records, err := es.log.Read(from, batchSize)
        if err != nil {
            return err
        }
        
        if len(records) == 0 {
            break
        }
        
        // Process events
        for _, rec := range records {
            var event Event
            if err := json.Unmarshal(rec.Data, &event); err != nil {
                return err
            }
            
            if err := handler(event); err != nil {
                return err
            }
            
            from = rec.Offset + 1
        }
        
        if len(records) < batchSize {
            break
        }
    }
    
    return nil
}
```

### Pattern 2: Audit Trail

```go
type AuditLog struct {
    log appendlog.Store
}

type AuditEntry struct {
    Timestamp time.Time
    UserID    string
    Action    string
    Resource  string
    Result    string
}

func (al *AuditLog) LogAction(entry AuditEntry) error {
    entry.Timestamp = time.Now()
    
    data, err := json.Marshal(entry)
    if err != nil {
        return err
    }
    
    _, err = al.log.Append(data)
    return err
}

func (al *AuditLog) QueryAudit(userID string, since time.Time) ([]AuditEntry, error) {
    var results []AuditEntry
    
    records, err := al.log.Read(0, 10000)  // Read all
    if err != nil {
        return nil, err
    }
    
    for _, rec := range records {
        var entry AuditEntry
        if err := json.Unmarshal(rec.Data, &entry); err != nil {
            continue
        }
        
        if entry.UserID == userID && entry.Timestamp.After(since) {
            results = append(results, entry)
        }
    }
    
    return results, nil
}
```

### Pattern 3: Message Queue

```go
type MessageQueue struct {
    log    appendlog.Store
    offset Offset  // Consumer offset
}

func (mq *MessageQueue) Enqueue(message []byte) error {
    _, err := mq.log.Append(message)
    return err
}

func (mq *MessageQueue) Dequeue() ([]byte, error) {
    records, err := mq.log.Read(mq.offset, 1)
    if err != nil {
        return nil, err
    }
    
    if len(records) == 0 {
        return nil, io.EOF  // No more messages
    }
    
    mq.offset = records[0].Offset + 1
    return records[0].Data, nil
}

func (mq *MessageQueue) DequeueBatch(size int) ([][]byte, error) {
    records, err := mq.log.Read(mq.offset, size)
    if err != nil {
        return nil, err
    }
    
    if len(records) == 0 {
        return nil, io.EOF
    }
    
    messages := make([][]byte, len(records))
    for i, rec := range records {
        messages[i] = rec.Data
    }
    
    mq.offset = records[len(records)-1].Offset + 1
    return messages, nil
}
```

### Pattern 4: Time-Series Data

```go
type TimeSeriesLog struct {
    log appendlog.Store
}

type DataPoint struct {
    Timestamp int64
    Metric    string
    Value     float64
    Tags      map[string]string
}

func (ts *TimeSeriesLog) WritePoint(point DataPoint) error {
    point.Timestamp = time.Now().UnixNano()
    
    data, err := json.Marshal(point)
    if err != nil {
        return err
    }
    
    _, err = ts.log.Append(data)
    return err
}

func (ts *TimeSeriesLog) ReadRange(start, end int64) ([]DataPoint, error) {
    var points []DataPoint
    
    records, err := ts.log.Read(0, 100000)  // Read chunk
    if err != nil {
        return nil, err
    }
    
    for _, rec := range records {
        var point DataPoint
        if err := json.Unmarshal(rec.Data, &point); err != nil {
            continue
        }
        
        if point.Timestamp >= start && point.Timestamp <= end {
            points = append(points, point)
        }
    }
    
    return points, nil
}
```

---

## 🔧 Configuration

### FSStoreConfig

```go
type FSStoreConfig struct {
    Dir            string      // Storage directory
    Durability     Durability  // Memory or Fsync
    MaxSegmentSize int64       // Max segment size in bytes
}

// Default configuration
config := appendlog.FSStoreConfig{
    Dir:            "./data",
    Durability:     appendlog.DurabilityFsync,
    MaxSegmentSize: 64 * 1024 * 1024,  // 64 MB
}
```

### Durability Modes

| Mode | Throughput | Durability | Use Case |
|------|-----------|------------|----------|
| **DurabilityMemory** | ⚡ Very High | ⚠️ Volatile | Development, non-critical logs |
| **DurabilityFsync** | 🚀 Good | ✅ Durable | Production, critical data |

---

## ⚡ Performance Considerations

### Write Performance

```go
// Batch writes for better throughput
batch := [][]byte{
    []byte("event 1"),
    []byte("event 2"),
    []byte("event 3"),
}

for _, data := range batch {
    store.Append(data)
}

// Sync once after batch (DurabilityMemory mode)
store.Sync()
```

### Read Performance

```go
// Read in batches
const batchSize = 1000
offset := Offset(0)

for {
    records, err := store.Read(offset, batchSize)
    if err != nil || len(records) == 0 {
        break
    }
    
    // Process batch
    for _, rec := range records {
        process(rec)
    }
    
    offset = records[len(records)-1].Offset + 1
}
```

### Segment Rotation

```go
// Rotate periodically for manageable segment sizes
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    
    for range ticker.C {
        if err := store.Rotate(); err != nil {
            log.Printf("Rotation failed: %v", err)
        }
    }
}()
```

---

## 🛡️ Error Handling

### Backpressure

```go
// Append fails fast when buffers full
offset, err := store.Append(data)
if err != nil {
    if errors.Is(err, appendlog.ErrBufferFull) {
        // Apply backpressure (wait or reject)
        time.Sleep(100 * time.Millisecond)
        // Retry or return error
    }
}
```

### Read Errors

```go
records, err := store.Read(offset, limit)
if err != nil {
    if errors.Is(err, io.EOF) {
        // No more records
    } else if errors.Is(err, appendlog.ErrInvalidOffset) {
        // Offset out of range
    } else {
        // Other error
    }
}
```

---

## 🧪 Testing

### Unit Tests

```bash
go test ./pkg/appendlog/...
```

### Integration Tests

```go
func TestAppendLog(t *testing.T) {
    store := appendlog.NewMemoryStore()
    defer store.Close()
    
    // Append records
    offset1, err := store.Append([]byte("test 1"))
    require.NoError(t, err)
    require.Equal(t, Offset(0), offset1)
    
    offset2, err := store.Append([]byte("test 2"))
    require.NoError(t, err)
    require.Equal(t, Offset(1), offset2)
    
    // Read records
    records, err := store.Read(0, 10)
    require.NoError(t, err)
    require.Len(t, records, 2)
    assert.Equal(t, "test 1", string(records[0].Data))
    assert.Equal(t, "test 2", string(records[1].Data))
}
```

---

## 🆚 Comparison

### vs Database

| Feature | appendlog | Database |
|---------|-----------|----------|
| **Write Speed** | ⚡ Very Fast | 🚀 Good |
| **Random Access** | ❌ Sequential only | ✅ Yes |
| **Updates** | ❌ Append-only | ✅ Yes |
| **Deletes** | ❌ No | ✅ Yes |
| **Queries** | ❌ Manual scan | ✅ Indexed |
| **Use Case** | Event logs, audit | General purpose |

### vs Message Queue (Kafka, NATS)

| Feature | appendlog | Kafka | NATS Streaming |
|---------|-----------|-------|----------------|
| **Complexity** | ✅ Simple | ⚠️ Complex | 🚀 Medium |
| **Durability** | ✅ Yes | ✅ Yes | ✅ Yes |
| **Distributed** | ❌ Single-node | ✅ Yes | ✅ Yes |
| **Throughput** | ⚡ Very High (local) | 🚀 High | 🚀 High |
| **Use Case** | Embedded, local | Distributed | Distributed |

---

## 🎯 Best Practices

### 1. Choose Right Durability Mode

```go
// Development / non-critical
config.Durability = appendlog.DurabilityMemory

// Production / critical data
config.Durability = appendlog.DurabilityFsync
```

### 2. Rotate Segments Regularly

```go
// Rotate daily or when segment reaches size limit
if stats.WrittenBytes > config.MaxSegmentSize {
    store.Rotate()
}
```

### 3. Batch Reads for Efficiency

```go
// ✅ Good: Batch reads
records, _ := store.Read(offset, 1000)

// ❌ Bad: One-by-one reads
for i := 0; i < 1000; i++ {
    records, _ := store.Read(Offset(i), 1)
}
```

### 4. Monitor Buffer Size

```go
stats := store.Stats()
if stats.BufferedBytes > 10*1024*1024 {
    log.Warn("High buffer size, consider sync")
    store.Sync()
}
```

### 5. Use Observers for Real-Time Processing

```go
// Real-time indexing
store.RegisterObserver(&IndexObserver{index: index})

// Real-time replication
store.RegisterObserver(&ReplicationObserver{remote: remote})
```

---

## 📚 Related Documentation

- [DOCUMENTATION.md](../../DOCUMENTATION.md) - Complete API reference
- [examples/appendlog](../../examples/appendlog) - Complete examples (if available)
- [PKG_DECISION_GUIDE.md](../../PKG_DECISION_GUIDE.md) - Package selection guide

---

## 🔗 Integration Examples

### With EventBus

```go
// Persist EventBus messages
consumer := eventBus.Consumer("important.events")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Append to log
    data, _ := json.Marshal(msg.Body())
    _, err := store.Append(data)
    return err
})
```

### With Workflow Engine

```go
// Audit workflow executions
engine.OnWorkflowComplete(func(workflowID string, result interface{}) {
    entry := WorkflowAuditEntry{
        WorkflowID: workflowID,
        Timestamp:  time.Now(),
        Result:     result,
    }
    
    data, _ := json.Marshal(entry)
    store.Append(data)
})
```

---

**Package**: `github.com/fluxorio/fluxor/pkg/appendlog`  
**Status**: ✅ Stable (C+ Grade)  
**Test Coverage**: 75%  
**Last Updated**: 2026-01-04
