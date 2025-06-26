package backtester

import (
	"fmt"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// StrategyContext implements the strategy.Context interface for backtesting
type StrategyContext struct {
	engine *Engine
}

// NewStrategyContext creates a new strategy context
func NewStrategyContext(engine *Engine) *StrategyContext {
	return &StrategyContext{
		engine: engine,
	}
}

// GetPortfolio returns the current portfolio state
func (sc *StrategyContext) GetPortfolio() *strategy.Portfolio {
	return sc.engine.portfolio.ToStrategyPortfolio()
}

// GetPosition returns the position for a symbol
func (sc *StrategyContext) GetPosition(symbol string) *strategy.Position {
	return sc.engine.portfolio.GetPosition(symbol)
}

// GetCash returns the current cash balance
func (sc *StrategyContext) GetCash() float64 {
	return sc.engine.portfolio.GetCash()
}

// GetBars returns historical bars for a symbol (simplified implementation)
func (sc *StrategyContext) GetBars(symbol string, timeframe string, limit int) ([]strategy.BarData, error) {
	// This is a simplified implementation
	// In a full implementation, we'd maintain historical data for quick access
	return []strategy.BarData{}, fmt.Errorf("GetBars not fully implemented in backtester context")
}

// GetLastBar returns the last bar for a symbol (simplified implementation)
func (sc *StrategyContext) GetLastBar(symbol string, timeframe string) (*strategy.BarData, error) {
	// This is a simplified implementation
	// In a full implementation, we'd track the last bar for each symbol
	return nil, fmt.Errorf("GetLastBar not fully implemented in backtester context")
}

// SMA calculates Simple Moving Average (simplified implementation)
func (sc *StrategyContext) SMA(symbol string, period int) (float64, error) {
	// This is a simplified implementation
	// In a full implementation, we'd calculate SMA from historical data
	return 0, fmt.Errorf("SMA not fully implemented in backtester context")
}

// EMA calculates Exponential Moving Average (simplified implementation)
func (sc *StrategyContext) EMA(symbol string, period int) (float64, error) {
	// This is a simplified implementation
	// In a full implementation, we'd calculate EMA from historical data
	return 0, fmt.Errorf("EMA not fully implemented in backtester context")
}

// RSI calculates Relative Strength Index (simplified implementation)
func (sc *StrategyContext) RSI(symbol string, period int) (float64, error) {
	// This is a simplified implementation
	// In a full implementation, we'd calculate RSI from historical data
	return 0, fmt.Errorf("RSI not fully implemented in backtester context")
}

// Log logs a message with the given level and fields
func (sc *StrategyContext) Log(level string, message string, fields map[string]interface{}) {
	// Simple logging implementation
	fmt.Printf("[%s] %s %+v\n", level, message, fields)
}
