package backtester

import (
	"fmt"
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// Broker simulates order execution for backtesting
type Broker struct {
	commission float64
	slippage   float64 // As a percentage
}

// NewBroker creates a new simulated broker
func NewBroker(commission, slippage float64) *Broker {
	return &Broker{
		commission: commission,
		slippage:   slippage,
	}
}

// ExecuteOrder executes an order and returns a trade event
func (b *Broker) ExecuteOrder(order strategy.Order, currentBar strategy.BarData) (*strategy.TradeEvent, error) {
	var fillPrice float64

	switch order.Type {
	case strategy.OrderTypeMarket:
		// For market orders, use the current close price with slippage
		if order.Side == strategy.OrderSideBuy {
			fillPrice = currentBar.Close * (1 + b.slippage/100)
		} else {
			fillPrice = currentBar.Close * (1 - b.slippage/100)
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
				fillPrice = order.StopPrice * (1 + b.slippage/100)
			} else {
				return nil, fmt.Errorf("stop buy order not triggered: price %f < high %f", order.StopPrice, currentBar.High)
			}
		} else {
			if currentBar.Low <= order.StopPrice {
				fillPrice = order.StopPrice * (1 - b.slippage/100)
			} else {
				return nil, fmt.Errorf("stop sell order not triggered: price %f > low %f", order.StopPrice, currentBar.Low)
			}
		}

	default:
		return nil, fmt.Errorf("unsupported order type: %s", order.Type)
	}

	// Calculate commission
	commission := order.Quantity * fillPrice * b.commission

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
		Strategy:   order.Strategy,
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
	switch order.Type {
	case strategy.OrderTypeMarket:
		if order.Side == strategy.OrderSideBuy {
			return currentBar.Close * (1 + b.slippage/100), nil
		} else {
			return currentBar.Close * (1 - b.slippage/100), nil
		}

	case strategy.OrderTypeLimit:
		if b.CanExecuteOrder(order, currentBar) {
			return order.Price, nil
		}
		return 0, fmt.Errorf("limit order cannot be executed")

	case strategy.OrderTypeStop:
		if b.CanExecuteOrder(order, currentBar) {
			if order.Side == strategy.OrderSideBuy {
				return order.StopPrice * (1 + b.slippage/100), nil
			} else {
				return order.StopPrice * (1 - b.slippage/100), nil
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
