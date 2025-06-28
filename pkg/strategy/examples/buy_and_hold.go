package examples

import (
	"math"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// BuyAndHoldStrategy is a simple buy-and-hold strategy for testing
type BuyAndHoldStrategy struct {
	*strategy.BaseStrategy
	hasBought      map[string]bool
	initialCapital float64
}

// NewBuyAndHoldStrategy creates a new buy-and-hold strategy
func NewBuyAndHoldStrategy(symbols []string, cash float64) *BuyAndHoldStrategy {
	params := map[string]interface{}{
		"symbols": symbols,
	}

	base := strategy.NewBaseStrategy("BuyAndHold", params)
	base.SetSymbols(symbols)

	return &BuyAndHoldStrategy{
		BaseStrategy:   base,
		hasBought:      make(map[string]bool),
		initialCapital: cash,
	}
}

// Initialize sets up the strategy
func (s *BuyAndHoldStrategy) Initialize(ctx strategy.Context) error {
	// Loop through all symbols and set hasBought to false for each
	for _, symbol := range s.GetSymbols() {
		s.hasBought[symbol] = false
	}
	return s.BaseStrategy.Initialize(ctx)
}

// OnBar processes each new bar of data
func (s *BuyAndHoldStrategy) OnDataPoint(ctx strategy.Context, dataPoint strategy.DataPoint) ([]strategy.Order, error) {
	orders := make([]strategy.Order, 0)

	for _, symbol := range s.GetSymbols() {
		// Only buy once at the beginning
		if !s.hasBought[symbol] {
			cash := ctx.GetCash()
			numSymbols := len(s.GetSymbols())
			if cash > 0 {
				// Buy as much as possible with available cash divided equally among symbols
				quantity := math.Floor(s.initialCapital / (float64(numSymbols) * dataPoint.Bars[symbol].Close))

				if quantity > 0 {
					order := s.CreateMarketOrder(symbol, strategy.OrderSideBuy, quantity)
					orders = append(orders, order)
					s.hasBought[symbol] = true

					ctx.Log("info", "Buying shares", map[string]interface{}{
						"symbol":   symbol,
						"quantity": quantity,
						"price":    dataPoint.Bars[symbol].Close,
					})
				}
			}
		}
	}

	return orders, nil
}

// OnTrade handles trade executions
func (s *BuyAndHoldStrategy) OnTrade(ctx strategy.Context, trade strategy.TradeEvent) error {
	ctx.Log("info", "Trade executed", map[string]interface{}{
		"symbol":   trade.Symbol,
		"side":     trade.Side,
		"quantity": trade.Quantity,
		"price":    trade.Price,
	})

	return s.BaseStrategy.OnTrade(ctx, trade)
}
