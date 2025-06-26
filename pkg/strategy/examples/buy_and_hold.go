package examples

import (
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// BuyAndHoldStrategy is a simple buy-and-hold strategy for testing
type BuyAndHoldStrategy struct {
	*strategy.BaseStrategy
	hasBought bool
}

// NewBuyAndHoldStrategy creates a new buy-and-hold strategy
func NewBuyAndHoldStrategy() *BuyAndHoldStrategy {
	params := map[string]interface{}{
		"symbol": "AAPL",
	}

	base := strategy.NewBaseStrategy("BuyAndHold", params)

	return &BuyAndHoldStrategy{
		BaseStrategy: base,
		hasBought:    false,
	}
}

// Initialize sets up the strategy
func (s *BuyAndHoldStrategy) Initialize(ctx strategy.Context) error {
	s.hasBought = false
	return s.BaseStrategy.Initialize(ctx)
}

// OnBar processes each new bar of data
func (s *BuyAndHoldStrategy) OnBar(ctx strategy.Context, bar strategy.BarData) ([]strategy.Order, error) {
	orders := make([]strategy.Order, 0)

	// Only buy once at the beginning
	if !s.hasBought {
		cash := ctx.GetCash()
		if cash > 0 {
			// Buy as much as possible with available cash
			quantity := cash / bar.Close * 0.95 // Use 95% of cash to leave some buffer

			if quantity > 0 {
				order := s.CreateMarketOrder(bar.Symbol, strategy.OrderSideBuy, quantity)
				orders = append(orders, order)
				s.hasBought = true

				ctx.Log("info", "Buying shares", map[string]interface{}{
					"symbol":   bar.Symbol,
					"quantity": quantity,
					"price":    bar.Close,
				})
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
