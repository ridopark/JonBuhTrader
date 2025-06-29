package examples

import (
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// RSIStrategy implements a simple RSI-based trading strategy
type RSIStrategy struct {
	symbols   []string
	rsiPeriod int
	buyLevel  float64 // RSI level to buy (oversold)
	sellLevel float64 // RSI level to sell (overbought)
	posSize   float64 // Position size as fraction of portfolio
}

// NewRSIStrategy creates a new RSI strategy
func NewRSIStrategy(symbols []string, rsiPeriod int, buyLevel, sellLevel, posSize float64) *RSIStrategy {
	return &RSIStrategy{
		symbols:   symbols,
		rsiPeriod: rsiPeriod,
		buyLevel:  buyLevel,
		sellLevel: sellLevel,
		posSize:   posSize,
	}
}

// Initialize is called before the strategy starts running
func (s *RSIStrategy) Initialize(ctx strategy.Context) error {
	ctx.Log("info", "RSI Strategy initialized", map[string]interface{}{
		"symbols":   s.symbols,
		"rsiPeriod": s.rsiPeriod,
		"buyLevel":  s.buyLevel,
		"sellLevel": s.sellLevel,
		"posSize":   s.posSize,
	})
	return nil
}

// OnData is called for each new data point
func (s *RSIStrategy) OnData(ctx strategy.Context, dataPoint strategy.DataPoint) error {
	for _, symbol := range s.symbols {
		bar, exists := dataPoint.Bars[symbol]
		if !exists {
			continue
		}

		// Get current RSI
		rsi, err := ctx.RSI(symbol, s.rsiPeriod)
		if err != nil {
			// Not enough data yet, skip
			continue
		}

		// Get current position
		position := ctx.GetPosition(symbol)
		cash := ctx.GetCash()

		ctx.Log("debug", "RSI analysis", map[string]interface{}{
			"symbol":   symbol,
			"price":    bar.Close,
			"rsi":      rsi,
			"position": position.Quantity,
			"cash":     cash,
		})

		// RSI oversold condition - consider buying
		if rsi <= s.buyLevel && position.Quantity == 0 {
			// Calculate position size
			positionValue := cash * s.posSize
			quantity := int(positionValue / bar.Close)

			if quantity > 0 {
				ctx.Log("info", "RSI Buy Signal", map[string]interface{}{
					"symbol":   symbol,
					"price":    bar.Close,
					"rsi":      rsi,
					"quantity": quantity,
					"reason":   "RSI oversold",
				})

				// Place buy order (simplified - market order)
				// In a real implementation, you'd use ctx.PlaceBuyOrder or similar
			}
		}

		// RSI overbought condition - consider selling
		if rsi >= s.sellLevel && position.Quantity > 0 {
			ctx.Log("info", "RSI Sell Signal", map[string]interface{}{
				"symbol":   symbol,
				"price":    bar.Close,
				"rsi":      rsi,
				"quantity": position.Quantity,
				"reason":   "RSI overbought",
			})

			// Place sell order (simplified - market order)
			// In a real implementation, you'd use ctx.PlaceSellOrder or similar
		}
	}

	return nil
}

// OnFinish is called when the strategy finishes
func (s *RSIStrategy) OnFinish(ctx strategy.Context) error {
	ctx.Log("info", "RSI Strategy finished", map[string]interface{}{
		"finalCash": ctx.GetCash(),
	})
	return nil
}

// GetName returns the strategy name
func (s *RSIStrategy) GetName() string {
	return "RSI Strategy"
}
