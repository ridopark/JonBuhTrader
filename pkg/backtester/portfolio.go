package backtester

import (
	"math"
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// Portfolio manages positions, cash, and P&L tracking
type Portfolio struct {
	cash             float64
	initialCash      float64
	positions        map[string]*strategy.Position
	trades           []strategy.TradeEvent
	totalValue       float64
	commissionConfig *CommissionConfig

	// Performance tracking
	dailyReturns    []float64
	equity          []EquityPoint
	maxDrawdown     float64
	currentDrawdown float64
	peakValue       float64
}

// EquityPoint represents equity at a point in time
type EquityPoint struct {
	Timestamp time.Time
	Value     float64
}

// NewPortfolio creates a new portfolio with the given initial capital
func NewPortfolio(initialCapital float64, commissionConfig *CommissionConfig) *Portfolio {
	return &Portfolio{
		cash:             initialCapital,
		initialCash:      initialCapital,
		positions:        make(map[string]*strategy.Position),
		trades:           make([]strategy.TradeEvent, 0),
		totalValue:       initialCapital,
		commissionConfig: commissionConfig,
		equity:           make([]EquityPoint, 0),
		peakValue:        initialCapital,
	}
}

// GetCash returns the current cash balance
func (p *Portfolio) GetCash() float64 {
	return p.cash
}

// GetPosition returns the position for a symbol, or nil if no position exists
func (p *Portfolio) GetPosition(symbol string) *strategy.Position {
	return p.positions[symbol]
}

// GetPositions returns all positions
func (p *Portfolio) GetPositions() map[string]*strategy.Position {
	return p.positions
}

// GetTrades returns all trades
func (p *Portfolio) GetTrades() []strategy.TradeEvent {
	return p.trades
}

// GetTotalValue returns the total portfolio value
func (p *Portfolio) GetTotalValue() float64 {
	return p.totalValue
}

// GetTotalPL returns the total profit/loss
func (p *Portfolio) GetTotalPL() float64 {
	return p.totalValue - p.initialCash
}

// GetTotalReturn returns the total return as a percentage
func (p *Portfolio) GetTotalReturn() float64 {
	return (p.totalValue - p.initialCash) / p.initialCash * 100
}

// ExecuteTrade processes a trade and updates the portfolio
func (p *Portfolio) ExecuteTrade(trade strategy.TradeEvent, currentPrice float64) error {
	symbol := trade.Symbol

	// Get or create position
	position, exists := p.positions[symbol]
	if !exists {
		position = &strategy.Position{
			Symbol:   symbol,
			Quantity: 0,
			AvgPrice: 0,
		}
		p.positions[symbol] = position
	}

	// Calculate trade value including all fees
	tradeValue := trade.Quantity * trade.Price
	totalFees := trade.Commission + trade.SecFee + trade.FinraTaf + trade.Slippage
	totalCost := tradeValue + totalFees

	// Update position based on trade side
	if trade.Side == strategy.OrderSideBuy {
		if position.Quantity >= 0 {
			// Adding to long position or opening new long position
			newQuantity := position.Quantity + trade.Quantity
			position.AvgPrice = ((position.AvgPrice * position.Quantity) + (trade.Price * trade.Quantity)) / newQuantity
			position.Quantity = newQuantity
			p.cash -= totalCost
		} else {
			// Covering short position
			if math.Abs(trade.Quantity) <= math.Abs(position.Quantity) {
				// Partial or full cover
				realizedPL := (position.AvgPrice - trade.Price) * trade.Quantity
				position.RealizedPL += realizedPL
				position.Quantity += trade.Quantity
				p.cash -= totalCost

				if position.Quantity == 0 {
					position.AvgPrice = 0
				}
			} else {
				// Cover and reverse
				coverQuantity := math.Abs(position.Quantity)
				realizedPL := (position.AvgPrice - trade.Price) * coverQuantity
				position.RealizedPL += realizedPL

				// New long position
				newLongQuantity := trade.Quantity - coverQuantity
				position.Quantity = newLongQuantity
				position.AvgPrice = trade.Price
				p.cash -= totalCost
			}
		}
	} else { // SELL
		if position.Quantity <= 0 {
			// Adding to short position or opening new short position
			newQuantity := position.Quantity - trade.Quantity
			if position.Quantity == 0 {
				position.AvgPrice = trade.Price
			} else {
				position.AvgPrice = ((position.AvgPrice * math.Abs(position.Quantity)) + (trade.Price * trade.Quantity)) / math.Abs(newQuantity)
			}
			position.Quantity = newQuantity
			p.cash += tradeValue - totalFees
		} else {
			// Selling long position
			if trade.Quantity <= position.Quantity {
				// Partial or full sale
				realizedPL := (trade.Price - position.AvgPrice) * trade.Quantity
				position.RealizedPL += realizedPL
				position.Quantity -= trade.Quantity
				p.cash += tradeValue - totalFees

				if position.Quantity == 0 {
					position.AvgPrice = 0
				}
			} else {
				// Sell and reverse
				sellQuantity := position.Quantity
				realizedPL := (trade.Price - position.AvgPrice) * sellQuantity
				position.RealizedPL += realizedPL

				// New short position
				newShortQuantity := trade.Quantity - sellQuantity
				position.Quantity = -newShortQuantity
				position.AvgPrice = trade.Price
				p.cash += tradeValue - totalFees
			}
		}
	}

	// Update market value and unrealized P&L
	position.MarketValue = position.MarketValue + position.Quantity*currentPrice
	if position.Quantity > 0 {
		position.UnrealizedPL = (currentPrice - position.AvgPrice) * position.Quantity
	} else if position.Quantity < 0 {
		position.UnrealizedPL = (position.AvgPrice - currentPrice) * math.Abs(position.Quantity)
	} else {
		position.UnrealizedPL = 0
	}

	// Remove position if quantity is zero
	if position.Quantity == 0 {
		delete(p.positions, symbol)
	}

	// Add trade to history
	p.trades = append(p.trades, trade)

	return nil
}

// UpdateMarketValues updates the market values of all positions
func (p *Portfolio) UpdateMarketValues(barData map[string]strategy.BarData) {
	totalMarketValue := 0.0

	for symbol, position := range p.positions {
		if bar, exists := barData[symbol]; exists {
			position.MarketValue = position.Quantity * bar.Close

			if position.Quantity > 0 {
				position.UnrealizedPL = (bar.Close - position.AvgPrice) * position.Quantity
			} else if position.Quantity < 0 {
				position.UnrealizedPL = (position.AvgPrice - bar.Close) * math.Abs(position.Quantity)
			}
		}
		totalMarketValue += position.MarketValue
	}

	p.totalValue = p.cash + totalMarketValue

	// Update drawdown tracking
	if p.totalValue > p.peakValue {
		p.peakValue = p.totalValue
		p.currentDrawdown = 0
	} else {
		p.currentDrawdown = (p.peakValue - p.totalValue) / p.peakValue
		if p.currentDrawdown > p.maxDrawdown {
			p.maxDrawdown = p.currentDrawdown
		}
	}
}

// AddEquityPoint adds an equity point for performance tracking
func (p *Portfolio) AddEquityPoint(timestamp time.Time) {
	p.equity = append(p.equity, EquityPoint{
		Timestamp: timestamp,
		Value:     p.totalValue,
	})
}

// GetEquityCurve returns the equity curve
func (p *Portfolio) GetEquityCurve() []EquityPoint {
	return p.equity
}

// GetMaxDrawdown returns the maximum drawdown
func (p *Portfolio) GetMaxDrawdown() float64 {
	return p.maxDrawdown
}

// GetCurrentDrawdown returns the current drawdown
func (p *Portfolio) GetCurrentDrawdown() float64 {
	return p.currentDrawdown
}

// CanAfford checks if the portfolio can afford a trade
func (p *Portfolio) CanAfford(order strategy.Order, price float64) bool {
	if order.Side == strategy.OrderSideBuy {
		tradeValue := order.Quantity * price
		commission := p.commissionConfig.CalculateCommission(tradeValue)
		totalCost := tradeValue + commission
		return p.cash >= totalCost
	}

	// For sell orders, check if we have enough shares
	position := p.GetPosition(order.Symbol)
	if position == nil {
		return false // Cannot sell if no position
	}

	return position.Quantity >= order.Quantity
}

// ToStrategyPortfolio converts to strategy.Portfolio format
func (p *Portfolio) ToStrategyPortfolio() *strategy.Portfolio {
	totalPL := 0.0
	for _, position := range p.positions {
		totalPL += position.RealizedPL + position.UnrealizedPL
	}

	return &strategy.Portfolio{
		Cash:       p.cash,
		TotalValue: p.totalValue,
		Positions:  p.positions,
		TotalPL:    totalPL,
		Trades:     p.trades,
	}
}
