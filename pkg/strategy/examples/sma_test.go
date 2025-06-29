package examples

import (
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// SMATestStrategy is a simple strategy that uses SMA for testing
type SMATestStrategy struct {
	*strategy.BaseStrategy
	smaPeriod int
	symbols   []string
}

// NewSMATestStrategy creates a new SMA test strategy
func NewSMATestStrategy(symbols []string, smaPeriod int) *SMATestStrategy {
	params := map[string]interface{}{
		"symbols":   symbols,
		"smaPeriod": smaPeriod,
	}

	base := strategy.NewBaseStrategy("SMATest", params)
	base.SetSymbols(symbols)

	return &SMATestStrategy{
		BaseStrategy: base,
		smaPeriod:    smaPeriod,
		symbols:      symbols,
	}
}

// Initialize sets up the strategy
func (s *SMATestStrategy) Initialize(ctx strategy.Context) error {
	ctx.Log("info", "SMA Test Strategy initialized", map[string]interface{}{
		"smaPeriod": s.smaPeriod,
		"symbols":   s.symbols,
	})
	return s.BaseStrategy.Initialize(ctx)
}

// OnDataPoint processes each new datapoint and tests SMA calculation
func (s *SMATestStrategy) OnDataPoint(ctx strategy.Context, dataPoint strategy.DataPoint) ([]strategy.Order, error) {
	orders := make([]strategy.Order, 0)

	for _, symbol := range s.symbols {
		// Try to calculate SMA
		sma, err := ctx.SMA(symbol, s.smaPeriod)
		if err != nil {
			// SMA not ready yet (not enough data), just log debug
			ctx.Log("debug", "SMA not ready", map[string]interface{}{
				"symbol": symbol,
				"error":  err.Error(),
			})
		} else {
			// SMA calculated successfully, log the value
			ctx.Log("info", "SMA calculated", map[string]interface{}{
				"symbol":       symbol,
				"sma":          sma,
				"currentPrice": dataPoint.Bars[symbol].Close,
				"timestamp":    dataPoint.Timestamp,
			})

			// Simple strategy: buy if price is above SMA, sell if below
			position := ctx.GetPosition(symbol)
			currentPrice := dataPoint.Bars[symbol].Close

			if currentPrice > sma && (position == nil || position.Quantity == 0) {
				// Price above SMA and no position - buy
				cash := ctx.GetCash()
				quantity := (cash * 0.5) / currentPrice // Use 50% of cash
				if quantity >= 1 {
					order := s.CreateMarketOrder(symbol, strategy.OrderSideBuy, quantity)
					orders = append(orders, order)
					ctx.Log("info", "Buy signal: price above SMA", map[string]interface{}{
						"symbol":   symbol,
						"price":    currentPrice,
						"sma":      sma,
						"quantity": quantity,
					})
				}
			} else if currentPrice < sma && position != nil && position.Quantity > 0 {
				// Price below SMA and have position - sell
				order := s.CreateMarketOrder(symbol, strategy.OrderSideSell, position.Quantity)
				orders = append(orders, order)
				ctx.Log("info", "Sell signal: price below SMA", map[string]interface{}{
					"symbol":   symbol,
					"price":    currentPrice,
					"sma":      sma,
					"quantity": position.Quantity,
				})
			}
		}
	}

	return orders, nil
}

// OnTrade handles trade executions
func (s *SMATestStrategy) OnTrade(ctx strategy.Context, trade strategy.TradeEvent) error {
	ctx.Log("info", "Trade executed", map[string]interface{}{
		"symbol":   trade.Symbol,
		"side":     trade.Side,
		"quantity": trade.Quantity,
		"price":    trade.Price,
	})

	return s.BaseStrategy.OnTrade(ctx, trade)
}
