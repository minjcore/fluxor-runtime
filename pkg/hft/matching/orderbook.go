package matching

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/hft/protocol"
)

// OrderBook represents a lock-free order book for a single symbol
type OrderBook struct {
	Symbol   string
	SymbolID uint32

	// Price levels (sorted linked lists)
	bestBid     *PriceLevel  // Best bid (highest buy price)
	bestAsk     *PriceLevel  // Best ask (lowest sell price)
	bids        sync.Map     // Price -> *PriceLevel (buy orders)
	asks        sync.Map     // Price -> *PriceLevel (sell orders)
	orders      sync.Map     // OrderID -> *Order
	
	// Metrics (cache-line padded)
	_                [7]uint64
	totalTrades      uint64
	_                [7]uint64
	totalVolume      int64
	_                [7]uint64
	lastTradePrice   int64
	_                [7]uint64
	lastTradeTime    int64
	_                [7]uint64
	
	// Sequence number for ordering
	sequence         uint64
	
	// Callbacks
	onTrade          func(*Trade)
	onOrderUpdate    func(*Order)
	
	mu               sync.RWMutex  // For structural updates
}

// NewOrderBook creates a new order book
func NewOrderBook(symbol string, symbolID uint32) *OrderBook {
	return &OrderBook{
		Symbol:   symbol,
		SymbolID: symbolID,
	}
}

// SetCallbacks sets callback functions
func (ob *OrderBook) SetCallbacks(onTrade func(*Trade), onOrderUpdate func(*Order)) {
	ob.onTrade = onTrade
	ob.onOrderUpdate = onOrderUpdate
}

// AddOrder adds a new order to the order book
func (ob *OrderBook) AddOrder(order *Order) error {
	if order == nil {
		return errors.New("nil order")
	}

	// Validate order
	if order.Quantity <= 0 {
		return errors.New("invalid quantity")
	}

	// Set initial values
	order.Status = protocol.StatusNew
	order.LeavesQty = order.Quantity
	order.FilledQty = 0
	order.Timestamp = time.Now().UnixNano()
	order.ID = atomic.AddUint64(&ob.sequence, 1)

	// Store order
	ob.orders.Store(order.ID, order)

	// Match immediately for market orders or matching limit orders
	if order.IsMarket() || ob.canMatchImmediately(order) {
		ob.matchOrder(order)
	}

	// Add remaining quantity to book
	if order.LeavesQty > 0 && order.Type == protocol.OrderTypeLimit {
		ob.addToBook(order)
	}

	// Trigger callback
	if ob.onOrderUpdate != nil {
		ob.onOrderUpdate(order.Clone())
	}

	return nil
}

// CancelOrder cancels an existing order
func (ob *OrderBook) CancelOrder(orderID uint64) error {
	value, exists := ob.orders.Load(orderID)
	if !exists {
		return errors.New("order not found")
	}

	order := value.(*Order)
	
	// Remove from book
	ob.removeFromBook(order)
	
	// Update status
	order.Status = protocol.StatusCanceled
	
	// Remove from orders map
	ob.orders.Delete(orderID)
	
	// Trigger callback
	if ob.onOrderUpdate != nil {
		ob.onOrderUpdate(order.Clone())
	}

	return nil
}

// canMatchImmediately checks if order can match immediately
func (ob *OrderBook) canMatchImmediately(order *Order) bool {
	if order.IsBuy() {
		if ob.bestAsk == nil {
			return false
		}
		return order.IsMarket() || order.Price >= ob.bestAsk.Price
	} else {
		if ob.bestBid == nil {
			return false
		}
		return order.IsMarket() || order.Price <= ob.bestBid.Price
	}
}

// matchOrder matches an aggressive order against the book
func (ob *OrderBook) matchOrder(order *Order) {
	var oppositeSide *PriceLevel
	
	if order.IsBuy() {
		oppositeSide = ob.bestAsk
	} else {
		oppositeSide = ob.bestBid
	}

	for oppositeSide != nil && order.LeavesQty > 0 {
		// Check if can match at this price
		if !order.IsMarket() && !order.CanMatch(&Order{Price: oppositeSide.Price, Side: oppositeSide.Orders.Side}) {
			break
		}

		// Match against orders at this price level
		iterator := oppositeSide.Iterator()
		for iterator.HasNext() && order.LeavesQty > 0 {
			restingOrder := iterator.Next()
			
			// Calculate match quantity
			matchQty := minInt64(order.LeavesQty, restingOrder.LeavesQty)
			matchPrice := restingOrder.Price // Price-time priority: resting order price

			// Execute fills
			order.Fill(matchQty, matchPrice)
			restingOrder.Fill(matchQty, matchPrice)

			// Update price level
			oppositeSide.UpdateQuantity(-matchQty)

			// Create trade
			trade := ob.createTrade(order, restingOrder, matchQty, matchPrice)
			
			// Update metrics
			atomic.AddUint64(&ob.totalTrades, 1)
			atomic.AddInt64(&ob.totalVolume, matchQty)
			atomic.StoreInt64(&ob.lastTradePrice, matchPrice)
			atomic.StoreInt64(&ob.lastTradeTime, time.Now().UnixNano())

			// Trigger callback
			if ob.onTrade != nil {
				ob.onTrade(trade)
			}

			// Remove filled orders
			if restingOrder.LeavesQty == 0 {
				oppositeSide.RemoveOrder(restingOrder)
				ob.orders.Delete(restingOrder.ID)
				
				if ob.onOrderUpdate != nil {
					ob.onOrderUpdate(restingOrder.Clone())
				}
			}

			// Update order
			if ob.onOrderUpdate != nil {
				ob.onOrderUpdate(order.Clone())
			}
		}

		// Move to next price level if current is empty
		if oppositeSide.IsEmpty() {
			ob.removeEmptyPriceLevel(oppositeSide)
			if order.IsBuy() {
				oppositeSide = ob.bestAsk
			} else {
				oppositeSide = ob.bestBid
			}
		} else {
			break
		}
	}

	// Check IOC (Immediate Or Cancel)
	if order.TimeInForce == protocol.TIFIOC && order.LeavesQty > 0 {
		order.Status = protocol.StatusCanceled
	}

	// Check FOK (Fill Or Kill)
	if order.TimeInForce == protocol.TIFFOK && order.FilledQty < order.Quantity {
		// Reverse fills (not implemented in this simplified version)
		order.Status = protocol.StatusCanceled
	}
}

// addToBook adds an order to the order book
func (ob *OrderBook) addToBook(order *Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var priceLevelMap *sync.Map
	if order.IsBuy() {
		priceLevelMap = &ob.bids
	} else {
		priceLevelMap = &ob.asks
	}

	// Get or create price level
	var priceLevel *PriceLevel
	value, exists := priceLevelMap.Load(order.Price)
	if exists {
		priceLevel = value.(*PriceLevel)
	} else {
		priceLevel = GetPriceLevel(order.Price)
		priceLevelMap.Store(order.Price, priceLevel)
	}

	// Add order to price level
	priceLevel.AddOrder(order)

	// Update best bid/ask
	ob.updateBestPrices(order.IsBuy(), priceLevel)
}

// removeFromBook removes an order from the order book
func (ob *OrderBook) removeFromBook(order *Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var priceLevelMap *sync.Map
	if order.IsBuy() {
		priceLevelMap = &ob.bids
	} else {
		priceLevelMap = &ob.asks
	}

	value, exists := priceLevelMap.Load(order.Price)
	if !exists {
		return
	}

	priceLevel := value.(*PriceLevel)
	isEmpty := priceLevel.RemoveOrder(order)

	if isEmpty {
		priceLevelMap.Delete(order.Price)
		PutPriceLevel(priceLevel)
		
		// Update best bid/ask
		if order.IsBuy() {
			ob.updateBestBid()
		} else {
			ob.updateBestAsk()
		}
	}
}

// removeEmptyPriceLevel removes an empty price level
func (ob *OrderBook) removeEmptyPriceLevel(pl *PriceLevel) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	if pl.Orders.IsBuy() {
		ob.bids.Delete(pl.Price)
		ob.updateBestBid()
	} else {
		ob.asks.Delete(pl.Price)
		ob.updateBestAsk()
	}

	PutPriceLevel(pl)
}

// updateBestPrices updates best bid/ask after adding an order
func (ob *OrderBook) updateBestPrices(isBuy bool, newLevel *PriceLevel) {
	if isBuy {
		if ob.bestBid == nil || newLevel.Price > ob.bestBid.Price {
			ob.bestBid = newLevel
		}
	} else {
		if ob.bestAsk == nil || newLevel.Price < ob.bestAsk.Price {
			ob.bestAsk = newLevel
		}
	}
}

// updateBestBid updates best bid price
func (ob *OrderBook) updateBestBid() {
	ob.bestBid = nil
	var maxPrice int64 = 0

	ob.bids.Range(func(key, value interface{}) bool {
		pl := value.(*PriceLevel)
		if !pl.IsEmpty() && pl.Price > maxPrice {
			maxPrice = pl.Price
			ob.bestBid = pl
		}
		return true
	})
}

// updateBestAsk updates best ask price
func (ob *OrderBook) updateBestAsk() {
	ob.bestAsk = nil
	var minPrice int64 = 1<<63 - 1 // Max int64

	ob.asks.Range(func(key, value interface{}) bool {
		pl := value.(*PriceLevel)
		if !pl.IsEmpty() && pl.Price < minPrice {
			minPrice = pl.Price
			ob.bestAsk = pl
		}
		return true
	})
}

// createTrade creates a trade from matched orders
func (ob *OrderBook) createTrade(aggressor, resting *Order, qty, price int64) *Trade {
	var buyOrderID, sellOrderID uint64
	if aggressor.IsBuy() {
		buyOrderID = aggressor.ID
		sellOrderID = resting.ID
	} else {
		buyOrderID = resting.ID
		sellOrderID = aggressor.ID
	}

	return &Trade{
		BuyOrderID:  buyOrderID,
		SellOrderID: sellOrderID,
		Symbol:      ob.Symbol,
		Price:       price,
		Quantity:    qty,
		Timestamp:   time.Now().UnixNano(),
	}
}

// GetBBO returns best bid and offer
func (ob *OrderBook) GetBBO() (bidPrice, bidQty, askPrice, askQty int64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if ob.bestBid != nil {
		bidPrice = ob.bestBid.Price
		bidQty = ob.bestBid.TotalQty
	}

	if ob.bestAsk != nil {
		askPrice = ob.bestAsk.Price
		askQty = ob.bestAsk.TotalQty
	}

	return
}

// GetDepth returns order book depth (top N levels)
func (ob *OrderBook) GetDepth(levels int) (bids, asks [][2]int64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	// Get bid depth
	count := 0
	current := ob.bestBid
	for current != nil && count < levels {
		bids = append(bids, [2]int64{current.Price, current.TotalQty})
		current = current.prev
		count++
	}

	// Get ask depth
	count = 0
	current = ob.bestAsk
	for current != nil && count < levels {
		asks = append(asks, [2]int64{current.Price, current.TotalQty})
		current = current.next
		count++
	}

	return
}

// Metrics returns order book metrics
func (ob *OrderBook) Metrics() OrderBookMetrics {
	bidPrice, bidQty, askPrice, askQty := ob.GetBBO()

	return OrderBookMetrics{
		Symbol:         ob.Symbol,
		BidPrice:       bidPrice,
		BidQty:         bidQty,
		AskPrice:       askPrice,
		AskQty:         askQty,
		Spread:         askPrice - bidPrice,
		TotalTrades:    atomic.LoadUint64(&ob.totalTrades),
		TotalVolume:    atomic.LoadInt64(&ob.totalVolume),
		LastTradePrice: atomic.LoadInt64(&ob.lastTradePrice),
		LastTradeTime:  atomic.LoadInt64(&ob.lastTradeTime),
	}
}

// OrderBookMetrics contains order book statistics
type OrderBookMetrics struct {
	Symbol         string
	BidPrice       int64
	BidQty         int64
	AskPrice       int64
	AskQty         int64
	Spread         int64
	TotalTrades    uint64
	TotalVolume    int64
	LastTradePrice int64
	LastTradeTime  int64
}

// String returns a string representation of the metrics
func (m OrderBookMetrics) String() string {
	return fmt.Sprintf("Symbol=%s Bid=%d@%d Ask=%d@%d Spread=%d Trades=%d Volume=%d Last=%d",
		m.Symbol, m.BidPrice, m.BidQty, m.AskPrice, m.AskQty, m.Spread, m.TotalTrades, m.TotalVolume, m.LastTradePrice)
}

// Helper functions
func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
