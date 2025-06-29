package backtester

import (
	"fmt"

	"github.com/ridopark/JonBuhTrader/pkg/logging"
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
	"github.com/rs/zerolog"
)

// StrategyContext implements the strategy.Context interface for backtesting
type StrategyContext struct {
	engine       *Engine
	logger       zerolog.Logger
	priceHistory map[string][]float64 // symbol -> slice of historical prices
}

// NewStrategyContext creates a new strategy context
func NewStrategyContext(engine *Engine) *StrategyContext {
	return &StrategyContext{
		engine:       engine,
		logger:       logging.GetLogger("strategy"),
		priceHistory: make(map[string][]float64),
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

// UpdatePriceHistory updates the price history for technical indicators
func (sc *StrategyContext) UpdatePriceHistory(dataPoint strategy.DataPoint) {
	for symbol, bar := range dataPoint.Bars {
		if sc.priceHistory[symbol] == nil {
			sc.priceHistory[symbol] = make([]float64, 0)
		}
		sc.priceHistory[symbol] = append(sc.priceHistory[symbol], bar.Close)

		// Keep only last 200 prices to avoid memory issues
		if len(sc.priceHistory[symbol]) > 200 {
			sc.priceHistory[symbol] = sc.priceHistory[symbol][1:]
		}
	}
}

// SMA calculates Simple Moving Average
func (sc *StrategyContext) SMA(symbol string, period int) (float64, error) {
	prices, exists := sc.priceHistory[symbol]
	if !exists {
		return 0, fmt.Errorf("no price history available for symbol %s", symbol)
	}

	if len(prices) < period {
		return 0, fmt.Errorf("insufficient data: need %d periods, have %d", period, len(prices))
	}

	// Calculate SMA using the last 'period' prices
	sum := 0.0
	start := len(prices) - period
	for i := start; i < len(prices); i++ {
		sum += prices[i]
	}

	return sum / float64(period), nil
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

func (sc *StrategyContext) MACD(symbol string, fastPeriod, slowPeriod, signalPeriod int) (float64, float64, float64, error) {
	// This is a simplified implementation
	// In a full implementation, we'd calculate MACD from historical data
	return 0, 0, 0, fmt.Errorf("MACD not fully implemented in backtester context")
}

func (sc *StrategyContext) ADX(symbol string, period int) (float64, error) {
	// This is a simplified implementation
	// In a full implementation, we'd calculate ADX from historical data
	return 0, fmt.Errorf("ADX not fully implemented in backtester context")
}

func (sc *StrategyContext) SuperTrend(symbol string, period int, multiplier float64) (float64, error) {
	// This is a simplified implementation
	// In a full implementation, we'd calculate SuperTrend from historical data
	return 0, fmt.Errorf("SuperTrend not fully implemented in backtester context")
}

func (sc *StrategyContext) ParbolicSAR(symbol string, step, max float64) (float64, error) {
	// This is a simplified implementation
	// In a full implementation, we'd calculate Parbolic SAR from historical data
	return 0, fmt.Errorf("ParbolicSAR not fully implemented in backtester context")
}

// Log logs a message with the given level and fields
func (sc *StrategyContext) Log(level string, message string, fields map[string]interface{}) {
	var event *zerolog.Event

	switch level {
	case "trace":
		event = sc.logger.Trace()
	case "debug":
		event = sc.logger.Debug()
	case "info":
		event = sc.logger.Info()
	case "warn":
		event = sc.logger.Warn()
	case "error":
		event = sc.logger.Error()
	case "fatal":
		event = sc.logger.Fatal()
	case "panic":
		event = sc.logger.Panic()
	default:
		event = sc.logger.Info()
	}

	// Add all fields to the log event
	for key, value := range fields {
		event = event.Interface(key, value)
	}

	event.Msg(message)
}
