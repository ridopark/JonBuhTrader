package examples

import (
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

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
	var orders []strategy.Order

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
			return orders, nil
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
			return orders, nil
		}

		// Check for crossover signals
		prevCross := s.lastShortMA > s.lastLongMA
		currentCross := s.currentShortMA > s.currentLongMA

		position := ctx.GetPosition(symbol)
		cash := ctx.GetCash()

		// Bullish crossover: short MA crosses above long MA
		if !prevCross && currentCross && !s.position {
			// Buy signal
			quantity := s.calculatePositionSize(cash, dataPoint.Bars[symbol].Close, 0.95) // Use 95% of available cash
			if quantity > 0 {
				order := strategy.Order{
					Symbol:   symbol,
					Side:     strategy.OrderSideBuy,
					Type:     strategy.OrderTypeMarket,
					Quantity: quantity,
					Strategy: s.GetName(),
				}
				orders = append(orders, order)
				s.position = true

				ctx.Log("info", "Bullish crossover detected - buying", map[string]interface{}{
					"symbol":   symbol,
					"price":    dataPoint.Bars[symbol].Close,
					"quantity": quantity,
					"shortMA":  s.currentShortMA,
					"longMA":   s.currentLongMA,
				})
			}
		}

		// Bearish crossover: short MA crosses below long MA
		if prevCross && !currentCross && s.position && position != nil && position.Quantity > 0 {
			// Sell signal
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

	// Round down to avoid insufficient funds
	return float64(int(quantity*100)) / 100 // Round to 2 decimal places
}
