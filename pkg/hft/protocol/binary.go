package protocol

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"sync"
	"unsafe"
)

// Message types for HFT binary protocol
const (
	MsgTypeNewOrder     uint16 = 1
	MsgTypeCancelOrder  uint16 = 2
	MsgTypeOrderAck     uint16 = 3
	MsgTypeFill         uint16 = 4
	MsgTypeMarketData   uint16 = 5
	MsgTypeReject       uint16 = 6
	MsgTypeHeartbeat    uint16 = 7
	MsgTypeRiskViolation uint16 = 8
)

// Side represents order side
type Side uint8

const (
	SideBuy  Side = 0
	SideSell Side = 1
)

// OrderType represents order type
type OrderType uint8

const (
	OrderTypeLimit  OrderType = 0
	OrderTypeMarket OrderType = 1
	OrderTypeStop   OrderType = 2
)

// TimeInForce represents order TIF
type TimeInForce uint8

const (
	TIFGTC TimeInForce = 0 // Good Till Cancel
	TIFIOC TimeInForce = 1 // Immediate Or Cancel
	TIFFOK TimeInForce = 2 // Fill Or Kill
)

// OrderStatus represents order status
type OrderStatus uint8

const (
	StatusNew       OrderStatus = 0
	StatusPartial   OrderStatus = 1
	StatusFilled    OrderStatus = 2
	StatusCanceled  OrderStatus = 3
	StatusRejected  OrderStatus = 4
)

// Message header (24 bytes)
type MessageHeader struct {
	Length    uint32 // Message length (including header)
	Type      uint16 // Message type
	Flags     uint16 // Flags
	Timestamp int64  // Nanosecond timestamp
	SeqNum    uint64 // Sequence number
}

// NewOrderMessage (48 bytes total: 24 header + 48 payload + 4 checksum = 76 bytes)
type NewOrderMessage struct {
	Header        MessageHeader // 24 bytes
	OrderID       uint64        // 8 bytes
	Price         int64         // 8 bytes (fixed-point: price * 1e8)
	Quantity      int64         // 8 bytes
	SymbolID      uint32        // 4 bytes
	Side          Side          // 1 byte
	Type          OrderType     // 1 byte
	TIF           TimeInForce   // 1 byte
	Flags         uint8         // 1 byte
	ClientOrderID [16]byte      // 16 bytes
}

// CancelOrderMessage (24 bytes payload)
type CancelOrderMessage struct {
	Header        MessageHeader
	OrderID       uint64
	OrigOrderID   uint64
	ClientOrderID [16]byte
}

// OrderAckMessage (32 bytes payload)
type OrderAckMessage struct {
	Header        MessageHeader
	OrderID       uint64
	Status        OrderStatus
	_             [7]byte // Padding
	ClientOrderID [16]byte
}

// FillMessage (40 bytes payload)
type FillMessage struct {
	Header      MessageHeader
	OrderID     uint64
	FillPrice   int64
	FillQty     int64
	Leaves      int64
	Status      OrderStatus
	_           [7]byte // Padding
}

// MarketDataMessage (56 bytes payload)
type MarketDataMessage struct {
	Header      MessageHeader
	SymbolID    uint32
	_           [4]byte // Padding
	BidPrice    int64
	BidQty      int64
	AskPrice    int64
	AskQty      int64
	LastPrice   int64
	LastQty     int64
}

// RejectMessage (32 bytes payload)
type RejectMessage struct {
	Header        MessageHeader
	OrderID       uint64
	RejectCode    uint32
	_             [4]byte // Padding
	RejectReason  [16]byte
}

// Buffer pool for zero-allocation encoding/decoding
var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 1024)
		return &buf
	},
}

// GetBuffer gets a buffer from the pool
func GetBuffer() *[]byte {
	return bufferPool.Get().(*[]byte)
}

// PutBuffer returns a buffer to the pool
func PutBuffer(buf *[]byte) {
	bufferPool.Put(buf)
}

// MarshalBinary encodes NewOrderMessage to binary (zero-copy)
func (m *NewOrderMessage) MarshalBinary() []byte {
	buf := GetBuffer()
	b := (*buf)[:80] // 24 header + 48 payload + 4 checksum + 4 length

	// Header
	binary.LittleEndian.PutUint32(b[0:4], 76) // Total length
	binary.LittleEndian.PutUint16(b[4:6], MsgTypeNewOrder)
	binary.LittleEndian.PutUint16(b[6:8], m.Header.Flags)
	binary.LittleEndian.PutUint64(b[8:16], uint64(m.Header.Timestamp))
	binary.LittleEndian.PutUint64(b[16:24], m.Header.SeqNum)

	// Payload
	binary.LittleEndian.PutUint64(b[24:32], m.OrderID)
	binary.LittleEndian.PutUint64(b[32:40], uint64(m.Price))
	binary.LittleEndian.PutUint64(b[40:48], uint64(m.Quantity))
	binary.LittleEndian.PutUint32(b[48:52], m.SymbolID)
	b[52] = uint8(m.Side)
	b[53] = uint8(m.Type)
	b[54] = uint8(m.TIF)
	b[55] = m.Flags
	copy(b[56:72], m.ClientOrderID[:])

	// Checksum (CRC32 of header + payload)
	checksum := crc32.ChecksumIEEE(b[0:72])
	binary.LittleEndian.PutUint32(b[72:76], checksum)

	return b
}

// UnmarshalBinary decodes NewOrderMessage from binary (zero-copy)
func (m *NewOrderMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 76 {
		return errors.New("invalid message length")
	}

	// Verify checksum
	expectedChecksum := binary.LittleEndian.Uint32(data[72:76])
	actualChecksum := crc32.ChecksumIEEE(data[0:72])
	if expectedChecksum != actualChecksum {
		return errors.New("checksum mismatch")
	}

	// Header
	m.Header.Length = binary.LittleEndian.Uint32(data[0:4])
	m.Header.Type = binary.LittleEndian.Uint16(data[4:6])
	m.Header.Flags = binary.LittleEndian.Uint16(data[6:8])
	m.Header.Timestamp = int64(binary.LittleEndian.Uint64(data[8:16]))
	m.Header.SeqNum = binary.LittleEndian.Uint64(data[16:24])

	// Verify message type
	if m.Header.Type != MsgTypeNewOrder {
		return errors.New("invalid message type")
	}

	// Payload
	m.OrderID = binary.LittleEndian.Uint64(data[24:32])
	m.Price = int64(binary.LittleEndian.Uint64(data[32:40]))
	m.Quantity = int64(binary.LittleEndian.Uint64(data[40:48]))
	m.SymbolID = binary.LittleEndian.Uint32(data[48:52])
	m.Side = Side(data[52])
	m.Type = OrderType(data[53])
	m.TIF = TimeInForce(data[54])
	m.Flags = data[55]
	copy(m.ClientOrderID[:], data[56:72])

	return nil
}

// MarshalBinary encodes MarketDataMessage to binary
func (m *MarketDataMessage) MarshalBinary() []byte {
	buf := GetBuffer()
	b := (*buf)[:84] // 24 header + 56 payload + 4 checksum

	// Header
	binary.LittleEndian.PutUint32(b[0:4], 84)
	binary.LittleEndian.PutUint16(b[4:6], MsgTypeMarketData)
	binary.LittleEndian.PutUint16(b[6:8], m.Header.Flags)
	binary.LittleEndian.PutUint64(b[8:16], uint64(m.Header.Timestamp))
	binary.LittleEndian.PutUint64(b[16:24], m.Header.SeqNum)

	// Payload
	binary.LittleEndian.PutUint32(b[24:28], m.SymbolID)
	// 4 bytes padding
	binary.LittleEndian.PutUint64(b[32:40], uint64(m.BidPrice))
	binary.LittleEndian.PutUint64(b[40:48], uint64(m.BidQty))
	binary.LittleEndian.PutUint64(b[48:56], uint64(m.AskPrice))
	binary.LittleEndian.PutUint64(b[56:64], uint64(m.AskQty))
	binary.LittleEndian.PutUint64(b[64:72], uint64(m.LastPrice))
	binary.LittleEndian.PutUint64(b[72:80], uint64(m.LastQty))

	// Checksum
	checksum := crc32.ChecksumIEEE(b[0:80])
	binary.LittleEndian.PutUint32(b[80:84], checksum)

	return b
}

// UnmarshalBinary decodes MarketDataMessage from binary
func (m *MarketDataMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 84 {
		return errors.New("invalid message length")
	}

	// Verify checksum
	expectedChecksum := binary.LittleEndian.Uint32(data[80:84])
	actualChecksum := crc32.ChecksumIEEE(data[0:80])
	if expectedChecksum != actualChecksum {
		return errors.New("checksum mismatch")
	}

	// Header
	m.Header.Length = binary.LittleEndian.Uint32(data[0:4])
	m.Header.Type = binary.LittleEndian.Uint16(data[4:6])
	m.Header.Flags = binary.LittleEndian.Uint16(data[6:8])
	m.Header.Timestamp = int64(binary.LittleEndian.Uint64(data[8:16]))
	m.Header.SeqNum = binary.LittleEndian.Uint64(data[16:24])

	// Verify message type
	if m.Header.Type != MsgTypeMarketData {
		return errors.New("invalid message type")
	}

	// Payload
	m.SymbolID = binary.LittleEndian.Uint32(data[24:28])
	m.BidPrice = int64(binary.LittleEndian.Uint64(data[32:40]))
	m.BidQty = int64(binary.LittleEndian.Uint64(data[40:48]))
	m.AskPrice = int64(binary.LittleEndian.Uint64(data[48:56]))
	m.AskQty = int64(binary.LittleEndian.Uint64(data[56:64]))
	m.LastPrice = int64(binary.LittleEndian.Uint64(data[64:72]))
	m.LastQty = int64(binary.LittleEndian.Uint64(data[72:80]))

	return nil
}

// Size returns message size
func (m *NewOrderMessage) Size() int { return 76 }
func (m *MarketDataMessage) Size() int { return 84 }

// Zero-copy unsafe conversions for ultra-low latency (use with caution)

// UnsafeMarshalNewOrder converts NewOrderMessage to bytes without allocation
// WARNING: The returned slice shares memory with the message struct
func UnsafeMarshalNewOrder(m *NewOrderMessage) []byte {
	return (*[76]byte)(unsafe.Pointer(m))[:]
}

// UnsafeUnmarshalNewOrder converts bytes to NewOrderMessage without allocation
// WARNING: The message struct shares memory with the byte slice
func UnsafeUnmarshalNewOrder(data []byte) *NewOrderMessage {
	if len(data) < 76 {
		return nil
	}
	return (*NewOrderMessage)(unsafe.Pointer(&data[0]))
}

// Batch encoder for processing multiple messages
type BatchEncoder struct {
	buf    []byte
	offset int
}

// NewBatchEncoder creates a batch encoder with specified capacity
func NewBatchEncoder(capacity int) *BatchEncoder {
	return &BatchEncoder{
		buf: make([]byte, capacity),
	}
}

// EncodeNewOrder encodes a new order into the batch
func (e *BatchEncoder) EncodeNewOrder(m *NewOrderMessage) error {
	if e.offset+76 > len(e.buf) {
		return errors.New("buffer overflow")
	}

	data := m.MarshalBinary()
	copy(e.buf[e.offset:], data)
	e.offset += 76
	PutBuffer(&data)

	return nil
}

// Bytes returns the encoded batch
func (e *BatchEncoder) Bytes() []byte {
	return e.buf[:e.offset]
}

// Reset resets the encoder for reuse
func (e *BatchEncoder) Reset() {
	e.offset = 0
}
