package risk

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/hft/matching"
	"github.com/fluxorio/fluxor/pkg/hft/protocol"
)

// RiskEngine provides pre-trade and post-trade risk checks
type RiskEngine struct {
	limits       *RiskLimits
	positions    sync.Map // Symbol -> *Position
	violations   *ViolationRing
	
	// Metrics (cache-line padded)
	_                [7]uint64
	checksTotal      uint64
	_                [7]uint64
	checksRejected   uint64
	_                [7]uint64
	checksLatencySum uint64 // nanoseconds
	_                [7]uint64
}

// RiskLimits defines risk parameters
type RiskLimits struct {
	// Position limits
	MaxPositionPerSymbol int64  // Max position size per symbol
	MaxNotionalPerOrder  int64  // Max notional per order
	MaxOrderSize         int64  // Max order quantity
	
	// Rate limits
	MaxOrdersPerSecond   int64  // Max orders per second
	MaxOrdersPerMinute   int64  // Max orders per minute
	
	// Price checks
	MaxPriceDeviation    float64 // Max price deviation from market (fat finger)
	
	// Credit limits
	MaxNotionalExposure  int64  // Max total notional exposure
	
	// Concentration limits
	MaxSymbolConcentration float64 // Max % of portfolio in one symbol
}

// DefaultRiskLimits returns sensible default limits
func DefaultRiskLimits() *RiskLimits {
	return &RiskLimits{
		MaxPositionPerSymbol:   100000,
		MaxNotionalPerOrder:    10000000, // $100k
		MaxOrderSize:           10000,
		MaxOrdersPerSecond:     100,
		MaxOrdersPerMinute:     1000,
		MaxPriceDeviation:      0.05, // 5%
		MaxNotionalExposure:    100000000, // $1M
		MaxSymbolConcentration: 0.20, // 20%
	}
}

// Position represents a trading position
type Position struct {
	Symbol       string
	Quantity     int64  // Net position (positive = long, negative = short)
	AvgPrice     int64  // Average entry price
	RealizedPnL  int64  // Realized P&L
	
	// Metrics
	TotalBought  int64
	TotalSold    int64
	
	mu           sync.RWMutex
}

// RiskViolation represents a risk rule violation
type RiskViolation struct {
	Timestamp int64
	OrderID   uint64
	Symbol    string
	Type      ViolationType
	Message   string
}

// ViolationType represents types of risk violations
type ViolationType int

const (
	ViolationPositionLimit ViolationType = iota
	ViolationNotionalLimit
	ViolationOrderSizeLimit
	ViolationRateLimit
	ViolationPriceDeviation
	ViolationCreditLimit
	ViolationConcentration
)

// ViolationRing is a ring buffer for violation history
type ViolationRing struct {
	violations [1000]*RiskViolation
	index      uint64
	mu         sync.RWMutex
}

// Add adds a violation to the ring
func (vr *ViolationRing) Add(v *RiskViolation) {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	
	idx := atomic.AddUint64(&vr.index, 1) % 1000
	vr.violations[idx] = v
}

// GetRecent returns recent violations
func (vr *ViolationRing) GetRecent(n int) []*RiskViolation {
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	
	if n > 1000 {
		n = 1000
	}
	
	result := make([]*RiskViolation, 0, n)
	idx := atomic.LoadUint64(&vr.index)
	
	for i := 0; i < n; i++ {
		pos := (idx - uint64(i)) % 1000
		if vr.violations[pos] != nil {
			result = append(result, vr.violations[pos])
		}
	}
	
	return result
}

// NewRiskEngine creates a new risk engine
func NewRiskEngine(limits *RiskLimits) *RiskEngine {
	if limits == nil {
		limits = DefaultRiskLimits()
	}
	
	return &RiskEngine{
		limits:     limits,
		violations: &ViolationRing{},
	}
}

// CheckOrder performs pre-trade risk checks
// Returns error if risk check fails, nil if passed
// Average latency target: < 1 microsecond
func (re *RiskEngine) CheckOrder(order *matching.Order, marketPrice int64) error {
	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime).Nanoseconds()
		atomic.AddUint64(&re.checksTotal, 1)
		atomic.AddUint64(&re.checksLatencySum, uint64(elapsed))
	}()
	
	// Check order size
	if order.Quantity > re.limits.MaxOrderSize {
		atomic.AddUint64(&re.checksRejected, 1)
		re.recordViolation(order, ViolationOrderSizeLimit, "Order size exceeds limit")
		return errors.New("order size exceeds limit")
	}
	
	// Check notional
	notional := order.Quantity * order.Price / 1e8 // Convert from fixed-point
	if notional > re.limits.MaxNotionalPerOrder {
		atomic.AddUint64(&re.checksRejected, 1)
		re.recordViolation(order, ViolationNotionalLimit, "Notional exceeds limit")
		return errors.New("notional exceeds limit")
	}
	
	// Check position limit
	if err := re.checkPositionLimit(order); err != nil {
		atomic.AddUint64(&re.checksRejected, 1)
		re.recordViolation(order, ViolationPositionLimit, err.Error())
		return err
	}
	
	// Check price deviation (fat finger)
	if order.Type == protocol.OrderTypeLimit && marketPrice > 0 {
		priceFloat := float64(order.Price) / 1e8
		marketFloat := float64(marketPrice) / 1e8
		deviation := abs(priceFloat-marketFloat) / marketFloat
		
		if deviation > re.limits.MaxPriceDeviation {
			atomic.AddUint64(&re.checksRejected, 1)
			re.recordViolation(order, ViolationPriceDeviation, "Price deviation too large")
			return errors.New("price deviation too large")
		}
	}
	
	// Check total notional exposure
	if err := re.checkNotionalExposure(order); err != nil {
		atomic.AddUint64(&re.checksRejected, 1)
		re.recordViolation(order, ViolationCreditLimit, err.Error())
		return err
	}
	
	return nil
}

// checkPositionLimit checks if order would exceed position limits
func (re *RiskEngine) checkPositionLimit(order *matching.Order) error {
	// Get current position
	pos := re.getOrCreatePosition(order.Symbol)
	pos.mu.RLock()
	currentQty := pos.Quantity
	pos.mu.RUnlock()
	
	// Calculate new position
	var newQty int64
	if order.Side == protocol.SideBuy {
		newQty = currentQty + order.Quantity
	} else {
		newQty = currentQty - order.Quantity
	}
	
	// Check limit
	if abs64(newQty) > re.limits.MaxPositionPerSymbol {
		return errors.New("position limit exceeded")
	}
	
	return nil
}

// checkNotionalExposure checks total notional exposure
func (re *RiskEngine) checkNotionalExposure(order *matching.Order) error {
	var totalExposure int64
	
	re.positions.Range(func(key, value interface{}) bool {
		pos := value.(*Position)
		pos.mu.RLock()
		exposure := abs64(pos.Quantity * pos.AvgPrice / 1e8)
		pos.mu.RUnlock()
		totalExposure += exposure
		return true
	})
	
	// Add new order exposure
	orderExposure := order.Quantity * order.Price / 1e8
	totalExposure += orderExposure
	
	if totalExposure > re.limits.MaxNotionalExposure {
		return errors.New("notional exposure limit exceeded")
	}
	
	return nil
}

// UpdatePosition updates position after a fill
func (re *RiskEngine) UpdatePosition(symbol string, side protocol.Side, qty, price int64) {
	pos := re.getOrCreatePosition(symbol)
	pos.mu.Lock()
	defer pos.mu.Unlock()
	
	if side == protocol.SideBuy {
		// Buying
		if pos.Quantity < 0 {
			// Reducing short position
			reduceQty := minInt64(qty, -pos.Quantity)
			realized := (price - pos.AvgPrice) * reduceQty / 1e8
			pos.RealizedPnL -= realized // Inverse for short
			pos.Quantity += reduceQty
			qty -= reduceQty
		}
		
		if qty > 0 {
			// Increasing long position or opening new long
			totalCost := pos.Quantity*pos.AvgPrice + qty*price
			pos.Quantity += qty
			if pos.Quantity != 0 {
				pos.AvgPrice = totalCost / pos.Quantity
			}
		}
		
		pos.TotalBought += qty
	} else {
		// Selling
		if pos.Quantity > 0 {
			// Reducing long position
			reduceQty := minInt64(qty, pos.Quantity)
			realized := (price - pos.AvgPrice) * reduceQty / 1e8
			pos.RealizedPnL += realized
			pos.Quantity -= reduceQty
			qty -= reduceQty
		}
		
		if qty > 0 {
			// Increasing short position or opening new short
			totalCost := pos.Quantity*pos.AvgPrice - qty*price
			pos.Quantity -= qty
			if pos.Quantity != 0 {
				pos.AvgPrice = totalCost / pos.Quantity
			}
		}
		
		pos.TotalSold += qty
	}
}

// GetPosition returns current position for a symbol
func (re *RiskEngine) GetPosition(symbol string) *Position {
	value, exists := re.positions.Load(symbol)
	if !exists {
		return &Position{Symbol: symbol}
	}
	
	pos := value.(*Position)
	pos.mu.RLock()
	defer pos.mu.RUnlock()
	
	// Return a copy
	return &Position{
		Symbol:      pos.Symbol,
		Quantity:    pos.Quantity,
		AvgPrice:    pos.AvgPrice,
		RealizedPnL: pos.RealizedPnL,
		TotalBought: pos.TotalBought,
		TotalSold:   pos.TotalSold,
	}
}

// GetAllPositions returns all current positions
func (re *RiskEngine) GetAllPositions() []*Position {
	positions := make([]*Position, 0)
	
	re.positions.Range(func(key, value interface{}) bool {
		pos := value.(*Position)
		pos.mu.RLock()
		positions = append(positions, &Position{
			Symbol:      pos.Symbol,
			Quantity:    pos.Quantity,
			AvgPrice:    pos.AvgPrice,
			RealizedPnL: pos.RealizedPnL,
			TotalBought: pos.TotalBought,
			TotalSold:   pos.TotalSold,
		})
		pos.mu.RUnlock()
		return true
	})
	
	return positions
}

// getOrCreatePosition gets or creates a position
func (re *RiskEngine) getOrCreatePosition(symbol string) *Position {
	value, _ := re.positions.LoadOrStore(symbol, &Position{Symbol: symbol})
	return value.(*Position)
}

// recordViolation records a risk violation
func (re *RiskEngine) recordViolation(order *matching.Order, violationType ViolationType, message string) {
	violation := &RiskViolation{
		Timestamp: time.Now().UnixNano(),
		OrderID:   order.ID,
		Symbol:    order.Symbol,
		Type:      violationType,
		Message:   message,
	}
	re.violations.Add(violation)
}

// Metrics returns risk engine metrics
func (re *RiskEngine) Metrics() RiskMetrics {
	total := atomic.LoadUint64(&re.checksTotal)
	rejected := atomic.LoadUint64(&re.checksRejected)
	latencySum := atomic.LoadUint64(&re.checksLatencySum)
	
	var avgLatencyNs uint64
	if total > 0 {
		avgLatencyNs = latencySum / total
	}
	
	return RiskMetrics{
		ChecksTotal:      total,
		ChecksRejected:   rejected,
		RejectRate:       float64(rejected) / float64(total) * 100.0,
		AvgLatencyNs:     avgLatencyNs,
		RecentViolations: re.violations.GetRecent(10),
	}
}

// RiskMetrics contains risk engine statistics
type RiskMetrics struct {
	ChecksTotal      uint64
	ChecksRejected   uint64
	RejectRate       float64
	AvgLatencyNs     uint64
	RecentViolations []*RiskViolation
}

// KillSwitch provides emergency trading halt
type KillSwitch struct {
	enabled     atomic.Bool
	reason      string
	triggeredAt int64
	mu          sync.RWMutex
}

// NewKillSwitch creates a new kill switch
func NewKillSwitch() *KillSwitch {
	return &KillSwitch{}
}

// Trigger activates the kill switch
func (ks *KillSwitch) Trigger(reason string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	
	ks.enabled.Store(true)
	ks.reason = reason
	ks.triggeredAt = time.Now().UnixNano()
}

// IsEnabled returns true if kill switch is active
func (ks *KillSwitch) IsEnabled() bool {
	return ks.enabled.Load()
}

// GetReason returns the kill switch reason
func (ks *KillSwitch) GetReason() (string, int64) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.reason, ks.triggeredAt
}

// Reset deactivates the kill switch
func (ks *KillSwitch) Reset() {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	
	ks.enabled.Store(false)
	ks.reason = ""
	ks.triggeredAt = 0
}

// Helper functions
func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func abs64(i int64) int64 {
	if i < 0 {
		return -i
	}
	return i
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
