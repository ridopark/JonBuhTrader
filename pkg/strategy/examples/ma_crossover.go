package examples

import (
	"sort"

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

	return &MovingAverageCrossoverStrategy{
		BaseStrategy: base,
		shortPeriod:  shortPeriod,
		longPeriod:   longPeriod,
		prices:       make([]float64, 0, longPeriod+1),
		position:     false,
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
	var potentialSignals []MACrossoverSignal
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
			// Collect potential buy signal
			potentialSignals = append(potentialSignals, MACrossoverSignal{
				Symbol:     symbol,
				Bar:        dataPoint.Bars[symbol],
				SignalType: "buy",
				Price:      dataPoint.Bars[symbol].Close,
				ShortMA:    s.currentShortMA,
				LongMA:     s.currentLongMA,
			})

			ctx.Log("debug", "MA Crossover potential BUY signal", map[string]interface{}{
				"symbol":  symbol,
				"price":   dataPoint.Bars[symbol].Close,
				"shortMA": s.currentShortMA,
				"longMA":  s.currentLongMA,
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
			}
			orders = append(orders, order)
			s.position = false

			ctx.Log("info", "Bearish crossover detected - selling", map[string]interface{}{
				"symbol":   symbol,
				"price":    dataPoint.Bars[symbol].Close,
				"quantity": position.Quantity,
				"shortMA":  s.currentShortMA,
				"longMA":   s.currentLongMA,
			})
		}
	}

	// Phase 2: Allocate capital to buy signals
	if len(potentialSignals) > 0 {
		buyOrders := s.allocateCapitalToSignals(ctx, potentialSignals)
		orders = append(orders, buyOrders...)
	}

	return orders, nil
}

// allocateCapitalToSignals prioritizes and allocates capital to trading signals
func (s *MovingAverageCrossoverStrategy) allocateCapitalToSignals(ctx strategy.Context, signals []MACrossoverSignal) []strategy.Order {
	if len(signals) == 0 {
		return nil
	}

	// Sort signals by symbol for deterministic ordering
	sort.Slice(signals, func(i, j int) bool {
		return signals[i].Symbol < signals[j].Symbol
	})

	var orders []strategy.Order
	availableCash := ctx.GetCash()

	ctx.Log("debug", "Allocating capital to MA Crossover signals", map[string]interface{}{
		"total_signals":  len(signals),
		"available_cash": availableCash,
	})

	for _, signal := range signals {
		if availableCash <= 0 {
			ctx.Log("debug", "No more cash available for allocation", map[string]interface{}{
				"remaining_signals": len(signals) - len(orders),
			})
			break
		}

		// Calculate position size based on current available cash (use 95% allocation)
		quantity := s.calculatePositionSize(availableCash, signal.Price, 0.95)
		if quantity > 0 {
			cost := quantity * signal.Price
			if cost <= availableCash {
				order := strategy.Order{
					Symbol:   signal.Symbol,
					Side:     strategy.OrderSideBuy,
					Type:     strategy.OrderTypeMarket,
					Quantity: quantity,
					Strategy: s.GetName(),
				}
				orders = append(orders, order)
				availableCash -= cost
				s.position = true // Update position state

				ctx.Log("info", "Bullish crossover detected - buying", map[string]interface{}{
					"symbol":         signal.Symbol,
					"price":          signal.Price,
					"quantity":       quantity,
					"cost":           cost,
					"shortMA":        signal.ShortMA,
					"longMA":         signal.LongMA,
					"remaining_cash": availableCash,
				})
			} else {
				ctx.Log("debug", "Insufficient cash for signal", map[string]interface{}{
					"symbol":         signal.Symbol,
					"required_cost":  cost,
					"available_cash": availableCash,
				})
			}
		}
	}

	ctx.Log("debug", "Capital allocation completed", map[string]interface{}{
		"orders_created": len(orders),
		"remaining_cash": availableCash,
	})

	return orders
}

// OnTrade handles trade execution notifications
func (s *MovingAverageCrossoverStrategy) OnTrade(ctx strategy.Context, trade strategy.TradeEvent) error {
	ctx.Log("info", "Trade executed", map[string]interface{}{
		"symbol":   trade.Symbol,
		"side":     string(trade.Side),
		"quantity": trade.Quantity,
		"price":    trade.Price,
		"strategy": s.GetName(),
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

// calculatePositionSize calculates position size based on available cash and allocation percentage
func (s *MovingAverageCrossoverStrategy) calculatePositionSize(cash, price, allocation float64) float64 {
	if cash <= 0 || price <= 0 || allocation <= 0 {
		return 0
	}

	// Calculate quantity based on allocation percentage
	targetValue := cash * allocation
	quantity := targetValue / price

	// Round down to nearest whole number (can't buy fractional shares)
	return float64(int(quantity))
}
