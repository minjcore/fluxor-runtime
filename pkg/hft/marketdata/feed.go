package marketdata

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/hft/protocol"
)

// Tick represents a market data tick
type Tick struct {
	Symbol    string
	SymbolID  uint32
	BidPrice  int64
	BidQty    int64
	AskPrice  int64
	AskQty    int64
	LastPrice int64
	LastQty   int64
	Timestamp int64
}

// TickStorage provides high-performance tick storage with rotation
type TickStorage struct {
	ticks      []Tick
	capacity   int
	index      uint64
	rotated    uint64
	mu         sync.RWMutex
}

// NewTickStorage creates a new tick storage
func NewTickStorage(capacity int) *TickStorage {
	return &TickStorage{
		ticks:    make([]Tick, capacity),
		capacity: capacity,
	}
}

// Write writes a tick to storage
func (ts *TickStorage) Write(tick Tick) {
	idx := atomic.AddUint64(&ts.index, 1) - 1
	pos := idx % uint64(ts.capacity)
	
	// Check for rotation
	if idx >= uint64(ts.capacity) && idx%uint64(ts.capacity) == 0 {
		atomic.AddUint64(&ts.rotated, 1)
	}
	
	ts.mu.Lock()
	ts.ticks[pos] = tick
	ts.mu.Unlock()
}

// GetLast returns the last N ticks
func (ts *TickStorage) GetLast(n int) []Tick {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	currentIdx := atomic.LoadUint64(&ts.index)
	if currentIdx == 0 {
		return nil
	}
	
	if n > ts.capacity {
		n = ts.capacity
	}
	
	result := make([]Tick, 0, n)
	for i := 0; i < n; i++ {
		pos := (currentIdx - uint64(i) - 1) % uint64(ts.capacity)
		if pos < uint64(len(ts.ticks)) {
			result = append(result, ts.ticks[pos])
		}
	}
	
	return result
}

// Stats returns storage statistics
func (ts *TickStorage) Stats() (total, rotations uint64) {
	return atomic.LoadUint64(&ts.index), atomic.LoadUint64(&ts.rotated)
}

// MarketDataFeed handles market data updates
type MarketDataFeed struct {
	symbols     sync.Map // Symbol -> *SymbolData
	subscribers sync.Map // Symbol -> []chan *Tick
	storage     *TickStorage
	
	// Metrics
	_               [7]uint64
	ticksReceived   uint64
	_               [7]uint64
	ticksPublished  uint64
	_               [7]uint64
}

// SymbolData contains current market data for a symbol
type SymbolData struct {
	Symbol    string
	SymbolID  uint32
	BidPrice  int64
	BidQty    int64
	AskPrice  int64
	AskQty    int64
	LastPrice int64
	LastQty   int64
	Timestamp int64
	mu        sync.RWMutex
}

// NewMarketDataFeed creates a new market data feed
func NewMarketDataFeed(tickStorageCapacity int) *MarketDataFeed {
	return &MarketDataFeed{
		storage: NewTickStorage(tickStorageCapacity),
	}
}

// Update updates market data for a symbol
func (mdf *MarketDataFeed) Update(tick *Tick) {
	atomic.AddUint64(&mdf.ticksReceived, 1)
	
	// Update symbol data
	data := mdf.getOrCreateSymbolData(tick.Symbol, tick.SymbolID)
	data.mu.Lock()
	data.BidPrice = tick.BidPrice
	data.BidQty = tick.BidQty
	data.AskPrice = tick.AskPrice
	data.AskQty = tick.AskQty
	data.LastPrice = tick.LastPrice
	data.LastQty = tick.LastQty
	data.Timestamp = tick.Timestamp
	data.mu.Unlock()
	
	// Store tick
	mdf.storage.Write(*tick)
	
	// Publish to subscribers
	value, ok := mdf.subscribers.Load(tick.Symbol)
	if ok {
		subs := value.([]chan *Tick)
		for _, ch := range subs {
			select {
			case ch <- tick:
				atomic.AddUint64(&mdf.ticksPublished, 1)
			default:
				// Subscriber slow, skip
			}
		}
	}
}

// UpdateFromMessage updates from binary protocol message
func (mdf *MarketDataFeed) UpdateFromMessage(msg *protocol.MarketDataMessage) {
	tick := &Tick{
		SymbolID:  msg.SymbolID,
		BidPrice:  msg.BidPrice,
		BidQty:    msg.BidQty,
		AskPrice:  msg.AskPrice,
		AskQty:    msg.AskQty,
		LastPrice: msg.LastPrice,
		LastQty:   msg.LastQty,
		Timestamp: msg.Header.Timestamp,
	}
	
	mdf.Update(tick)
}

// Subscribe subscribes to market data updates for a symbol
func (mdf *MarketDataFeed) Subscribe(symbol string) chan *Tick {
	ch := make(chan *Tick, 100) // Buffered channel
	
	value, _ := mdf.subscribers.LoadOrStore(symbol, []chan *Tick{})
	subs := value.([]chan *Tick)
	subs = append(subs, ch)
	mdf.subscribers.Store(symbol, subs)
	
	return ch
}

// GetBBO returns best bid/offer for a symbol
func (mdf *MarketDataFeed) GetBBO(symbol string) (bidPrice, bidQty, askPrice, askQty int64, ok bool) {
	value, exists := mdf.symbols.Load(symbol)
	if !exists {
		return 0, 0, 0, 0, false
	}
	
	data := value.(*SymbolData)
	data.mu.RLock()
	defer data.mu.RUnlock()
	
	return data.BidPrice, data.BidQty, data.AskPrice, data.AskQty, true
}

// GetLastTick returns the last tick for a symbol
func (mdf *MarketDataFeed) GetLastTick(symbol string) (*Tick, bool) {
	value, exists := mdf.symbols.Load(symbol)
	if !exists {
		return nil, false
	}
	
	data := value.(*SymbolData)
	data.mu.RLock()
	defer data.mu.RUnlock()
	
	return &Tick{
		Symbol:    data.Symbol,
		SymbolID:  data.SymbolID,
		BidPrice:  data.BidPrice,
		BidQty:    data.BidQty,
		AskPrice:  data.AskPrice,
		AskQty:    data.AskQty,
		LastPrice: data.LastPrice,
		LastQty:   data.LastQty,
		Timestamp: data.Timestamp,
	}, true
}

// GetHistoricalTicks returns last N ticks for a symbol
func (mdf *MarketDataFeed) GetHistoricalTicks(n int) []Tick {
	return mdf.storage.GetLast(n)
}

// getOrCreateSymbolData gets or creates symbol data
func (mdf *MarketDataFeed) getOrCreateSymbolData(symbol string, symbolID uint32) *SymbolData {
	value, _ := mdf.symbols.LoadOrStore(symbol, &SymbolData{
		Symbol:   symbol,
		SymbolID: symbolID,
	})
	return value.(*SymbolData)
}

// Metrics returns feed metrics
func (mdf *MarketDataFeed) Metrics() MarketDataMetrics {
	ticksTotal, rotations := mdf.storage.Stats()
	
	return MarketDataMetrics{
		TicksReceived:  atomic.LoadUint64(&mdf.ticksReceived),
		TicksPublished: atomic.LoadUint64(&mdf.ticksPublished),
		TicksStored:    ticksTotal,
		StorageRotations: rotations,
	}
}

// MarketDataMetrics contains market data statistics
type MarketDataMetrics struct {
	TicksReceived    uint64
	TicksPublished   uint64
	TicksStored      uint64
	StorageRotations uint64
}

// BarAggregator aggregates ticks into OHLCV bars
type BarAggregator struct {
	interval time.Duration
	bars     sync.Map // Symbol -> *Bar
}

// Bar represents an OHLCV bar
type Bar struct {
	Symbol    string
	Open      int64
	High      int64
	Low       int64
	Close     int64
	Volume    int64
	Timestamp int64
	mu        sync.Mutex
}

// NewBarAggregator creates a new bar aggregator
func NewBarAggregator(interval time.Duration) *BarAggregator {
	return &BarAggregator{
		interval: interval,
	}
}

// Update updates bar with a tick
func (ba *BarAggregator) Update(tick *Tick) {
	bar := ba.getOrCreateBar(tick.Symbol, tick.Timestamp)
	bar.mu.Lock()
	defer bar.mu.Unlock()
	
	if bar.Open == 0 {
		bar.Open = tick.LastPrice
	}
	
	if tick.LastPrice > bar.High {
		bar.High = tick.LastPrice
	}
	
	if bar.Low == 0 || tick.LastPrice < bar.Low {
		bar.Low = tick.LastPrice
	}
	
	bar.Close = tick.LastPrice
	bar.Volume += tick.LastQty
}

// GetBar returns current bar for a symbol
func (ba *BarAggregator) GetBar(symbol string) *Bar {
	value, ok := ba.bars.Load(symbol)
	if !ok {
		return nil
	}
	
	bar := value.(*Bar)
	bar.mu.Lock()
	defer bar.mu.Unlock()
	
	// Return a copy
	return &Bar{
		Symbol:    bar.Symbol,
		Open:      bar.Open,
		High:      bar.High,
		Low:       bar.Low,
		Close:     bar.Close,
		Volume:    bar.Volume,
		Timestamp: bar.Timestamp,
	}
}

func (ba *BarAggregator) getOrCreateBar(symbol string, timestamp int64) *Bar {
	barTimestamp := (timestamp / int64(ba.interval)) * int64(ba.interval)
	key := symbol + "_" + string(rune(barTimestamp))
	
	value, _ := ba.bars.LoadOrStore(key, &Bar{
		Symbol:    symbol,
		Timestamp: barTimestamp,
	})
	
	return value.(*Bar)
}
