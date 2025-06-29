package examples

import (
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// MACrossoverSignal represents a potential trading signal
type MACrossoverSignal struct {
	Symbol     string
	Bar        strategy.BarData
	SignalType string // "buy" or "sell"
	Price      float64
	ShortMA    float64
	LongMA     float64
}

// MovingAverageCrossoverStrategy implements a simple moving average crossover strategy
type MovingAverageCrossoverStrategy struct {
	*strategy.BaseStrategy
	shortPeriod    int
	longPeriod     int
	prices         []float64
	position       bool // true if long, false if flat
	lastShortMA    float64
	lastLongMA     float64
	currentShortMA float64
	currentLongMA  float64
	allocator      *strategy.CapitalAllocator // Capital allocation system
}

// NewMovingAverageCrossoverStrategy creates a new moving average crossover strategy
func NewMovingAverageCrossoverStrategy(shortPeriod, longPeriod int) *MovingAverageCrossoverStrategy {
	if shortPeriod >= longPeriod {
		panic("short period must be less than long period")
	}

	base := strategy.NewBaseStrategy("MovingAverageCrossover", map[string]interface{}{
		"shortPeriod": shortPeriod,
		"longPeriod":  longPeriod,
	})

	// Configure capital allocation
	allocConfig := strategy.DefaultAllocationConfig()
	allocConfig.Method = strategy.AllocateSequential
	allocConfig.PositionSize = 0.95 // Use 95% of available cash
	allocConfig.MaxPositions = 3    // Max 3 positions

	return &MovingAverageCrossoverStrategy{
		BaseStrategy: base,
		shortPeriod:  shortPeriod,
		longPeriod:   longPeriod,
		prices:       make([]float64, 0, longPeriod+1),
		position:     false,
		allocator:    strategy.NewCapitalAllocator(allocConfig),
	}
}

// SetSymbols sets the symbols for this strategy (called by the engine)
func (s *MovingAverageCrossoverStrategy) SetSymbols(symbols []string) {
	s.BaseStrategy.SetSymbols(symbols)
}

// Initialize sets up the strategy
func (s *MovingAverageCrossoverStrategy) Initialize(ctx strategy.Context) error {
	ctx.Log("info", "Strategy initialized", map[string]interface{}{
		"strategy":    s.GetName(),
		"shortPeriod": s.shortPeriod,
		"longPeriod":  s.longPeriod,
	})
	return nil
}

// OnBar processes each bar and generates trading signals
func (s *MovingAverageCrossoverStrategy) OnDataPoint(ctx strategy.Context, dataPoint strategy.DataPoint) ([]strategy.Order, error) {
	var potentialSignals []strategy.TradingSignal
	var orders []strategy.Order

	// Phase 1: Analyze all symbols and collect potential buy signals
	for _, symbol := range s.GetSymbols() {
		// Add current price to our price history
		s.prices = append(s.prices, dataPoint.Bars[symbol].Close)

		// Keep only the data we need (longPeriod + 1 for crossover detection)
		if len(s.prices) > s.longPeriod+1 {
			s.prices = s.prices[1:]
		}

		ctx.Log("debug", "Price history updated", map[string]interface{}{
			"symbol":        symbol,
			"price":         dataPoint.Bars[symbol].Close,
			"history_count": len(s.prices),
			"need_count":    s.longPeriod,
		})

		// Need at least longPeriod prices to calculate moving averages
		if len(s.prices) < s.longPeriod {
			// Test the context SMA function even with limited data
			if len(s.prices) >= s.shortPeriod {
				contextShortSMA, err := ctx.SMA(symbol, s.shortPeriod)
				if err == nil {
					internalSMA := s.calculateSMA(s.shortPeriod)
					ctx.Log("debug", "SMA comparison (early)", map[string]interface{}{
						"symbol":       symbol,
						"internal_sma": internalSMA,
						"context_sma":  contextShortSMA,
						"price":        dataPoint.Bars[symbol].Close,
						"period":       s.shortPeriod,
						"data_points":  len(s.prices),
					})
				} else {
					ctx.Log("debug", "Context SMA error", map[string]interface{}{
						"symbol": symbol,
						"error":  err.Error(),
						"period": s.shortPeriod,
					})
				}
			}
			continue
		}

		// Calculate moving averages
		s.lastShortMA = s.currentShortMA
		s.lastLongMA = s.currentLongMA

		s.currentShortMA = s.calculateSMA(s.shortPeriod)
		s.currentLongMA = s.calculateSMA(s.longPeriod)

		// Test the context SMA function (for comparison)
		contextShortSMA, err := ctx.SMA(symbol, s.shortPeriod)
		if err == nil {
			ctx.Log("debug", "SMA comparison", map[string]interface{}{
				"symbol":       symbol,
				"internal_sma": s.currentShortMA,
				"context_sma":  contextShortSMA,
				"price":        dataPoint.Bars[symbol].Close,
				"period":       s.shortPeriod,
			})
		}

		// Need at least one previous calculation for crossover detection
		if s.lastShortMA == 0 || s.lastLongMA == 0 {
			continue
		}

		// Check for crossover signals
		prevCross := s.lastShortMA > s.lastLongMA
		currentCross := s.currentShortMA > s.currentLongMA

		position := ctx.GetPosition(symbol)

		// Bullish crossover: short MA crosses above long MA
		if !prevCross && currentCross && !s.position {
			// Calculate confidence based on the strength of the crossover
			confidence := s.calculateCrossoverConfidence(s.currentShortMA, s.currentLongMA)

			// Collect potential buy signal
			potentialSignals = append(potentialSignals, MACrossoverSignalImpl{
				Symbol:     symbol,
				Bar:        dataPoint.Bars[symbol],
				SignalType: "bullish_crossover",
				Price:      dataPoint.Bars[symbol].Close,
				ShortMA:    s.currentShortMA,
				LongMA:     s.currentLongMA,
				Confidence: confidence,
				Priority:   confidence, // Use confidence as priority
			})

			ctx.Log("debug", "MA Crossover potential BUY signal", map[string]interface{}{
				"symbol":     symbol,
				"price":      dataPoint.Bars[symbol].Close,
				"shortMA":    s.currentShortMA,
				"longMA":     s.currentLongMA,
				"confidence": confidence,
			})
		}

		// Bearish crossover: short MA crosses below long MA
		if prevCross && !currentCross && s.position && position != nil && position.Quantity > 0 {
			// Sell signal (immediate execution)
			order := strategy.Order{
				Symbol:   symbol,
				Side:     strategy.OrderSideSell,
				Type:     strategy.OrderTypeMarket,
				Quantity: position.Quantity,
				Strategy: s.GetName(),
				Reason:   "bearish_crossover",
			}
			orders = append(orders, order)
			s.position = false

			ctx.Log("info", "Bearish crossover detected - selling", map[string]interface{}{
				"symbol":   symbol,
				"price":    dataPoint.Bars[symbol].Close,
				"quantity": position.Quantity,
				"shortMA":  s.currentShortMA,
				"longMA":   s.currentLongMA,
				"reason":   "bearish_crossover",
			})
		}
	}

	// Phase 2: Allocate capital to buy signals using the common allocation system
	if len(potentialSignals) > 0 {
		buyOrders := s.allocator.AllocateCapital(ctx, potentialSignals, s.GetName())
		orders = append(orders, buyOrders...)
	}

	return orders, nil
}

// OnTrade handles trade execution notifications
func (s *MovingAverageCrossoverStrategy) OnTrade(ctx strategy.Context, trade strategy.TradeEvent) error {
	ctx.Log("info", "Trade executed", map[string]interface{}{
		"symbol":   trade.Symbol,
		"side":     string(trade.Side),
		"quantity": trade.Quantity,
		"price":    trade.Price,
		"strategy": s.GetName(),
		"reason":   trade.Reason,
	})
	return nil
}

// Cleanup performs strategy cleanup
func (s *MovingAverageCrossoverStrategy) Cleanup(ctx strategy.Context) error {
	ctx.Log("info", "Strategy cleanup", map[string]interface{}{
		"strategy": s.GetName(),
	})
	return nil
}

// GetParameters returns the strategy parameters
func (s *MovingAverageCrossoverStrategy) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"shortPeriod": s.shortPeriod,
		"longPeriod":  s.longPeriod,
	}
}

// calculateSMA calculates simple moving average for the given period
func (s *MovingAverageCrossoverStrategy) calculateSMA(period int) float64 {
	if len(s.prices) < period {
		return 0
	}

	sum := 0.0
	start := len(s.prices) - period
	for i := start; i < len(s.prices); i++ {
		sum += s.prices[i]
	}
	return sum / float64(period)
}

// calculateCrossoverConfidence calculates confidence based on the strength of the crossover
func (s *MovingAverageCrossoverStrategy) calculateCrossoverConfidence(shortMA, longMA float64) float64 {
	// Simple confidence calculation based on the gap between moving averages
	// Larger gaps indicate stronger signals
	gap := (shortMA - longMA) / longMA
	confidence := 0.5 + (gap * 10) // Base confidence of 0.5, adjust based on gap

	// Clamp confidence between 0.1 and 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.1 {
		confidence = 0.1
	}

	return confidence
}
