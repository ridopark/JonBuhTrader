package backtester

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// CommissionType defines the type of commission calculation
type CommissionType string

const (
	CommissionTypePercentage CommissionType = "percentage"
	CommissionTypeFixed      CommissionType = "fixed"
)

// CommissionConfig holds commission configuration
type CommissionConfig struct {
	Type CommissionType
	Rate float64 // For percentage: decimal (0.001 = 0.1%), For fixed: dollar amount per trade
}

// NewCommissionConfig creates a new commission configuration
func NewCommissionConfig(commissionType CommissionType, rate float64) *CommissionConfig {
	return &CommissionConfig{
		Type: commissionType,
		Rate: rate,
	}
}

// CalculateCommission calculates commission based on trade value and configuration
func (cc *CommissionConfig) CalculateCommission(tradeValue float64) float64 {
	switch cc.Type {
	case CommissionTypePercentage:
		return tradeValue * cc.Rate
	case CommissionTypeFixed:
		return cc.Rate
	default:
		return tradeValue * cc.Rate // Default to percentage
	}
}

// Broker simulates order execution for backtesting
type Broker struct {
	commissionConfig *CommissionConfig
	slippage         float64 // Base slippage as a percentage
	maxSlippage      float64 // Maximum randomized slippage as a percentage
}

// NewBroker creates a new simulated broker
func NewBroker(commissionConfig *CommissionConfig, slippage float64, maxSlippage float64) *Broker {
	return &Broker{
		commissionConfig: commissionConfig,
		slippage:         slippage,
		maxSlippage:      maxSlippage,
	}
}

// calculateRandomizedSlippage calculates randomized slippage using noise model
func (b *Broker) calculateRandomizedSlippage() float64 {
	// Base slippage + randomized component
	randomSlippage := rand.Float64() * b.maxSlippage
	return b.slippage + randomSlippage
}

// ExecuteOrder executes an order and returns a trade event
func (b *Broker) ExecuteOrder(order strategy.Order, currentBar strategy.BarData) (*strategy.TradeEvent, error) {
	var fillPrice float64

	// Calculate randomized slippage for this trade
	totalSlippage := b.calculateRandomizedSlippage()

	switch order.Type {
	case strategy.OrderTypeMarket:
		// For market orders, use the current close price with randomized slippage
		if order.Side == strategy.OrderSideBuy {
			fillPrice = currentBar.Close * (1 + totalSlippage/100)
		} else {
			fillPrice = currentBar.Close * (1 - totalSlippage/100)
		}

	case strategy.OrderTypeLimit:
		// For limit orders, check if the order can be filled
		if order.Side == strategy.OrderSideBuy {
			if currentBar.Low <= order.Price {
				fillPrice = order.Price
			} else {
				return nil, fmt.Errorf("limit buy order not filled: price %f > low %f", order.Price, currentBar.Low)
			}
		} else {
			if currentBar.High >= order.Price {
				fillPrice = order.Price
			} else {
				return nil, fmt.Errorf("limit sell order not filled: price %f < high %f", order.Price, currentBar.High)
			}
		}

	case strategy.OrderTypeStop:
		// For stop orders, check if the stop is triggered
		if order.Side == strategy.OrderSideBuy {
			if currentBar.High >= order.StopPrice {
				fillPrice = order.StopPrice * (1 + totalSlippage/100)
			} else {
				return nil, fmt.Errorf("stop buy order not triggered: price %f < high %f", order.StopPrice, currentBar.High)
			}
		} else {
			if currentBar.Low <= order.StopPrice {
				fillPrice = order.StopPrice * (1 - totalSlippage/100)
			} else {
				return nil, fmt.Errorf("stop sell order not triggered: price %f > low %f", order.StopPrice, currentBar.Low)
			}
		}

	default:
		return nil, fmt.Errorf("unsupported order type: %s", order.Type)
	}

	// Calculate fees and costs
	tradeValue := order.Quantity * fillPrice
	commission := b.commissionConfig.CalculateCommission(tradeValue)
	slippageCost := 0.0

	// Calculate slippage cost (difference from expected price)
	expectedPrice := currentBar.Close
	if order.Type == strategy.OrderTypeLimit {
		expectedPrice = order.Price
	}
	slippageCost = math.Abs(fillPrice-expectedPrice) * order.Quantity

	// Calculate SEC fee (only on sells, $0.0000278 per dollar of sale proceeds)
	secFee := 0.0
	if order.Side == strategy.OrderSideSell {
		secFee = tradeValue * 0.0000278
	}

	// Calculate FINRA TAF (Trading Activity Fee: $0.000145 per share, max $7.27)
	finraTaf := math.Min(order.Quantity*0.000145, 7.27)

	// Create trade event
	trade := &strategy.TradeEvent{
		ID:         generateTradeID(),
		OrderID:    order.ID,
		Symbol:     order.Symbol,
		Side:       order.Side,
		Quantity:   order.Quantity,
		Price:      fillPrice,
		Timestamp:  currentBar.Timestamp,
		Commission: commission,
		SecFee:     secFee,
		FinraTaf:   finraTaf,
		Slippage:   slippageCost,
		Strategy:   order.Strategy,
		Reason:     order.Reason,
	}

	return trade, nil
}

// CanExecuteOrder checks if an order can be executed at the current bar
func (b *Broker) CanExecuteOrder(order strategy.Order, currentBar strategy.BarData) bool {
	switch order.Type {
	case strategy.OrderTypeMarket:
		return true

	case strategy.OrderTypeLimit:
		if order.Side == strategy.OrderSideBuy {
			return currentBar.Low <= order.Price
		} else {
			return currentBar.High >= order.Price
		}

	case strategy.OrderTypeStop:
		if order.Side == strategy.OrderSideBuy {
			return currentBar.High >= order.StopPrice
		} else {
			return currentBar.Low <= order.StopPrice
		}

	default:
		return false
	}
}

// GetExecutionPrice returns the price at which an order would be executed
func (b *Broker) GetExecutionPrice(order strategy.Order, currentBar strategy.BarData) (float64, error) {
	// Calculate randomized slippage for this execution price calculation
	totalSlippage := b.calculateRandomizedSlippage()

	switch order.Type {
	case strategy.OrderTypeMarket:
		if order.Side == strategy.OrderSideBuy {
			return currentBar.Close * (1 + totalSlippage/100), nil
		} else {
			return currentBar.Close * (1 - totalSlippage/100), nil
		}

	case strategy.OrderTypeLimit:
		if b.CanExecuteOrder(order, currentBar) {
			return order.Price, nil
		}
		return 0, fmt.Errorf("limit order cannot be executed")

	case strategy.OrderTypeStop:
		if b.CanExecuteOrder(order, currentBar) {
			if order.Side == strategy.OrderSideBuy {
				return order.StopPrice * (1 + totalSlippage/100), nil
			} else {
				return order.StopPrice * (1 - totalSlippage/100), nil
			}
		}
		return 0, fmt.Errorf("stop order not triggered")

	default:
		return 0, fmt.Errorf("unsupported order type: %s", order.Type)
	}
}

// Helper function to generate unique trade IDs
func generateTradeID() string {
	return fmt.Sprintf("TRD_%d", time.Now().UnixNano())
}
