package strategy

import (
	"fmt"
	"time"
)

// BaseStrategy provides a default implementation of common strategy functionality
type BaseStrategy struct {
	name       string
	parameters map[string]interface{}
	symbols    []string
	timeframe  string
}

// NewBaseStrategy creates a new base strategy
func NewBaseStrategy(name string, parameters map[string]interface{}) *BaseStrategy {
	return &BaseStrategy{
		name:       name,
		parameters: parameters,
		symbols:    []string{},
		timeframe:  "1m",
	}
}

// GetName returns the strategy name
func (s *BaseStrategy) GetName() string {
	return s.name
}

// GetParameters returns the strategy parameters
func (s *BaseStrategy) GetParameters() map[string]interface{} {
	return s.parameters
}

// SetSymbols sets the symbols this strategy will trade
func (s *BaseStrategy) SetSymbols(symbols []string) {
	s.symbols = symbols
}

// GetSymbols returns the symbols this strategy trades
func (s *BaseStrategy) GetSymbols() []string {
	return s.symbols
}

// SetTimeframe sets the timeframe for this strategy
func (s *BaseStrategy) SetTimeframe(timeframe string) {
	s.timeframe = timeframe
}

// GetTimeframe returns the timeframe for this strategy
func (s *BaseStrategy) GetTimeframe() string {
	return s.timeframe
}

// GetParameter returns a parameter value with type assertion helpers
func (s *BaseStrategy) GetParameter(key string) interface{} {
	return s.parameters[key]
}

// GetParameterFloat64 returns a parameter as float64
func (s *BaseStrategy) GetParameterFloat64(key string) (float64, error) {
	val, ok := s.parameters[key]
	if !ok {
		return 0, fmt.Errorf("parameter %s not found", key)
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("parameter %s is not a number", key)
	}
}

// GetParameterInt returns a parameter as int
func (s *BaseStrategy) GetParameterInt(key string) (int, error) {
	val, ok := s.parameters[key]
	if !ok {
		return 0, fmt.Errorf("parameter %s not found", key)
	}

	switch v := val.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("parameter %s is not an integer", key)
	}
}

// GetParameterString returns a parameter as string
func (s *BaseStrategy) GetParameterString(key string) (string, error) {
	val, ok := s.parameters[key]
	if !ok {
		return "", fmt.Errorf("parameter %s not found", key)
	}

	if str, ok := val.(string); ok {
		return str, nil
	}

	return "", fmt.Errorf("parameter %s is not a string", key)
}

// CreateMarketOrder creates a market order
func (s *BaseStrategy) CreateMarketOrder(symbol string, side OrderSide, quantity float64) Order {
	return Order{
		ID:        generateOrderID(),
		Symbol:    symbol,
		Side:      side,
		Type:      OrderTypeMarket,
		Quantity:  quantity,
		Timestamp: time.Now(),
		Strategy:  s.name,
	}
}

// CreateLimitOrder creates a limit order
func (s *BaseStrategy) CreateLimitOrder(symbol string, side OrderSide, quantity float64, price float64) Order {
	return Order{
		ID:        generateOrderID(),
		Symbol:    symbol,
		Side:      side,
		Type:      OrderTypeLimit,
		Quantity:  quantity,
		Price:     price,
		Timestamp: time.Now(),
		Strategy:  s.name,
	}
}

// Default implementations for strategy interface (to be overridden)

// Initialize provides a default initialization
func (s *BaseStrategy) Initialize(ctx Context) error {
	ctx.Log("info", "Strategy initialized", map[string]interface{}{
		"strategy": s.name,
		"symbols":  s.symbols,
	})
	return nil
}

// OnBar provides a default implementation that does nothing
func (s *BaseStrategy) OnBar(ctx Context, bar BarData) ([]Order, error) {
	return []Order{}, nil
}

// OnTrade provides a default implementation that logs the trade
func (s *BaseStrategy) OnTrade(ctx Context, trade TradeEvent) error {
	ctx.Log("info", "Trade executed", map[string]interface{}{
		"strategy": s.name,
		"symbol":   trade.Symbol,
		"side":     trade.Side,
		"quantity": trade.Quantity,
		"price":    trade.Price,
	})
	return nil
}

// Cleanup provides a default cleanup
func (s *BaseStrategy) Cleanup(ctx Context) error {
	ctx.Log("info", "Strategy cleanup", map[string]interface{}{
		"strategy": s.name,
	})
	return nil
}

// Helper function to generate unique order IDs
func generateOrderID() string {
	return fmt.Sprintf("ORD_%d", time.Now().UnixNano())
}
