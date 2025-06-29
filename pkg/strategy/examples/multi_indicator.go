package examples

import (
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// MultiIndicatorSignal represents a potential trading signal with priority
type MultiIndicatorSignal struct {
	Symbol     string
	Bar        strategy.BarData
	SignalType string // "buy" or "sell"
	Confidence int    // Number of confirming indicators
	Price      float64
}

// MultiIndicatorStrategy implements a strategy that uses multiple technical indicators
type MultiIndicatorStrategy struct {
	*strategy.BaseStrategy
	rsiPeriod     int
	smaPeriod     int
	emaPeriod     int
	macdFast      int
	macdSlow      int
	macdSignal    int
	rsiOversold   float64
	rsiOverbought float64
	positionSize  float64

	// Enhanced features
	confidenceThreshold float64

	// Internal state
	priceHistory map[string][]float64

	// Capital allocation
	allocator *strategy.CapitalAllocator
}

// NewMultiIndicatorStrategy creates a new multi-indicator strategy
func NewMultiIndicatorStrategy() *MultiIndicatorStrategy {
	base := strategy.NewBaseStrategy("MultiIndicator", map[string]interface{}{
		"rsiPeriod":     14,
		"smaPeriod":     20,
		"emaPeriod":     12,
		"macdFast":      12,
		"macdSlow":      26,
		"macdSignal":    9,
		"rsiOversold":   30,
		"rsiOverbought": 70,
		"positionSize":  0.95,
	})

	// Configure capital allocation
	allocConfig := strategy.DefaultAllocationConfig()
	allocConfig.Method = strategy.AllocateByConfidence
	allocConfig.PositionSize = 0.95
	allocConfig.MaxPositions = 3

	return &MultiIndicatorStrategy{
		BaseStrategy:        base,
		rsiPeriod:           14,
		smaPeriod:           20,
		emaPeriod:           12,
		macdFast:            12,
		macdSlow:            26,
		macdSignal:          9,
		rsiOversold:         30,
		rsiOverbought:       70,
		positionSize:        0.95,
		confidenceThreshold: 0.6,
		priceHistory:        make(map[string][]float64),
		allocator:           strategy.NewCapitalAllocator(allocConfig),
	}
}

// SetSymbols sets the symbols for this strategy
func (s *MultiIndicatorStrategy) SetSymbols(symbols []string) {
	s.BaseStrategy.SetSymbols(symbols)
}

// Initialize sets up the strategy
func (s *MultiIndicatorStrategy) Initialize(ctx strategy.Context) error {
	ctx.Log("info", "Multi-Indicator Strategy initialized", map[string]interface{}{
		"strategy":      s.GetName(),
		"rsiPeriod":     s.rsiPeriod,
		"smaPeriod":     s.smaPeriod,
		"emaPeriod":     s.emaPeriod,
		"macdFast":      s.macdFast,
		"macdSlow":      s.macdSlow,
		"macdSignal":    s.macdSignal,
		"rsiOversold":   s.rsiOversold,
		"rsiOverbought": s.rsiOverbought,
		"positionSize":  s.positionSize,
	})
	return nil
}

// OnDataPoint processes each data point and generates trading signals
func (s *MultiIndicatorStrategy) OnDataPoint(ctx strategy.Context, dataPoint strategy.DataPoint) ([]strategy.Order, error) {
	var potentialSignals []strategy.TradingSignal
	var orders []strategy.Order

	// Phase 1: Collect all potential buy signals
	for _, symbol := range s.GetSymbols() {
		bar, exists := dataPoint.Bars[symbol]
		if !exists {
			continue
		}

		// Get all indicators
		rsi, rsiErr := ctx.RSI(symbol, s.rsiPeriod)
		sma, smaErr := ctx.SMA(symbol, s.smaPeriod)
		ema, emaErr := ctx.EMA(symbol, s.emaPeriod)
		macd, signal, histogram, macdErr := ctx.MACD(symbol, s.macdFast, s.macdSlow, s.macdSignal)

		// Skip if we don't have enough data for key indicators
		if rsiErr != nil || smaErr != nil {
			continue
		}

		position := ctx.GetPosition(symbol)

		// Handle nil position (no position exists)
		positionQuantity := 0.0
		if position != nil {
			positionQuantity = position.Quantity
		}

		// Log all indicator values for analysis
		logFields := map[string]interface{}{
			"symbol":   symbol,
			"price":    bar.Close,
			"position": positionQuantity,
			"rsi":      rsi,
			"sma":      sma,
		}

		if emaErr == nil {
			logFields["ema"] = ema
		}
		if macdErr == nil {
			logFields["macd"] = macd
			logFields["macd_signal"] = signal
			logFields["macd_histogram"] = histogram
		}

		ctx.Log("debug", "Multi-indicator analysis", logFields)

		// Buy signal logic: Multiple confirmations required
		if positionQuantity == 0 {
			buySignals := 0

			// RSI oversold condition
			if rsi <= s.rsiOversold {
				buySignals++
				ctx.Log("debug", "Buy signal: RSI oversold", map[string]interface{}{
					"symbol": symbol,
					"rsi":    rsi,
				})
			}

			// Price above SMA (trend confirmation)
			if bar.Close > sma {
				buySignals++
				ctx.Log("debug", "Buy signal: Price above SMA", map[string]interface{}{
					"symbol": symbol,
					"price":  bar.Close,
					"sma":    sma,
				})
			}

			// EMA above SMA (bullish trend)
			if emaErr == nil && ema > sma {
				buySignals++
				ctx.Log("debug", "Buy signal: EMA above SMA", map[string]interface{}{
					"symbol": symbol,
					"ema":    ema,
					"sma":    sma,
				})
			}

			// MACD bullish
			if macdErr == nil && macd > signal && histogram > 0 {
				buySignals++
				ctx.Log("debug", "Buy signal: MACD bullish", map[string]interface{}{
					"symbol":    symbol,
					"macd":      macd,
					"signal":    signal,
					"histogram": histogram,
				})
			}

			// Require at least 2 buy signals
			if buySignals >= 2 {
				confidence := float64(buySignals) / 4.0 // Normalize to 0-1 range
				potentialSignals = append(potentialSignals, MultiIndicatorSignalImpl{
					Symbol:     symbol,
					Bar:        bar,
					SignalType: "buy",
					Price:      bar.Close,
					RSI:        rsi,
					SMA:        sma,
					EMA:        ema,
					MACD:       macd,
					MACDSignal: signal,
					MACDHisto:  histogram,
					Confidence: confidence,
					Priority:   confidence, // Use confidence as priority
				})

				ctx.Log("debug", "Multi-indicator potential BUY signal", map[string]interface{}{
					"symbol":     symbol,
					"price":      bar.Close,
					"buySignals": buySignals,
					"rsi":        rsi,
				})
			}
		}

		// Sell signal logic
		if positionQuantity > 0 {
			sellSignals := 0

			// RSI overbought condition
			if rsi >= s.rsiOverbought {
				sellSignals++
				ctx.Log("debug", "Sell signal: RSI overbought", map[string]interface{}{
					"symbol": symbol,
					"rsi":    rsi,
				})
			}

			// Price below SMA (trend reversal)
			if bar.Close < sma {
				sellSignals++
				ctx.Log("debug", "Sell signal: Price below SMA", map[string]interface{}{
					"symbol": symbol,
					"price":  bar.Close,
					"sma":    sma,
				})
			}

			// EMA below SMA (bearish trend)
			if emaErr == nil && ema < sma {
				sellSignals++
				ctx.Log("debug", "Sell signal: EMA below SMA", map[string]interface{}{
					"symbol": symbol,
					"ema":    ema,
					"sma":    sma,
				})
			}

			// MACD bearish
			if macdErr == nil && macd < signal && histogram < 0 {
				sellSignals++
				ctx.Log("debug", "Sell signal: MACD bearish", map[string]interface{}{
					"symbol":    symbol,
					"macd":      macd,
					"signal":    signal,
					"histogram": histogram,
				})
			}

			// Require at least 1 sell signal (more conservative exit)
			if sellSignals >= 1 {
				order := strategy.Order{
					Symbol:   symbol,
					Side:     strategy.OrderSideSell,
					Type:     strategy.OrderTypeMarket,
					Quantity: positionQuantity,
					Strategy: s.GetName(),
				}
				orders = append(orders, order)

				ctx.Log("info", "Multi-indicator SELL signal", map[string]interface{}{
					"symbol":      symbol,
					"price":       bar.Close,
					"quantity":    positionQuantity,
					"sellSignals": sellSignals,
					"rsi":         rsi,
				})
			}
		}
	}

	// Phase 2: Allocate capital to buy signals using the common allocation system
	if len(potentialSignals) > 0 {
		buyOrders := s.allocator.AllocateCapital(ctx, potentialSignals, s.GetName())
		orders = append(orders, buyOrders...)
	}

	return orders, nil
}

// OnFinish is called when the strategy finishes
func (s *MultiIndicatorStrategy) OnFinish(ctx strategy.Context) error {
	ctx.Log("info", "Multi-Indicator Strategy finished", map[string]interface{}{
		"finalCash": ctx.GetCash(),
	})
	return nil
}

// calculatePositionSize calculates the position size based on available cash and target allocation
func (s *MultiIndicatorStrategy) calculatePositionSize(cash, price, allocation float64) float64 {
	targetValue := cash * allocation
	quantity := targetValue / price
	// Round down to nearest whole number (can't buy fractional shares)
	return float64(int(quantity))
}
