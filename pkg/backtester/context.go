package backtester

import (
	"fmt"
	"math"

	"github.com/ridopark/JonBuhTrader/pkg/logging"
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
	"github.com/rs/zerolog"
)

// ADXData stores data for ADX calculation
type ADXData struct {
	TrueRanges []float64 // True Range values
	DMPlus     []float64 // Directional Movement Plus
	DMMinus    []float64 // Directional Movement Minus
	PrevHigh   float64   // Previous high
	PrevLow    float64   // Previous low
	PrevClose  float64   // Previous close
}

// IndicatorData stores data for technical indicators
type IndicatorData struct {
	PriceHistory []float64       // Historical close prices
	HighHistory  []float64       // Historical high prices
	LowHistory   []float64       // Historical low prices
	EMAValues    map[int]float64 // EMA values by period
	RSIData      *RSIData        // RSI calculation data
	MACDData     *MACDData       // MACD calculation data
	ADXData      *ADXData        // ADX calculation data
}

// RSIData stores data for RSI calculation
type RSIData struct {
	Gains  []float64 // Price gains
	Losses []float64 // Price losses
}

// MACDData stores data for MACD calculation
type MACDData struct {
	FastEMA   float64 // Fast EMA value
	SlowEMA   float64 // Slow EMA value
	SignalEMA float64 // Signal line EMA value
}

// StrategyContext implements the strategy.Context interface for backtesting
type StrategyContext struct {
	engine     *Engine
	logger     zerolog.Logger
	indicators map[string]*IndicatorData // symbol -> indicator data
}

// NewStrategyContext creates a new strategy context
func NewStrategyContext(engine *Engine) *StrategyContext {
	return &StrategyContext{
		engine:     engine,
		logger:     logging.GetLogger("strategy"),
		indicators: make(map[string]*IndicatorData),
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
		if sc.indicators[symbol] == nil {
			sc.indicators[symbol] = &IndicatorData{
				PriceHistory: make([]float64, 0),
				HighHistory:  make([]float64, 0),
				LowHistory:   make([]float64, 0),
				EMAValues:    make(map[int]float64),
				RSIData:      &RSIData{Gains: make([]float64, 0), Losses: make([]float64, 0)},
				MACDData:     &MACDData{},
				ADXData:      &ADXData{TrueRanges: make([]float64, 0), DMPlus: make([]float64, 0), DMMinus: make([]float64, 0)},
			}
		}

		data := sc.indicators[symbol]
		data.PriceHistory = append(data.PriceHistory, bar.Close)
		data.HighHistory = append(data.HighHistory, bar.High)
		data.LowHistory = append(data.LowHistory, bar.Low)

		// Keep only last 200 prices to avoid memory issues
		if len(data.PriceHistory) > 200 {
			data.PriceHistory = data.PriceHistory[1:]
			data.HighHistory = data.HighHistory[1:]
			data.LowHistory = data.LowHistory[1:]
			// Also trim RSI data
			if len(data.RSIData.Gains) > 200 {
				data.RSIData.Gains = data.RSIData.Gains[1:]
			}
			if len(data.RSIData.Losses) > 200 {
				data.RSIData.Losses = data.RSIData.Losses[1:]
			}
			// Trim ADX data
			if len(data.ADXData.TrueRanges) > 200 {
				data.ADXData.TrueRanges = data.ADXData.TrueRanges[1:]
			}
			if len(data.ADXData.DMPlus) > 200 {
				data.ADXData.DMPlus = data.ADXData.DMPlus[1:]
			}
			if len(data.ADXData.DMMinus) > 200 {
				data.ADXData.DMMinus = data.ADXData.DMMinus[1:]
			}
		}

		// Update RSI data if we have previous price
		if len(data.PriceHistory) > 1 {
			prevPrice := data.PriceHistory[len(data.PriceHistory)-2]
			currentPrice := data.PriceHistory[len(data.PriceHistory)-1]
			change := currentPrice - prevPrice

			if change > 0 {
				data.RSIData.Gains = append(data.RSIData.Gains, change)
				data.RSIData.Losses = append(data.RSIData.Losses, 0)
			} else {
				data.RSIData.Gains = append(data.RSIData.Gains, 0)
				data.RSIData.Losses = append(data.RSIData.Losses, -change)
			}
		}

		// Update ADX data
		sc.updateADXData(data, bar.High, bar.Low, bar.Close)
	}
}

// SMA calculates Simple Moving Average
func (sc *StrategyContext) SMA(symbol string, period int) (float64, error) {
	data, exists := sc.indicators[symbol]
	if !exists || data.PriceHistory == nil {
		return 0, fmt.Errorf("no price history available for symbol %s", symbol)
	}

	prices := data.PriceHistory
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

// EMA calculates Exponential Moving Average
func (sc *StrategyContext) EMA(symbol string, period int) (float64, error) {
	data, exists := sc.indicators[symbol]
	if !exists || data.PriceHistory == nil {
		return 0, fmt.Errorf("no price history available for symbol %s", symbol)
	}

	prices := data.PriceHistory
	if len(prices) < period {
		return 0, fmt.Errorf("insufficient data: need %d periods, have %d", period, len(prices))
	}

	// Check if we already have a calculated EMA for this period
	if emaValue, exists := data.EMAValues[period]; exists {
		// Update existing EMA with new price
		multiplier := 2.0 / (float64(period) + 1.0)
		currentPrice := prices[len(prices)-1]
		newEMA := (currentPrice * multiplier) + (emaValue * (1.0 - multiplier))
		data.EMAValues[period] = newEMA
		return newEMA, nil
	}

	// Calculate initial EMA using SMA as seed
	if len(prices) < period {
		return 0, fmt.Errorf("insufficient data for initial EMA calculation")
	}

	// Calculate SMA for the first 'period' values as initial EMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	ema := sum / float64(period)

	// Calculate EMA for remaining values
	multiplier := 2.0 / (float64(period) + 1.0)
	for i := period; i < len(prices); i++ {
		ema = (prices[i] * multiplier) + (ema * (1.0 - multiplier))
	}

	// Store the calculated EMA
	data.EMAValues[period] = ema
	return ema, nil
}

// RSI calculates Relative Strength Index
func (sc *StrategyContext) RSI(symbol string, period int) (float64, error) {
	data, exists := sc.indicators[symbol]
	if !exists || data.RSIData == nil {
		return 0, fmt.Errorf("no RSI data available for symbol %s", symbol)
	}

	gains := data.RSIData.Gains
	losses := data.RSIData.Losses

	if len(gains) < period || len(losses) < period {
		return 0, fmt.Errorf("insufficient data for RSI: need %d periods, have %d", period, len(gains))
	}

	// Calculate average gain and average loss for the period
	avgGain := 0.0
	avgLoss := 0.0

	// Use the last 'period' values
	start := len(gains) - period
	for i := start; i < len(gains); i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}

	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Avoid division by zero
	if avgLoss == 0 {
		return 100, nil // RSI = 100 when there are no losses
	}

	// Calculate RSI
	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi, nil
}

// MACD calculates Moving Average Convergence Divergence
func (sc *StrategyContext) MACD(symbol string, fastPeriod, slowPeriod, signalPeriod int) (float64, float64, float64, error) {
	// Get fast and slow EMAs
	fastEMA, err := sc.EMA(symbol, fastPeriod)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error calculating fast EMA: %v", err)
	}

	slowEMA, err := sc.EMA(symbol, slowPeriod)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error calculating slow EMA: %v", err)
	}

	// MACD line = Fast EMA - Slow EMA
	macdLine := fastEMA - slowEMA

	data, exists := sc.indicators[symbol]
	if !exists {
		return 0, 0, 0, fmt.Errorf("no indicator data available for symbol %s", symbol)
	}

	// Update MACD data
	data.MACDData.FastEMA = fastEMA
	data.MACDData.SlowEMA = slowEMA

	// Calculate Signal line (EMA of MACD line)
	// For simplicity, we'll use a basic signal calculation
	// In a full implementation, we'd maintain a history of MACD values
	// and calculate the EMA of those values
	signalLine := data.MACDData.SignalEMA
	if signalLine == 0 {
		// Initialize signal line with first MACD value
		signalLine = macdLine
	} else {
		// Update signal line using EMA formula
		multiplier := 2.0 / (float64(signalPeriod) + 1.0)
		signalLine = (macdLine * multiplier) + (signalLine * (1.0 - multiplier))
	}
	data.MACDData.SignalEMA = signalLine

	// Histogram = MACD Line - Signal Line
	histogram := macdLine - signalLine

	return macdLine, signalLine, histogram, nil
}

// ADX calculates Average Directional Index
func (sc *StrategyContext) ADX(symbol string, period int) (float64, error) {
	data, exists := sc.indicators[symbol]
	if !exists || data.ADXData == nil {
		return 0, fmt.Errorf("no ADX data available for symbol %s", symbol)
	}

	trueRanges := data.ADXData.TrueRanges
	dmPlus := data.ADXData.DMPlus
	dmMinus := data.ADXData.DMMinus

	if len(trueRanges) < period || len(dmPlus) < period || len(dmMinus) < period {
		return 0, fmt.Errorf("insufficient data for ADX: need %d periods, have %d", period, len(trueRanges))
	}

	// Calculate ATR (Average True Range)
	atr := 0.0
	start := len(trueRanges) - period
	for i := start; i < len(trueRanges); i++ {
		atr += trueRanges[i]
	}
	atr /= float64(period)

	if atr == 0 {
		return 0, nil // Avoid division by zero
	}

	// Calculate DI+ and DI-
	diPlus := 0.0
	diMinus := 0.0
	for i := start; i < len(dmPlus); i++ {
		diPlus += dmPlus[i]
		diMinus += dmMinus[i]
	}
	diPlus = (diPlus / float64(period)) / atr * 100
	diMinus = (diMinus / float64(period)) / atr * 100

	// Calculate DX
	if diPlus+diMinus == 0 {
		return 0, nil
	}
	dx := math.Abs(diPlus-diMinus) / (diPlus + diMinus) * 100

	// For simplicity, return DX as ADX approximation
	// In a full implementation, ADX would be the smoothed average of DX values
	return dx, nil
}

// SuperTrend calculates SuperTrend indicator
func (sc *StrategyContext) SuperTrend(symbol string, period int, multiplier float64) (float64, error) {
	data, exists := sc.indicators[symbol]
	if !exists || data.HighHistory == nil || data.LowHistory == nil {
		return 0, fmt.Errorf("no price history available for symbol %s", symbol)
	}

	if len(data.HighHistory) < period || len(data.LowHistory) < period || len(data.PriceHistory) < period {
		return 0, fmt.Errorf("insufficient data for SuperTrend: need %d periods, have %d", period, len(data.PriceHistory))
	}

	// Calculate ATR for the period
	atr := 0.0
	if len(data.ADXData.TrueRanges) >= period {
		start := len(data.ADXData.TrueRanges) - period
		for i := start; i < len(data.ADXData.TrueRanges); i++ {
			atr += data.ADXData.TrueRanges[i]
		}
		atr /= float64(period)
	} else {
		// Fallback: simple range calculation
		start := len(data.HighHistory) - period
		for i := start; i < len(data.HighHistory); i++ {
			atr += data.HighHistory[i] - data.LowHistory[i]
		}
		atr /= float64(period)
	}

	// Calculate HL2 (median price)
	currentHigh := data.HighHistory[len(data.HighHistory)-1]
	currentLow := data.LowHistory[len(data.LowHistory)-1]
	hl2 := (currentHigh + currentLow) / 2

	// Calculate SuperTrend
	upperBand := hl2 + (multiplier * atr)
	lowerBand := hl2 - (multiplier * atr)

	currentClose := data.PriceHistory[len(data.PriceHistory)-1]

	// Simple SuperTrend logic: return lower band if price is above, upper band if below
	if currentClose > hl2 {
		return lowerBand, nil
	} else {
		return upperBand, nil
	}
}

// ParabolicSAR calculates Parabolic SAR
func (sc *StrategyContext) ParbolicSAR(symbol string, step, max float64) (float64, error) {
	data, exists := sc.indicators[symbol]
	if !exists || data.HighHistory == nil || data.LowHistory == nil {
		return 0, fmt.Errorf("no price history available for symbol %s", symbol)
	}

	highs := data.HighHistory
	lows := data.LowHistory

	if len(highs) < 2 || len(lows) < 2 {
		return 0, fmt.Errorf("insufficient data for Parabolic SAR: need at least 2 periods")
	}

	// Simplified Parabolic SAR calculation
	// In a full implementation, this would maintain state across calls
	currentHigh := highs[len(highs)-1]
	prevHigh := highs[len(highs)-2]
	prevLow := lows[len(lows)-2]

	// Simple approximation: use previous low as SAR for uptrend
	if currentHigh > prevHigh {
		return prevLow, nil
	} else {
		return prevHigh, nil
	}
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

// updateADXData updates the ADX calculation data
func (sc *StrategyContext) updateADXData(data *IndicatorData, high, low, close float64) {
	if data.ADXData.PrevClose == 0 {
		// First data point, just store values
		data.ADXData.PrevHigh = high
		data.ADXData.PrevLow = low
		data.ADXData.PrevClose = close
		return
	}

	// Calculate True Range
	tr1 := high - low
	tr2 := math.Abs(high - data.ADXData.PrevClose)
	tr3 := math.Abs(low - data.ADXData.PrevClose)
	tr := math.Max(tr1, math.Max(tr2, tr3))
	data.ADXData.TrueRanges = append(data.ADXData.TrueRanges, tr)

	// Calculate Directional Movements
	dmPlus := 0.0
	dmMinus := 0.0

	if high-data.ADXData.PrevHigh > data.ADXData.PrevLow-low {
		if high-data.ADXData.PrevHigh > 0 {
			dmPlus = high - data.ADXData.PrevHigh
		}
	} else {
		if data.ADXData.PrevLow-low > 0 {
			dmMinus = data.ADXData.PrevLow - low
		}
	}

	data.ADXData.DMPlus = append(data.ADXData.DMPlus, dmPlus)
	data.ADXData.DMMinus = append(data.ADXData.DMMinus, dmMinus)

	// Update previous values
	data.ADXData.PrevHigh = high
	data.ADXData.PrevLow = low
	data.ADXData.PrevClose = close
}
