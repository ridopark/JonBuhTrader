package strategy

import (
	"math"
	"sort"
)

// TradingSignal represents a generic trading signal with priority and confidence
type TradingSignal interface {
	GetSymbol() string
	GetPrice() float64
	GetConfidence() float64
	GetSignalType() string
	GetBarData() BarData
	GetPriority() float64 // Higher values = higher priority
}

// AllocationMethod defines how capital should be allocated among signals
type AllocationMethod string

const (
	// AllocateEqually divides available capital equally among all signals
	AllocateEqually AllocationMethod = "equal"

	// AllocateByConfidence weights allocation by signal confidence
	AllocateByConfidence AllocationMethod = "confidence"

	// AllocateByPriority weights allocation by signal priority
	AllocateByPriority AllocationMethod = "priority"

	// AllocateSequential allocates to highest priority signals first until cash runs out
	AllocateSequential AllocationMethod = "sequential"
)

// AllocationConfig configures how capital allocation should work
type AllocationConfig struct {
	Method             AllocationMethod
	MaxPositions       int                         // Maximum number of positions to open simultaneously
	PositionSize       float64                     // Base position size as percentage of cash (0.0-1.0)
	MinCashBuffer      float64                     // Minimum cash to keep available (e.g., 100.0 for $100)
	SlippageBuffer     float64                     // Buffer for slippage/fees as percentage (e.g., 0.02 for 2%)
	AllowFractional    bool                        // Whether to allow fractional shares
	VolatilityAdjust   bool                        // Whether to adjust position size based on volatility
	VolatilityCallback func(symbol string) float64 // Function to get volatility for a symbol
}

// DefaultAllocationConfig returns a sensible default configuration
func DefaultAllocationConfig() AllocationConfig {
	return AllocationConfig{
		Method:           AllocateSequential,
		MaxPositions:     3,
		PositionSize:     0.95, // Use 95% of available cash
		MinCashBuffer:    100.0,
		SlippageBuffer:   0.02, // 2% buffer
		AllowFractional:  false,
		VolatilityAdjust: false,
	}
}

// CapitalAllocator handles capital allocation across multiple trading signals
type CapitalAllocator struct {
	config AllocationConfig
}

// NewCapitalAllocator creates a new capital allocator with the given configuration
func NewCapitalAllocator(config AllocationConfig) *CapitalAllocator {
	return &CapitalAllocator{
		config: config,
	}
}

// AllocateCapital allocates capital to trading signals and returns orders
func (ca *CapitalAllocator) AllocateCapital(ctx Context, signals []TradingSignal, strategyName string) []Order {
	if len(signals) == 0 {
		return nil
	}

	var orders []Order
	availableCash := ctx.GetCash()

	// Ensure we have minimum cash buffer
	if availableCash <= ca.config.MinCashBuffer {
		ctx.Log("warn", "Insufficient cash for trading", map[string]interface{}{
			"available_cash": availableCash,
			"min_buffer":     ca.config.MinCashBuffer,
		})
		return nil
	}

	// Apply slippage buffer
	tradableCash := availableCash * (1.0 - ca.config.SlippageBuffer)
	if tradableCash < ca.config.MinCashBuffer {
		return nil
	}

	// Limit number of signals if necessary
	maxSignals := len(signals)
	if ca.config.MaxPositions > 0 && maxSignals > ca.config.MaxPositions {
		maxSignals = ca.config.MaxPositions
	}

	// Sort signals based on allocation method
	sortedSignals := make([]TradingSignal, len(signals))
	copy(sortedSignals, signals)
	ca.sortSignals(sortedSignals)

	// Take only the top signals
	if maxSignals < len(sortedSignals) {
		sortedSignals = sortedSignals[:maxSignals]
	}

	ctx.Log("debug", "Allocating capital to signals", map[string]interface{}{
		"total_signals":     len(signals),
		"selected_signals":  len(sortedSignals),
		"available_cash":    availableCash,
		"tradable_cash":     tradableCash,
		"allocation_method": ca.config.Method,
		"max_positions":     ca.config.MaxPositions,
	})

	// Allocate capital based on method
	orders = ca.allocateByMethod(ctx, sortedSignals, tradableCash, strategyName)

	ctx.Log("debug", "Capital allocation completed", map[string]interface{}{
		"orders_created": len(orders),
		"method":         ca.config.Method,
	})

	return orders
}

// sortSignals sorts signals based on the allocation method
func (ca *CapitalAllocator) sortSignals(signals []TradingSignal) {
	switch ca.config.Method {
	case AllocateByConfidence, AllocateSequential:
		sort.Slice(signals, func(i, j int) bool {
			// First sort by confidence (higher first)
			if signals[i].GetConfidence() != signals[j].GetConfidence() {
				return signals[i].GetConfidence() > signals[j].GetConfidence()
			}
			// Then by priority as tiebreaker
			return signals[i].GetPriority() > signals[j].GetPriority()
		})
	case AllocateByPriority:
		sort.Slice(signals, func(i, j int) bool {
			// First sort by priority (higher first)
			if signals[i].GetPriority() != signals[j].GetPriority() {
				return signals[i].GetPriority() > signals[j].GetPriority()
			}
			// Then by confidence as tiebreaker
			return signals[i].GetConfidence() > signals[j].GetConfidence()
		})
	case AllocateEqually:
		// No sorting needed for equal allocation
	}
}

// allocateByMethod allocates capital using the configured method
func (ca *CapitalAllocator) allocateByMethod(ctx Context, signals []TradingSignal, tradableCash float64, strategyName string) []Order {
	switch ca.config.Method {
	case AllocateEqually:
		return ca.allocateEqually(ctx, signals, tradableCash, strategyName)
	case AllocateByConfidence:
		return ca.allocateByConfidence(ctx, signals, tradableCash, strategyName)
	case AllocateByPriority:
		return ca.allocateByPriority(ctx, signals, tradableCash, strategyName)
	case AllocateSequential:
		return ca.allocateSequential(ctx, signals, tradableCash, strategyName)
	default:
		return ca.allocateSequential(ctx, signals, tradableCash, strategyName)
	}
}

// allocateEqually divides cash equally among all signals
func (ca *CapitalAllocator) allocateEqually(ctx Context, signals []TradingSignal, tradableCash float64, strategyName string) []Order {
	var orders []Order
	allocationPerSignal := (tradableCash * ca.config.PositionSize) / float64(len(signals))

	for _, signal := range signals {
		quantity := ca.calculatePositionSize(signal, allocationPerSignal)
		if quantity > 0 {
			cost := quantity * signal.GetPrice()

			order := Order{
				Symbol:   signal.GetSymbol(),
				Side:     OrderSideBuy,
				Type:     OrderTypeMarket,
				Quantity: quantity,
				Strategy: strategyName,
				Reason:   signal.GetSignalType(),
			}
			orders = append(orders, order)

			ctx.Log("info", "Equal allocation trade", map[string]interface{}{
				"symbol":     signal.GetSymbol(),
				"price":      signal.GetPrice(),
				"quantity":   quantity,
				"cost":       cost,
				"allocation": allocationPerSignal,
				"confidence": signal.GetConfidence(),
				"reason":     signal.GetSignalType(),
			})
		}
	}
	return orders
}

// allocateByConfidence weights allocation by signal confidence
func (ca *CapitalAllocator) allocateByConfidence(ctx Context, signals []TradingSignal, tradableCash float64, strategyName string) []Order {
	var orders []Order
	totalConfidence := 0.0
	for _, signal := range signals {
		totalConfidence += signal.GetConfidence()
	}

	if totalConfidence == 0 {
		return ca.allocateEqually(ctx, signals, tradableCash, strategyName)
	}

	remainingCash := tradableCash * ca.config.PositionSize

	for i, signal := range signals {
		if remainingCash <= ca.config.MinCashBuffer {
			break
		}

		var allocation float64
		if i == len(signals)-1 {
			// Last signal gets whatever is left
			allocation = remainingCash
		} else {
			// Proportional allocation based on confidence
			confidenceWeight := signal.GetConfidence() / totalConfidence
			allocation = tradableCash * ca.config.PositionSize * confidenceWeight
			allocation = math.Min(allocation, remainingCash)
		}

		quantity := ca.calculatePositionSize(signal, allocation)
		if quantity > 0 {
			cost := quantity * signal.GetPrice()
			if cost <= remainingCash {
				order := Order{
					Symbol:   signal.GetSymbol(),
					Side:     OrderSideBuy,
					Type:     OrderTypeMarket,
					Quantity: quantity,
					Strategy: strategyName,
					Reason:   signal.GetSignalType(),
				}
				orders = append(orders, order)
				remainingCash -= cost

				ctx.Log("info", "Confidence-weighted allocation trade", map[string]interface{}{
					"symbol":         signal.GetSymbol(),
					"price":          signal.GetPrice(),
					"quantity":       quantity,
					"cost":           cost,
					"allocation":     allocation,
					"confidence":     signal.GetConfidence(),
					"remaining_cash": remainingCash,
					"reason":         signal.GetSignalType(),
				})
			}
		}
	}
	return orders
}

// allocateByPriority weights allocation by signal priority
func (ca *CapitalAllocator) allocateByPriority(ctx Context, signals []TradingSignal, tradableCash float64, strategyName string) []Order {
	var orders []Order
	totalPriority := 0.0
	for _, signal := range signals {
		totalPriority += signal.GetPriority()
	}

	if totalPriority == 0 {
		return ca.allocateEqually(ctx, signals, tradableCash, strategyName)
	}

	remainingCash := tradableCash * ca.config.PositionSize

	for i, signal := range signals {
		if remainingCash <= ca.config.MinCashBuffer {
			break
		}

		var allocation float64
		if i == len(signals)-1 {
			// Last signal gets whatever is left
			allocation = remainingCash
		} else {
			// Proportional allocation based on priority
			priorityWeight := signal.GetPriority() / totalPriority
			allocation = tradableCash * ca.config.PositionSize * priorityWeight
			allocation = math.Min(allocation, remainingCash)
		}

		quantity := ca.calculatePositionSize(signal, allocation)
		if quantity > 0 {
			cost := quantity * signal.GetPrice()
			if cost <= remainingCash {
				order := Order{
					Symbol:   signal.GetSymbol(),
					Side:     OrderSideBuy,
					Type:     OrderTypeMarket,
					Quantity: quantity,
					Strategy: strategyName,
					Reason:   signal.GetSignalType(),
				}
				orders = append(orders, order)
				remainingCash -= cost

				ctx.Log("info", "Priority-weighted allocation trade", map[string]interface{}{
					"symbol":         signal.GetSymbol(),
					"price":          signal.GetPrice(),
					"quantity":       quantity,
					"cost":           cost,
					"allocation":     allocation,
					"priority":       signal.GetPriority(),
					"remaining_cash": remainingCash,
					"reason":         signal.GetSignalType(),
				})
			}
		}
	}
	return orders
}

// allocateSequential allocates to highest priority signals until cash runs out
func (ca *CapitalAllocator) allocateSequential(ctx Context, signals []TradingSignal, tradableCash float64, strategyName string) []Order {
	var orders []Order
	remainingCash := tradableCash

	for _, signal := range signals {
		if remainingCash <= ca.config.MinCashBuffer {
			ctx.Log("debug", "Insufficient remaining cash for more signals", map[string]interface{}{
				"remaining_cash": remainingCash,
				"min_buffer":     ca.config.MinCashBuffer,
			})
			break
		}

		// Calculate position size based on remaining cash
		allocation := math.Min(ca.config.PositionSize, remainingCash/tradableCash)
		quantity := ca.calculatePositionSize(signal, remainingCash*allocation)

		if quantity > 0 {
			cost := quantity * signal.GetPrice()
			if cost <= remainingCash {
				order := Order{
					Symbol:   signal.GetSymbol(),
					Side:     OrderSideBuy,
					Type:     OrderTypeMarket,
					Quantity: quantity,
					Strategy: strategyName,
					Reason:   signal.GetSignalType(),
				}
				orders = append(orders, order)
				remainingCash -= cost

				ctx.Log("info", "Sequential allocation trade", map[string]interface{}{
					"symbol":         signal.GetSymbol(),
					"price":          signal.GetPrice(),
					"quantity":       quantity,
					"cost":           cost,
					"allocation":     allocation,
					"confidence":     signal.GetConfidence(),
					"remaining_cash": remainingCash,
					"reason":         signal.GetSignalType(),
				})
			} else {
				ctx.Log("debug", "Insufficient cash for signal", map[string]interface{}{
					"symbol":         signal.GetSymbol(),
					"required_cost":  cost,
					"remaining_cash": remainingCash,
				})
			}
		}
	}
	return orders
}

// calculatePositionSize calculates the position size for a signal
func (ca *CapitalAllocator) calculatePositionSize(signal TradingSignal, allocation float64) float64 {
	if allocation <= 0 || signal.GetPrice() <= 0 {
		return 0
	}

	quantity := allocation / signal.GetPrice()

	// Apply volatility adjustment if enabled
	if ca.config.VolatilityAdjust && ca.config.VolatilityCallback != nil {
		volatility := ca.config.VolatilityCallback(signal.GetSymbol())
		volatilityAdjustment := ca.getVolatilityAdjustment(volatility)
		quantity *= volatilityAdjustment
	}

	// Handle fractional shares
	if !ca.config.AllowFractional {
		quantity = math.Floor(quantity)
	}

	return math.Max(0, quantity)
}

// getVolatilityAdjustment returns a position size adjustment based on volatility
func (ca *CapitalAllocator) getVolatilityAdjustment(volatility float64) float64 {
	// Reduce position size in high volatility environments
	if volatility > 0.03 { // 3% daily volatility
		return 0.7 // Reduce to 70% of normal size
	} else if volatility > 0.02 { // 2% daily volatility
		return 0.85 // Reduce to 85% of normal size
	}
	return 1.0 // No adjustment
}
