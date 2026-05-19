package matching

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/hft/protocol"
)

// Order represents a trading order in the matching engine
type Order struct {
	ID            uint64
	Symbol        string
	SymbolID      uint32
	Side          protocol.Side
	Type          protocol.OrderType
	Price         int64 // Fixed-point: price * 1e8
	Quantity      int64
	FilledQty     int64
	LeavesQty     int64
	Status        protocol.OrderStatus
	TimeInForce   protocol.TimeInForce
	Timestamp     int64 // Nanoseconds
	ClientOrderID [16]byte
	
	// Internal fields
	next *Order // Linked list for price level
	prev *Order
}

// OrderPool provides zero-allocation order pooling
type OrderPool struct {
	pool       sync.Pool
	allocCount uint64
}

// NewOrderPool creates a new order pool
func NewOrderPool() *OrderPool {
	return &OrderPool{
		pool: sync.Pool{
			New: func() interface{} {
				atomic.AddUint64(&orderPoolAllocCount, 1)
				return &Order{}
			},
		},
	}
}

var orderPoolAllocCount uint64

// Get retrieves an order from the pool
func (p *OrderPool) Get() *Order {
	return p.pool.Get().(*Order)
}

// Put returns an order to the pool
func (p *OrderPool) Put(o *Order) {
	o.Reset()
	p.pool.Put(o)
}

// Reset clears the order for reuse
func (o *Order) Reset() {
	o.ID = 0
	o.Symbol = ""
	o.SymbolID = 0
	o.Side = 0
	o.Type = 0
	o.Price = 0
	o.Quantity = 0
	o.FilledQty = 0
	o.LeavesQty = 0
	o.Status = 0
	o.TimeInForce = 0
	o.Timestamp = 0
	for i := range o.ClientOrderID {
		o.ClientOrderID[i] = 0
	}
	o.next = nil
	o.prev = nil
}

// Clone creates a copy of the order
func (o *Order) Clone() *Order {
	clone := &Order{
		ID:            o.ID,
		Symbol:        o.Symbol,
		SymbolID:      o.SymbolID,
		Side:          o.Side,
		Type:          o.Type,
		Price:         o.Price,
		Quantity:      o.Quantity,
		FilledQty:     o.FilledQty,
		LeavesQty:     o.LeavesQty,
		Status:        o.Status,
		TimeInForce:   o.TimeInForce,
		Timestamp:     o.Timestamp,
		ClientOrderID: o.ClientOrderID,
	}
	return clone
}

// IsBuy returns true if order is buy side
func (o *Order) IsBuy() bool {
	return o.Side == protocol.SideBuy
}

// IsSell returns true if order is sell side
func (o *Order) IsSell() bool {
	return o.Side == protocol.SideSell
}

// IsLimit returns true if order is limit type
func (o *Order) IsLimit() bool {
	return o.Type == protocol.OrderTypeLimit
}

// IsMarket returns true if order is market type
func (o *Order) IsMarket() bool {
	return o.Type == protocol.OrderTypeMarket
}

// CanMatch checks if this order can match with another order
func (o *Order) CanMatch(other *Order) bool {
	// Same side cannot match
	if o.Side == other.Side {
		return false
	}

	// Market orders can always match
	if o.IsMarket() || other.IsMarket() {
		return true
	}

	// Buy order can match if bid >= ask
	if o.IsBuy() {
		return o.Price >= other.Price
	}

	// Sell order can match if ask <= bid
	return o.Price <= other.Price
}

// Fill executes a partial or full fill on the order
func (o *Order) Fill(qty int64, price int64) *Fill {
	if qty <= 0 || qty > o.LeavesQty {
		return nil
	}

	fill := &Fill{
		OrderID:   o.ID,
		Timestamp: time.Now().UnixNano(),
		Price:     price,
		Quantity:  qty,
	}

	o.FilledQty += qty
	o.LeavesQty -= qty

	if o.LeavesQty == 0 {
		o.Status = protocol.StatusFilled
	} else {
		o.Status = protocol.StatusPartial
	}

	return fill
}

// Fill represents an order fill
type Fill struct {
	OrderID   uint64
	Timestamp int64
	Price     int64
	Quantity  int64
}

// Trade represents a matched trade between two orders
type Trade struct {
	BuyOrderID  uint64
	SellOrderID uint64
	Symbol      string
	Price       int64
	Quantity    int64
	Timestamp   int64
}

// PriceLevel represents all orders at a specific price point
type PriceLevel struct {
	Price    int64
	TotalQty int64
	Orders   *Order // Head of linked list
	Count    int32
	next     *PriceLevel // For price ladder
	prev     *PriceLevel
}

// NewPriceLevel creates a new price level
func NewPriceLevel(price int64) *PriceLevel {
	return &PriceLevel{
		Price: price,
	}
}

// AddOrder adds an order to this price level (FIFO - price-time priority)
func (pl *PriceLevel) AddOrder(order *Order) {
	if pl.Orders == nil {
		// First order at this price
		pl.Orders = order
		order.next = nil
		order.prev = nil
	} else {
		// Add to end of list (FIFO)
		tail := pl.Orders
		for tail.next != nil {
			tail = tail.next
		}
		tail.next = order
		order.prev = tail
		order.next = nil
	}

	pl.TotalQty += order.LeavesQty
	pl.Count++
}

// RemoveOrder removes an order from this price level
func (pl *PriceLevel) RemoveOrder(order *Order) bool {
	if order.prev != nil {
		order.prev.next = order.next
	} else {
		// Removing head
		pl.Orders = order.next
	}

	if order.next != nil {
		order.next.prev = order.prev
	}

	pl.TotalQty -= order.LeavesQty
	pl.Count--

	order.next = nil
	order.prev = nil

	return pl.Orders == nil // True if price level is now empty
}

// UpdateQuantity updates total quantity when an order is filled
func (pl *PriceLevel) UpdateQuantity(qtyChange int64) {
	pl.TotalQty += qtyChange
}

// IsEmpty returns true if no orders at this price
func (pl *PriceLevel) IsEmpty() bool {
	return pl.Orders == nil
}

// GetBestOrder returns the first order at this price level
func (pl *PriceLevel) GetBestOrder() *Order {
	return pl.Orders
}

// OrderIterator iterates through orders at a price level
type OrderIterator struct {
	current *Order
}

// NewOrderIterator creates an iterator for a price level
func (pl *PriceLevel) Iterator() *OrderIterator {
	return &OrderIterator{current: pl.Orders}
}

// Next returns the next order
func (it *OrderIterator) Next() *Order {
	if it.current == nil {
		return nil
	}
	order := it.current
	it.current = it.current.next
	return order
}

// HasNext returns true if more orders available
func (it *OrderIterator) HasNext() bool {
	return it.current != nil
}

// PriceLevelPool provides pooling for price levels
type PriceLevelPool struct {
	pool sync.Pool
}

// NewPriceLevelPool creates a price level pool
func NewPriceLevelPool() *PriceLevelPool {
	return &PriceLevelPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &PriceLevel{}
			},
		},
	}
}

// Get retrieves a price level from the pool
func (p *PriceLevelPool) Get(price int64) *PriceLevel {
	pl := p.pool.Get().(*PriceLevel)
	pl.Price = price
	pl.TotalQty = 0
	pl.Orders = nil
	pl.Count = 0
	pl.next = nil
	pl.prev = nil
	return pl
}

// Put returns a price level to the pool
func (p *PriceLevelPool) Put(pl *PriceLevel) {
	pl.Price = 0
	pl.TotalQty = 0
	pl.Orders = nil
	pl.Count = 0
	pl.next = nil
	pl.prev = nil
	p.pool.Put(pl)
}

// Global pools
var (
	globalOrderPool      = NewOrderPool()
	globalPriceLevelPool = NewPriceLevelPool()
)

// GetOrder gets an order from the global pool
func GetOrder() *Order {
	return globalOrderPool.Get()
}

// PutOrder returns an order to the global pool
func PutOrder(o *Order) {
	globalOrderPool.Put(o)
}

// GetPriceLevel gets a price level from the global pool
func GetPriceLevel(price int64) *PriceLevel {
	return globalPriceLevelPool.Get(price)
}

// PutPriceLevel returns a price level to the global pool
func PutPriceLevel(pl *PriceLevel) {
	globalPriceLevelPool.Put(pl)
}
