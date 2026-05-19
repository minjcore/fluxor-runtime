package protocol

import (
	"sync"
	"sync/atomic"
)

// ObjectPool provides zero-allocation object pooling for HFT
type ObjectPool struct {
	pool       sync.Pool
	newFunc    func() interface{}
	getCount   uint64
	putCount   uint64
	allocCount uint64
}

// NewObjectPool creates a new object pool
func NewObjectPool(newFunc func() interface{}) *ObjectPool {
	return &ObjectPool{
		pool: sync.Pool{
			New: func() interface{} {
				atomic.AddUint64(&allocCount, 1)
				return newFunc()
			},
		},
		newFunc: newFunc,
	}
}

// Get retrieves an object from the pool
func (p *ObjectPool) Get() interface{} {
	atomic.AddUint64(&p.getCount, 1)
	return p.pool.Get()
}

// Put returns an object to the pool
func (p *ObjectPool) Put(obj interface{}) {
	atomic.AddUint64(&p.putCount, 1)
	p.pool.Put(obj)
}

// Stats returns pool statistics
func (p *ObjectPool) Stats() (get, put, alloc uint64) {
	return atomic.LoadUint64(&p.getCount),
		atomic.LoadUint64(&p.putCount),
		atomic.LoadUint64(&p.allocCount)
}

// Global message pools
var (
	allocCount uint64 // Track total allocations across all pools

	newOrderPool = NewObjectPool(func() interface{} {
		return &NewOrderMessage{}
	})

	cancelOrderPool = NewObjectPool(func() interface{} {
		return &CancelOrderMessage{}
	})

	orderAckPool = NewObjectPool(func() interface{} {
		return &OrderAckMessage{}
	})

	fillPool = NewObjectPool(func() interface{} {
		return &FillMessage{}
	})

	marketDataPool = NewObjectPool(func() interface{} {
		return &MarketDataMessage{}
	})

	rejectPool = NewObjectPool(func() interface{} {
		return &RejectMessage{}
	})
)

// GetNewOrder gets a NewOrderMessage from the pool
func GetNewOrder() *NewOrderMessage {
	return newOrderPool.Get().(*NewOrderMessage)
}

// PutNewOrder returns a NewOrderMessage to the pool
func PutNewOrder(m *NewOrderMessage) {
	// Clear sensitive data
	m.Reset()
	newOrderPool.Put(m)
}

// Reset clears the NewOrderMessage for reuse
func (m *NewOrderMessage) Reset() {
	m.OrderID = 0
	m.Price = 0
	m.Quantity = 0
	m.SymbolID = 0
	m.Side = 0
	m.Type = 0
	m.TIF = 0
	m.Flags = 0
	for i := range m.ClientOrderID {
		m.ClientOrderID[i] = 0
	}
}

// GetCancelOrder gets a CancelOrderMessage from the pool
func GetCancelOrder() *CancelOrderMessage {
	return cancelOrderPool.Get().(*CancelOrderMessage)
}

// PutCancelOrder returns a CancelOrderMessage to the pool
func PutCancelOrder(m *CancelOrderMessage) {
	m.Reset()
	cancelOrderPool.Put(m)
}

// Reset clears the CancelOrderMessage for reuse
func (m *CancelOrderMessage) Reset() {
	m.OrderID = 0
	m.OrigOrderID = 0
	for i := range m.ClientOrderID {
		m.ClientOrderID[i] = 0
	}
}

// GetOrderAck gets an OrderAckMessage from the pool
func GetOrderAck() *OrderAckMessage {
	return orderAckPool.Get().(*OrderAckMessage)
}

// PutOrderAck returns an OrderAckMessage to the pool
func PutOrderAck(m *OrderAckMessage) {
	m.Reset()
	orderAckPool.Put(m)
}

// Reset clears the OrderAckMessage for reuse
func (m *OrderAckMessage) Reset() {
	m.OrderID = 0
	m.Status = 0
	for i := range m.ClientOrderID {
		m.ClientOrderID[i] = 0
	}
}

// GetFill gets a FillMessage from the pool
func GetFill() *FillMessage {
	return fillPool.Get().(*FillMessage)
}

// PutFill returns a FillMessage to the pool
func PutFill(m *FillMessage) {
	m.Reset()
	fillPool.Put(m)
}

// Reset clears the FillMessage for reuse
func (m *FillMessage) Reset() {
	m.OrderID = 0
	m.FillPrice = 0
	m.FillQty = 0
	m.Leaves = 0
	m.Status = 0
}

// GetMarketData gets a MarketDataMessage from the pool
func GetMarketData() *MarketDataMessage {
	return marketDataPool.Get().(*MarketDataMessage)
}

// PutMarketData returns a MarketDataMessage to the pool
func PutMarketData(m *MarketDataMessage) {
	m.Reset()
	marketDataPool.Put(m)
}

// Reset clears the MarketDataMessage for reuse
func (m *MarketDataMessage) Reset() {
	m.SymbolID = 0
	m.BidPrice = 0
	m.BidQty = 0
	m.AskPrice = 0
	m.AskQty = 0
	m.LastPrice = 0
	m.LastQty = 0
}

// GetReject gets a RejectMessage from the pool
func GetReject() *RejectMessage {
	return rejectPool.Get().(*RejectMessage)
}

// PutReject returns a RejectMessage to the pool
func PutReject(m *RejectMessage) {
	m.Reset()
	rejectPool.Put(m)
}

// Reset clears the RejectMessage for reuse
func (m *RejectMessage) Reset() {
	m.OrderID = 0
	m.RejectCode = 0
	for i := range m.RejectReason {
		m.RejectReason[i] = 0
	}
}

// PoolStats returns statistics for all message pools
func PoolStats() map[string][3]uint64 {
	return map[string][3]uint64{
		"newOrder":    {newOrderPool.getCount, newOrderPool.putCount, newOrderPool.allocCount},
		"cancelOrder": {cancelOrderPool.getCount, cancelOrderPool.putCount, cancelOrderPool.allocCount},
		"orderAck":    {orderAckPool.getCount, orderAckPool.putCount, orderAckPool.allocCount},
		"fill":        {fillPool.getCount, fillPool.putCount, fillPool.allocCount},
		"marketData":  {marketDataPool.getCount, marketDataPool.putCount, marketDataPool.allocCount},
		"reject":      {rejectPool.getCount, rejectPool.putCount, rejectPool.allocCount},
	}
}
