package algo

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/hft/matching"
	"github.com/fluxorio/fluxor/pkg/hft/protocol"
)

// MarketMaker implements market making strategies
type MarketMaker struct {
	symbol           string
	symbolID         uint32
	orderBook        *matching.OrderBook
	
	// Configuration
	config           *MarketMakerConfig
	
	// State
	targetInventory  int64  // Target inventory level
	currentInventory int64  // Current position
	
	// Pricing
	midPrice         int64  // Current mid price
	volatility       float64 // Estimated volatility
	
	// Metrics
	_                [7]uint64
	quotesGenerated  uint64
	_                [7]uint64
	tradesExecuted   uint64
	_                [7]uint64
	pnl              int64  // Realized P&L
	_                [7]uint64
	
	mu               sync.RWMutex
	running          atomic.Bool
}

// MarketMakerConfig contains market maker parameters
type MarketMakerConfig struct {
	// Spread management
	BaseSpread       int64   // Base spread in ticks
	SpreadMultiplier float64 // Spread multiplier based on volatility
	MaxSpread        int64   // Maximum spread
	
	// Position management
	TargetInventory  int64   // Target position (usually 0 for neutral)
	MaxInventory     int64   // Maximum allowed position
	InventorySkew    float64 // Skew per unit of inventory
	
	// Quote sizing
	BaseSize         int64   // Base quote size
	MaxSize          int64   // Maximum quote size
	SizeDecay        float64 // Size decay with distance
	
	// Risk parameters
	MaxQuoteAge      time.Duration // Maximum quote age before refresh
	MinEdge          int64         // Minimum edge to quote
	
	// Hedging
	HedgeThreshold   int64         // Inventory threshold to hedge
	HedgeRatio       float64       // Ratio to hedge
}

// DefaultMarketMakerConfig returns default configuration
func DefaultMarketMakerConfig() *MarketMakerConfig {
	return &MarketMakerConfig{
		BaseSpread:       100000,     // $0.001 = 1 tick
		SpreadMultiplier: 1.5,
		MaxSpread:        1000000,    // $0.01
		TargetInventory:  0,
		MaxInventory:     1000,
		InventorySkew:    0.001,      // 0.1% skew per unit
		BaseSize:         100,
		MaxSize:          1000,
		SizeDecay:        0.1,
		MaxQuoteAge:      1 * time.Second,
		MinEdge:          50000,      // $0.0005
		HedgeThreshold:   500,
		HedgeRatio:       0.5,
	}
}

// NewMarketMaker creates a new market maker
func NewMarketMaker(symbol string, symbolID uint32, orderBook *matching.OrderBook, config *MarketMakerConfig) *MarketMaker {
	if config == nil {
		config = DefaultMarketMakerConfig()
	}
	
	return &MarketMaker{
		symbol:          symbol,
		symbolID:        symbolID,
		orderBook:       orderBook,
		config:          config,
		targetInventory: config.TargetInventory,
	}
}

// Start starts the market maker
func (mm *MarketMaker) Start() {
	mm.running.Store(true)
	
	go mm.run()
}

// Stop stops the market maker
func (mm *MarketMaker) Stop() {
	mm.running.Store(false)
}

// run is the main market maker loop
func (mm *MarketMaker) run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for mm.running.Load() {
		select {
		case <-ticker.C:
			mm.updateMarketState()
			mm.generateQuotes()
		}
	}
}

// updateMarketState updates current market state
func (mm *MarketMaker) updateMarketState() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	// Get current BBO
	bidPrice, _, askPrice, _ := mm.orderBook.GetBBO()
	
	if bidPrice > 0 && askPrice > 0 {
		// Calculate mid price
		mm.midPrice = (bidPrice + askPrice) / 2
		
		// Estimate volatility (simplified - use historical data in production)
		spread := askPrice - bidPrice
		mm.volatility = float64(spread) / float64(mm.midPrice) * 100.0
	}
}

// generateQuotes generates bid/ask quotes
func (mm *MarketMaker) generateQuotes() {
	mm.mu.RLock()
	midPrice := mm.midPrice
	volatility := mm.volatility
	inventory := atomic.LoadInt64(&mm.currentInventory)
	mm.mu.RUnlock()
	
	if midPrice == 0 {
		return // No market data yet
	}
	
	// Calculate spread
	spread := mm.calculateSpread(volatility)
	
	// Calculate inventory skew
	inventoryDelta := inventory - mm.config.TargetInventory
	skew := int64(float64(inventoryDelta) * mm.config.InventorySkew * float64(midPrice))
	
	// Calculate bid/ask prices with skew
	bidPrice := midPrice - spread/2 - skew
	askPrice := midPrice + spread/2 - skew
	
	// Ensure minimum edge
	if askPrice-bidPrice < mm.config.MinEdge*2 {
		return
	}
	
	// Calculate quote sizes
	bidSize := mm.calculateSize(inventoryDelta, true)
	askSize := mm.calculateSize(inventoryDelta, false)
	
	// Generate quotes
	if bidSize > 0 {
		mm.submitQuote(protocol.SideBuy, bidPrice, bidSize)
	}
	
	if askSize > 0 {
		mm.submitQuote(protocol.SideSell, askPrice, askSize)
	}
	
	atomic.AddUint64(&mm.quotesGenerated, 2)
}

// calculateSpread calculates current spread based on volatility
func (mm *MarketMaker) calculateSpread(volatility float64) int64 {
	spread := float64(mm.config.BaseSpread) * (1.0 + volatility*mm.config.SpreadMultiplier)
	
	spreadInt := int64(spread)
	if spreadInt > mm.config.MaxSpread {
		spreadInt = mm.config.MaxSpread
	}
	
	return spreadInt
}

// calculateSize calculates quote size based on inventory
func (mm *MarketMaker) calculateSize(inventoryDelta int64, isBid bool) int64 {
	baseSize := mm.config.BaseSize
	
	// Reduce size when inventory is away from target
	if isBid && inventoryDelta > 0 {
		// Long inventory, reduce bid size
		factor := 1.0 - float64(inventoryDelta)/float64(mm.config.MaxInventory)
		baseSize = int64(float64(baseSize) * factor)
	} else if !isBid && inventoryDelta < 0 {
		// Short inventory, reduce ask size
		factor := 1.0 - float64(-inventoryDelta)/float64(mm.config.MaxInventory)
		baseSize = int64(float64(baseSize) * factor)
	}
	
	// Ensure minimum size
	if baseSize < 10 {
		baseSize = 10
	}
	
	// Cap at maximum
	if baseSize > mm.config.MaxSize {
		baseSize = mm.config.MaxSize
	}
	
	return baseSize
}

// submitQuote submits a quote to the order book
func (mm *MarketMaker) submitQuote(side protocol.Side, price, size int64) {
	order := matching.GetOrder()
	order.Symbol = mm.symbol
	order.SymbolID = mm.symbolID
	order.Side = side
	order.Type = protocol.OrderTypeLimit
	order.Price = price
	order.Quantity = size
	order.TimeInForce = protocol.TIFIOC  // IOC for market making
	
	// Submit order (async in production)
	if err := mm.orderBook.AddOrder(order); err != nil {
		matching.PutOrder(order)
	}
}

// OnFill handles order fills
func (mm *MarketMaker) OnFill(order *matching.Order) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	// Update inventory
	if order.Side == protocol.SideBuy {
		atomic.AddInt64(&mm.currentInventory, order.FilledQty)
	} else {
		atomic.AddInt64(&mm.currentInventory, -order.FilledQty)
	}
	
	atomic.AddUint64(&mm.tradesExecuted, 1)
	
	// Check if hedging is needed
	inventory := atomic.LoadInt64(&mm.currentInventory)
	if abs64(inventory) > mm.config.HedgeThreshold {
		mm.hedge(inventory)
	}
}

// hedge performs hedging when inventory exceeds threshold
func (mm *MarketMaker) hedge(inventory int64) {
	hedgeSize := int64(float64(abs64(inventory)) * mm.config.HedgeRatio)
	
	if hedgeSize == 0 {
		return
	}
	
	// Determine hedge side (opposite of inventory)
	var hedgeSide protocol.Side
	if inventory > 0 {
		hedgeSide = protocol.SideSell
	} else {
		hedgeSide = protocol.SideBuy
	}
	
	// Submit hedge order (market order for immediate execution)
	order := matching.GetOrder()
	order.Symbol = mm.symbol
	order.SymbolID = mm.symbolID
	order.Side = hedgeSide
	order.Type = protocol.OrderTypeMarket
	order.Quantity = hedgeSize
	order.TimeInForce = protocol.TIFIOC
	
	if err := mm.orderBook.AddOrder(order); err != nil {
		matching.PutOrder(order)
	}
}

// GetMetrics returns market maker metrics
func (mm *MarketMaker) GetMetrics() MarketMakerMetrics {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	return MarketMakerMetrics{
		Symbol:          mm.symbol,
		CurrentInventory: atomic.LoadInt64(&mm.currentInventory),
		TargetInventory: mm.targetInventory,
		MidPrice:        mm.midPrice,
		Volatility:      mm.volatility,
		QuotesGenerated: atomic.LoadUint64(&mm.quotesGenerated),
		TradesExecuted:  atomic.LoadUint64(&mm.tradesExecuted),
		RealizedPnL:     atomic.LoadInt64(&mm.pnl),
	}
}

// MarketMakerMetrics contains market maker statistics
type MarketMakerMetrics struct {
	Symbol           string
	CurrentInventory int64
	TargetInventory  int64
	MidPrice         int64
	Volatility       float64
	QuotesGenerated  uint64
	TradesExecuted   uint64
	RealizedPnL      int64
}

// SpreadStrategy defines different spread strategies
type SpreadStrategy int

const (
	StrategyFixed SpreadStrategy = iota
	StrategyVolatility
	StrategyInventory
	StrategyAdaptive
)

// AdvancedMarketMaker with multiple strategies
type AdvancedMarketMaker struct {
	*MarketMaker
	strategy SpreadStrategy
}

// NewAdvancedMarketMaker creates an advanced market maker
func NewAdvancedMarketMaker(symbol string, symbolID uint32, orderBook *matching.OrderBook, 
	config *MarketMakerConfig, strategy SpreadStrategy) *AdvancedMarketMaker {
	
	return &AdvancedMarketMaker{
		MarketMaker: NewMarketMaker(symbol, symbolID, orderBook, config),
		strategy:    strategy,
	}
}

// calculateSpread with advanced strategies
func (amm *AdvancedMarketMaker) calculateSpread(volatility float64) int64 {
	switch amm.strategy {
	case StrategyFixed:
		return amm.config.BaseSpread
		
	case StrategyVolatility:
		// Spread increases with volatility
		spread := float64(amm.config.BaseSpread) * (1.0 + volatility)
		return int64(math.Min(spread, float64(amm.config.MaxSpread)))
		
	case StrategyInventory:
		// Spread widens when inventory is away from target
		inventory := atomic.LoadInt64(&amm.currentInventory)
		inventoryFactor := float64(abs64(inventory)) / float64(amm.config.MaxInventory)
		spread := float64(amm.config.BaseSpread) * (1.0 + inventoryFactor)
		return int64(math.Min(spread, float64(amm.config.MaxSpread)))
		
	case StrategyAdaptive:
		// Combination of volatility and inventory
		inventory := atomic.LoadInt64(&amm.currentInventory)
		inventoryFactor := float64(abs64(inventory)) / float64(amm.config.MaxInventory)
		spread := float64(amm.config.BaseSpread) * (1.0 + volatility + inventoryFactor)
		return int64(math.Min(spread, float64(amm.config.MaxSpread)))
		
	default:
		return amm.config.BaseSpread
	}
}

func abs64(i int64) int64 {
	if i < 0 {
		return -i
	}
	return i
}
