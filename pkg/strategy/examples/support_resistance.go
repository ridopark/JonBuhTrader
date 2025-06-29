package examples

import (
	"math"
	"os"
	"sort"
	"strconv"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// SupportResistanceLevel represents a support or resistance level
type SupportResistanceLevel struct {
	Price          float64
	Strength       int     // Number of times this level has been tested
	LastTouch      int     // Bar index of last touch
	Type           string  // "support" or "resistance"
	Volume         float64 // Average volume at this level
	Timeframe      string  // "short", "medium", "long" - timeframe where level was identified
	Age            int     // How many bars since the level was last reinforced
	Confidence     float64 // Confidence score (0.0 to 1.0)
	BreakoutFailed bool    // Whether a previous breakout of this level failed
}

// SupportResistanceStrategy implements a strategy based on support and resistance levels
type SupportResistanceStrategy struct {
	*strategy.BaseStrategy
	lookbackPeriod       int
	minTouches           int
	levelTolerance       float64
	breakoutConfirmation int
	positionSize         float64
	stopLoss             float64
	takeProfit           float64
	minLevelStrength     int
	useVolumeFilter      bool
	volumeMultiplier     float64

	// Enhanced features
	adaptiveTolerance   bool    // Use volatility-based tolerance
	trendAware          bool    // Consider trend direction
	maxLevelAge         int     // Maximum age for levels before removal
	multiTimeframe      bool    // Use multiple timeframes
	volatilityPeriod    int     // Period for volatility calculation
	confidenceThreshold float64 // Minimum confidence for trading

	// Internal state
	levels          map[string][]SupportResistanceLevel // Support/resistance levels per symbol
	priceHistory    map[string][]float64                // Price history per symbol
	volumeHistory   map[string][]float64                // Volume history per symbol
	volatility      map[string]float64                  // Current volatility per symbol
	trend           map[string]string                   // Current trend per symbol ("up", "down", "sideways")
	barCount        map[string]int                      // Bar count per symbol
	breakoutBars    map[string]int                      // Bars since breakout per symbol
	failedBreakouts map[string]map[float64]int          // Failed breakout attempts per level
}

// NewSupportResistanceStrategy creates a new support and resistance strategy
func NewSupportResistanceStrategy() *SupportResistanceStrategy {
	// Load configuration from environment variables
	lookbackPeriod := getEnvInt("SR_LOOKBACK_PERIOD", 20)
	minTouches := getEnvInt("SR_MIN_TOUCHES", 2)
	levelTolerance := getEnvFloat("SR_LEVEL_TOLERANCE", 0.5) / 100.0 // Convert percentage to decimal
	breakoutConfirmation := getEnvInt("SR_BREAKOUT_CONFIRMATION", 2)
	positionSize := getEnvFloat("SR_POSITION_SIZE", 0.95)
	stopLoss := getEnvFloat("SR_STOP_LOSS", 2.0) / 100.0     // Convert percentage to decimal
	takeProfit := getEnvFloat("SR_TAKE_PROFIT", 4.0) / 100.0 // Convert percentage to decimal
	minLevelStrength := getEnvInt("SR_MIN_LEVEL_STRENGTH", 3)
	useVolumeFilter := getEnvBool("SR_USE_VOLUME_FILTER", true)
	volumeMultiplier := getEnvFloat("SR_VOLUME_MULTIPLIER", 1.5)

	// Enhanced features
	adaptiveTolerance := getEnvBool("SR_ADAPTIVE_TOLERANCE", true)
	trendAware := getEnvBool("SR_TREND_AWARE", true)
	maxLevelAge := getEnvInt("SR_MAX_LEVEL_AGE", 50)
	multiTimeframe := getEnvBool("SR_MULTI_TIMEFRAME", true)
	volatilityPeriod := getEnvInt("SR_VOLATILITY_PERIOD", 14)
	confidenceThreshold := getEnvFloat("SR_CONFIDENCE_THRESHOLD", 0.6)

	base := strategy.NewBaseStrategy("SupportResistance", map[string]interface{}{
		"lookbackPeriod":       lookbackPeriod,
		"minTouches":           minTouches,
		"levelTolerance":       levelTolerance * 100, // Show as percentage in logs
		"breakoutConfirmation": breakoutConfirmation,
		"positionSize":         positionSize,
		"stopLoss":             stopLoss * 100,   // Show as percentage in logs
		"takeProfit":           takeProfit * 100, // Show as percentage in logs
		"minLevelStrength":     minLevelStrength,
		"useVolumeFilter":      useVolumeFilter,
		"volumeMultiplier":     volumeMultiplier,
		"adaptiveTolerance":    adaptiveTolerance,
		"trendAware":           trendAware,
		"maxLevelAge":          maxLevelAge,
		"multiTimeframe":       multiTimeframe,
		"volatilityPeriod":     volatilityPeriod,
		"confidenceThreshold":  confidenceThreshold,
	})

	return &SupportResistanceStrategy{
		BaseStrategy:         base,
		lookbackPeriod:       lookbackPeriod,
		minTouches:           minTouches,
		levelTolerance:       levelTolerance,
		breakoutConfirmation: breakoutConfirmation,
		positionSize:         positionSize,
		stopLoss:             stopLoss,
		takeProfit:           takeProfit,
		minLevelStrength:     minLevelStrength,
		useVolumeFilter:      useVolumeFilter,
		volumeMultiplier:     volumeMultiplier,
		adaptiveTolerance:    adaptiveTolerance,
		trendAware:           trendAware,
		maxLevelAge:          maxLevelAge,
		multiTimeframe:       multiTimeframe,
		volatilityPeriod:     volatilityPeriod,
		confidenceThreshold:  confidenceThreshold,
		levels:               make(map[string][]SupportResistanceLevel),
		priceHistory:         make(map[string][]float64),
		volumeHistory:        make(map[string][]float64),
		volatility:           make(map[string]float64),
		trend:                make(map[string]string),
		barCount:             make(map[string]int),
		breakoutBars:         make(map[string]int),
		failedBreakouts:      make(map[string]map[float64]int),
	}
}

// SetSymbols sets the symbols for this strategy
func (s *SupportResistanceStrategy) SetSymbols(symbols []string) {
	s.BaseStrategy.SetSymbols(symbols)

	// Initialize maps for each symbol
	for _, symbol := range symbols {
		s.levels[symbol] = []SupportResistanceLevel{}
		s.priceHistory[symbol] = []float64{}
		s.volumeHistory[symbol] = []float64{}
		s.volatility[symbol] = 0.0
		s.trend[symbol] = "sideways"
		s.barCount[symbol] = 0
		s.breakoutBars[symbol] = 0
		s.failedBreakouts[symbol] = make(map[float64]int)
	}
}

// Initialize sets up the strategy
func (s *SupportResistanceStrategy) Initialize(ctx strategy.Context) error {
	ctx.Log("info", "Support & Resistance Strategy initialized", map[string]interface{}{
		"strategy":             s.GetName(),
		"lookbackPeriod":       s.lookbackPeriod,
		"minTouches":           s.minTouches,
		"levelTolerance":       s.levelTolerance * 100,
		"breakoutConfirmation": s.breakoutConfirmation,
		"positionSize":         s.positionSize,
		"stopLoss":             s.stopLoss * 100,
		"takeProfit":           s.takeProfit * 100,
		"minLevelStrength":     s.minLevelStrength,
		"useVolumeFilter":      s.useVolumeFilter,
		"volumeMultiplier":     s.volumeMultiplier,
	})
	return nil
}

// PotentialSignal represents a potential trading signal with priority
type PotentialSignal struct {
	Symbol     string
	Bar        strategy.BarData
	Level      SupportResistanceLevel
	SignalType string // "support_bounce" or "resistance_breakout"
	Confidence float64
	Price      float64
}

// OnDataPoint processes each data point and generates trading signals
func (s *SupportResistanceStrategy) OnDataPoint(ctx strategy.Context, dataPoint strategy.DataPoint) ([]strategy.Order, error) {
	var orders []strategy.Order
	var potentialSignals []PotentialSignal

	// First pass: Update all data and collect exit signals
	for _, symbol := range s.GetSymbols() {
		bar, exists := dataPoint.Bars[symbol]
		if !exists {
			continue
		}

		// Update price and volume history
		s.priceHistory[symbol] = append(s.priceHistory[symbol], bar.Close)
		s.volumeHistory[symbol] = append(s.volumeHistory[symbol], bar.Volume)
		s.barCount[symbol]++

		// Keep only lookback period worth of data
		if len(s.priceHistory[symbol]) > s.lookbackPeriod*2 {
			s.priceHistory[symbol] = s.priceHistory[symbol][1:]
			s.volumeHistory[symbol] = s.volumeHistory[symbol][1:]
		}

		// Need enough data to identify levels
		if len(s.priceHistory[symbol]) < s.lookbackPeriod {
			continue
		}

		// Update support and resistance levels
		s.updateLevels(symbol, bar)

		// Update volatility and trend analysis
		s.updateVolatility(symbol, bar)
		s.updateTrend(symbol)
		s.ageLevels(symbol)

		position := ctx.GetPosition(symbol)

		// Handle nil position (no position exists)
		positionQuantity := 0.0
		if position != nil {
			positionQuantity = position.Quantity
		}

		// Check for stop loss or take profit if we have a position (high priority)
		if positionQuantity != 0 {
			stopOrder := s.checkStopLossTakeProfit(symbol, bar, position)
			if stopOrder != nil {
				orders = append(orders, *stopOrder)
				continue
			}
		}

		// Collect potential entry signals if no position
		if positionQuantity == 0 {
			signal := s.evaluateEntrySignal(symbol, bar)
			if signal != nil {
				potentialSignals = append(potentialSignals, *signal)
			}
		}

		// Increment breakout confirmation counter
		if s.breakoutBars[symbol] > 0 {
			s.breakoutBars[symbol]++
		}
	}

	// Second pass: Process entry signals with capital allocation
	if len(potentialSignals) > 0 {
		entryOrders := s.allocateCapitalToSignals(ctx, potentialSignals)
		orders = append(orders, entryOrders...)
	}

	return orders, nil
}

// updateLevels identifies and updates support and resistance levels
func (s *SupportResistanceStrategy) updateLevels(symbol string, bar strategy.BarData) {
	prices := s.priceHistory[symbol]
	if len(prices) < s.lookbackPeriod {
		return
	}

	// Find pivot highs and lows
	pivots := s.findPivots(prices)

	// Update existing levels and find new ones
	s.levels[symbol] = s.consolidateLevels(symbol, pivots)
}

// findPivots identifies pivot highs and lows in the price data
func (s *SupportResistanceStrategy) findPivots(prices []float64) []float64 {
	var pivots []float64
	lookback := 3 // Look 3 bars on each side for pivot confirmation

	for i := lookback; i < len(prices)-lookback; i++ {
		isPivotHigh := true
		isPivotLow := true

		// Check if it's a pivot high
		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && prices[j] >= prices[i] {
				isPivotHigh = false
				break
			}
		}

		// Check if it's a pivot low
		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && prices[j] <= prices[i] {
				isPivotLow = false
				break
			}
		}

		if isPivotHigh || isPivotLow {
			pivots = append(pivots, prices[i])
		}
	}

	return pivots
}

// consolidateLevels groups similar price levels and calculates their strength
func (s *SupportResistanceStrategy) consolidateLevels(symbol string, newPivots []float64) []SupportResistanceLevel {
	allPrices := append(newPivots, s.extractLevelPrices(s.levels[symbol])...)

	var consolidatedLevels []SupportResistanceLevel
	tolerance := s.getAdaptiveTolerance(symbol)

	// Sort prices for processing
	sort.Float64s(allPrices)

	i := 0
	for i < len(allPrices) {
		levelPrice := allPrices[i]
		strength := 1
		totalVolume := 0.0

		// Find all prices within tolerance of this level
		j := i + 1
		for j < len(allPrices) {
			if math.Abs(allPrices[j]-levelPrice)/levelPrice <= tolerance {
				strength++
				j++
			} else {
				break
			}
		}

		// Only keep levels with minimum touches
		if strength >= s.minTouches {
			levelType := s.determineLevelType(symbol, levelPrice)

			// Determine timeframe based on lookback period
			timeframe := "medium"
			if s.lookbackPeriod <= 10 {
				timeframe = "short"
			} else if s.lookbackPeriod >= 50 {
				timeframe = "long"
			}

			level := SupportResistanceLevel{
				Price:          levelPrice,
				Strength:       strength,
				LastTouch:      s.barCount[symbol],
				Type:           levelType,
				Volume:         totalVolume / float64(strength),
				Timeframe:      timeframe,
				Age:            0,
				Confidence:     0.0, // Will be calculated below
				BreakoutFailed: s.hasFailedBreakout(symbol, levelPrice),
			}

			// Calculate confidence score
			level.Confidence = s.calculateLevelConfidence(level, symbol)

			consolidatedLevels = append(consolidatedLevels, level)
		}

		i = j
	}

	return consolidatedLevels
}

// extractLevelPrices extracts prices from existing levels
func (s *SupportResistanceStrategy) extractLevelPrices(levels []SupportResistanceLevel) []float64 {
	var prices []float64
	for _, level := range levels {
		prices = append(prices, level.Price)
	}
	return prices
}

// determineLevelType determines if a level is support or resistance
func (s *SupportResistanceStrategy) determineLevelType(symbol string, levelPrice float64) string {
	prices := s.priceHistory[symbol]
	if len(prices) == 0 {
		return "unknown"
	}

	currentPrice := prices[len(prices)-1]

	if levelPrice < currentPrice {
		return "support"
	}
	return "resistance"
}

// evaluateEntrySignal evaluates if a symbol has a valid entry signal
func (s *SupportResistanceStrategy) evaluateEntrySignal(symbol string, bar strategy.BarData) *PotentialSignal {
	levels := s.levels[symbol]
	if len(levels) == 0 {
		return nil
	}

	tolerance := s.getAdaptiveTolerance(symbol)

	for _, level := range levels {
		// Enhanced filtering
		if level.Strength < s.minLevelStrength {
			continue
		}

		// Check confidence threshold
		if level.Confidence < s.confidenceThreshold {
			continue
		}

		// Check volatility-based entry conditions
		if !s.isVolatilityBasedEntry(symbol, level) {
			continue
		}

		// Check for bounce off support (buy signal)
		if level.Type == "support" && s.isPriceBouncingEnhanced(bar.Close, level.Price, true, tolerance) {
			if !s.checkTrendAlignment(symbol, true) {
				continue
			}

			if s.useVolumeFilter && !s.hasVolumeConfirmation(symbol) {
				continue
			}

			return &PotentialSignal{
				Symbol:     symbol,
				Bar:        bar,
				Level:      level,
				SignalType: "support_bounce",
				Confidence: level.Confidence,
				Price:      bar.Close,
			}
		}

		// Check for breakout above resistance (buy signal)
		if level.Type == "resistance" && s.isPriceBreakingEnhanced(bar.Close, level.Price, true, tolerance) {
			if !s.checkTrendAlignment(symbol, true) {
				continue
			}

			if s.useVolumeFilter && !s.hasVolumeConfirmation(symbol) {
				continue
			}

			return &PotentialSignal{
				Symbol:     symbol,
				Bar:        bar,
				Level:      level,
				SignalType: "resistance_breakout",
				Confidence: level.Confidence,
				Price:      bar.Close,
			}
		}
	}

	return nil
}

// allocateCapitalToSignals prioritizes and allocates capital to trading signals
func (s *SupportResistanceStrategy) allocateCapitalToSignals(ctx strategy.Context, signals []PotentialSignal) []strategy.Order {
	if len(signals) == 0 {
		return nil
	}

	var orders []strategy.Order
	availableCash := ctx.GetCash()

	// Sort signals by confidence score (highest first)
	sort.Slice(signals, func(i, j int) bool {
		return signals[i].Confidence > signals[j].Confidence
	})

	// Calculate how to split capital among signals
	maxPositions := len(signals)
	if maxPositions > 3 { // Limit to max 3 simultaneous positions
		maxPositions = 3
		signals = signals[:3] // Take only the top 3
	}

	// Allocate capital proportionally to confidence, but ensure we don't exceed available cash
	totalConfidence := 0.0
	for _, signal := range signals {
		totalConfidence += signal.Confidence
	}

	// Track remaining cash as we allocate
	remainingCash := availableCash

	for i, signal := range signals {
		if remainingCash <= 100 { // Need at least $100 to trade
			break
		}

		// Calculate allocation for this signal
		var allocation float64
		if i == len(signals)-1 {
			// Last signal gets whatever is left (up to position size limit)
			allocation = math.Min(s.positionSize, remainingCash/availableCash)
		} else {
			// Proportional allocation based on confidence
			confidenceWeight := signal.Confidence / totalConfidence
			baseAllocation := s.positionSize / float64(maxPositions)                // Equal base allocation
			confidenceBonus := (confidenceWeight - 1.0/float64(len(signals))) * 0.5 // Up to 50% bonus
			allocation = baseAllocation + confidenceBonus

			// Ensure allocation doesn't exceed remaining cash ratio
			maxAllocationForRemaining := remainingCash / availableCash
			allocation = math.Min(allocation, maxAllocationForRemaining)
		}

		// Calculate position size with volatility adjustment
		quantity := s.calculateVolatilityAdjustedPositionSize(signal.Symbol, remainingCash, signal.Price, allocation)

		if quantity > 0 {
			orderValue := quantity * signal.Price

			// Double-check we have enough cash (with some buffer for slippage/fees)
			if orderValue <= remainingCash*0.98 { // 2% buffer
				tolerance := s.getAdaptiveTolerance(signal.Symbol)

				ctx.Log("info", "Enhanced "+signal.SignalType+" BUY signal", map[string]interface{}{
					"symbol":        signal.Symbol,
					"price":         signal.Price,
					"level":         signal.Level.Price,
					"strength":      signal.Level.Strength,
					"confidence":    signal.Confidence,
					"tolerance":     tolerance * 100,
					"trend":         s.trend[signal.Symbol],
					"volatility":    s.volatility[signal.Symbol] * 100,
					"quantity":      quantity,
					"allocation":    allocation,
					"orderValue":    orderValue,
					"remainingCash": remainingCash,
				})

				orders = append(orders, strategy.Order{
					Symbol:   signal.Symbol,
					Side:     strategy.OrderSideBuy,
					Type:     strategy.OrderTypeMarket,
					Quantity: quantity,
					Strategy: s.GetName(),
				})

				// Update remaining cash and breakout tracking
				remainingCash -= orderValue
				if signal.SignalType == "resistance_breakout" {
					s.breakoutBars[signal.Symbol] = 1
				}
			} else {
				ctx.Log("warn", "Insufficient cash for signal", map[string]interface{}{
					"symbol":        signal.Symbol,
					"requiredValue": orderValue,
					"remainingCash": remainingCash,
					"skipping":      true,
				})
			}
		}
	}

	return orders
}

// OnFinish is called when the strategy finishes
func (s *SupportResistanceStrategy) OnFinish(ctx strategy.Context) error {
	// Log final levels for each symbol
	for symbol, levels := range s.levels {
		if len(levels) > 0 {
			ctx.Log("info", "Final support/resistance levels", map[string]interface{}{
				"symbol": symbol,
				"levels": len(levels),
			})
		}
	}

	ctx.Log("info", "Support & Resistance Strategy finished", map[string]interface{}{
		"finalCash": ctx.GetCash(),
	})
	return nil
}

// updateVolatility calculates current volatility for adaptive tolerance
func (s *SupportResistanceStrategy) updateVolatility(symbol string, bar strategy.BarData) {
	prices := s.priceHistory[symbol]
	if len(prices) < s.volatilityPeriod {
		s.volatility[symbol] = 0.02 // Default 2% volatility
		return
	}

	// Calculate ATR-based volatility
	var returns []float64
	for i := 1; i < len(prices); i++ {
		ret := math.Abs((prices[i] - prices[i-1]) / prices[i-1])
		returns = append(returns, ret)
	}

	// Calculate average of last volatilityPeriod returns
	start := len(returns) - s.volatilityPeriod
	if start < 0 {
		start = 0
	}

	var sum float64
	count := 0
	for i := start; i < len(returns); i++ {
		sum += returns[i]
		count++
	}

	if count > 0 {
		s.volatility[symbol] = sum / float64(count)
	}
}

// updateTrend determines the current trend direction
func (s *SupportResistanceStrategy) updateTrend(symbol string) {
	prices := s.priceHistory[symbol]
	if len(prices) < 20 {
		s.trend[symbol] = "sideways"
		return
	}

	// Simple trend detection using short vs long SMA
	shortPeriod := 10
	longPeriod := 20

	shortSMA := s.calculateSMA(prices, shortPeriod)
	longSMA := s.calculateSMA(prices, longPeriod)

	if shortSMA > longSMA*1.005 { // 0.5% threshold
		s.trend[symbol] = "up"
	} else if shortSMA < longSMA*0.995 {
		s.trend[symbol] = "down"
	} else {
		s.trend[symbol] = "sideways"
	}
}

// calculateSMA calculates simple moving average
func (s *SupportResistanceStrategy) calculateSMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	start := len(prices) - period
	var sum float64
	for i := start; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

// ageLevels increases age of levels and removes old ones
func (s *SupportResistanceStrategy) ageLevels(symbol string) {
	var activeLevels []SupportResistanceLevel

	for _, level := range s.levels[symbol] {
		level.Age++

		// Remove levels that are too old or have low confidence
		if level.Age <= s.maxLevelAge && level.Confidence >= s.confidenceThreshold {
			activeLevels = append(activeLevels, level)
		}
	}

	s.levels[symbol] = activeLevels
}

// getAdaptiveTolerance calculates tolerance based on volatility
func (s *SupportResistanceStrategy) getAdaptiveTolerance(symbol string) float64 {
	if !s.adaptiveTolerance {
		return s.levelTolerance
	}

	volatility := s.volatility[symbol]
	baseTolerance := s.levelTolerance

	// Adjust tolerance based on volatility (min 0.2%, max 2.0%)
	adaptedTolerance := baseTolerance + (volatility * 0.5)
	if adaptedTolerance < 0.002 {
		adaptedTolerance = 0.002
	}
	if adaptedTolerance > 0.020 {
		adaptedTolerance = 0.020
	}

	return adaptedTolerance
}

// calculateLevelConfidence calculates confidence score for a level
func (s *SupportResistanceStrategy) calculateLevelConfidence(level SupportResistanceLevel, symbol string) float64 {
	confidence := 0.0

	// Base confidence from strength
	confidence += float64(level.Strength) * 0.2
	if confidence > 0.6 {
		confidence = 0.6
	}

	// Age factor (newer levels are more confident)
	ageFactor := 1.0 - (float64(level.Age) / float64(s.maxLevelAge))
	if ageFactor < 0 {
		ageFactor = 0
	}
	confidence += ageFactor * 0.3

	// Volume factor
	if level.Volume > 0 {
		confidence += 0.1
	}

	// Failed breakout penalty
	if level.BreakoutFailed {
		confidence += 0.1 // Actually increases confidence if breakout failed
	}

	// Trend alignment bonus
	currentTrend := s.trend[symbol]
	if (level.Type == "support" && currentTrend == "up") ||
		(level.Type == "resistance" && currentTrend == "down") {
		confidence += 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// isVolatilityBasedEntry checks if entry conditions are met considering volatility
func (s *SupportResistanceStrategy) isVolatilityBasedEntry(symbol string, level SupportResistanceLevel) bool {
	volatility := s.volatility[symbol]

	// In high volatility, require higher confidence
	if volatility > 0.03 { // 3% daily volatility
		return level.Confidence >= 0.8
	}

	// In low volatility, standard confidence is fine
	return level.Confidence >= s.confidenceThreshold
}

// checkTrendAlignment verifies if trade aligns with trend
func (s *SupportResistanceStrategy) checkTrendAlignment(symbol string, isBuySignal bool) bool {
	if !s.trendAware {
		return true
	}

	trend := s.trend[symbol]

	// Only allow buy signals in uptrend or sideways market
	if isBuySignal {
		return trend == "up" || trend == "sideways"
	}

	// Only allow sell signals in downtrend or sideways market
	return trend == "down" || trend == "sideways"
}

// hasFailedBreakout checks if a level has had failed breakout attempts
func (s *SupportResistanceStrategy) hasFailedBreakout(symbol string, levelPrice float64) bool {
	if failedCounts, exists := s.failedBreakouts[symbol]; exists {
		tolerance := s.getAdaptiveTolerance(symbol) * levelPrice
		for price, count := range failedCounts {
			if math.Abs(price-levelPrice) <= tolerance && count > 0 {
				return true
			}
		}
	}
	return false
}

// recordFailedBreakout records a failed breakout attempt
func (s *SupportResistanceStrategy) recordFailedBreakout(symbol string, levelPrice float64) {
	if s.failedBreakouts[symbol] == nil {
		s.failedBreakouts[symbol] = make(map[float64]int)
	}
	s.failedBreakouts[symbol][levelPrice]++
}

// checkStopLossTakeProfit checks for stop loss or take profit conditions
func (s *SupportResistanceStrategy) checkStopLossTakeProfit(symbol string, bar strategy.BarData, position *strategy.Position) *strategy.Order {
	if position.Quantity == 0 {
		return nil
	}

	currentPrice := bar.Close
	entryPrice := position.AvgPrice

	// Calculate P&L percentage
	var pnlPercent float64
	if position.Quantity > 0 { // Long position
		pnlPercent = (currentPrice - entryPrice) / entryPrice
	} else { // Short position (if supported)
		pnlPercent = (entryPrice - currentPrice) / entryPrice
	}

	// Check stop loss
	if pnlPercent <= -s.stopLoss {
		// Record failed breakout if we're stopping out shortly after a breakout
		if s.breakoutBars[symbol] > 0 && s.breakoutBars[symbol] <= s.breakoutConfirmation {
			s.recordFailedBreakout(symbol, entryPrice)
		}

		return &strategy.Order{
			Symbol:   symbol,
			Side:     strategy.OrderSideSell,
			Type:     strategy.OrderTypeMarket,
			Quantity: math.Abs(position.Quantity),
			Strategy: s.GetName(),
		}
	}

	// Check take profit
	if pnlPercent >= s.takeProfit {
		return &strategy.Order{
			Symbol:   symbol,
			Side:     strategy.OrderSideSell,
			Type:     strategy.OrderTypeMarket,
			Quantity: math.Abs(position.Quantity),
			Strategy: s.GetName(),
		}
	}

	return nil
}

// isPriceBouncingEnhanced checks if price is bouncing off a level with adaptive tolerance
func (s *SupportResistanceStrategy) isPriceBouncingEnhanced(currentPrice, levelPrice float64, isSupport bool, tolerance float64) bool {
	toleranceAmount := levelPrice * tolerance

	if isSupport {
		// Price should be near support level (within tolerance) and moving up
		return currentPrice >= levelPrice-toleranceAmount && currentPrice <= levelPrice+toleranceAmount
	} else {
		// Price should be near resistance level (within tolerance) and moving down
		return currentPrice >= levelPrice-toleranceAmount && currentPrice <= levelPrice+toleranceAmount
	}
}

// isPriceBreakingEnhanced checks if price is breaking through a level with adaptive tolerance
func (s *SupportResistanceStrategy) isPriceBreakingEnhanced(currentPrice, levelPrice float64, isUpward bool, tolerance float64) bool {
	breakoutThreshold := levelPrice * tolerance

	if isUpward {
		// Breaking above resistance
		return currentPrice > levelPrice+breakoutThreshold
	} else {
		// Breaking below support
		return currentPrice < levelPrice-breakoutThreshold
	}
}

// hasVolumeConfirmation checks if there's volume confirmation for the signal
func (s *SupportResistanceStrategy) hasVolumeConfirmation(symbol string) bool {
	volumes := s.volumeHistory[symbol]
	if len(volumes) < 10 {
		return true // Not enough data, assume confirmation
	}

	currentVolume := volumes[len(volumes)-1]

	// Calculate average volume over last 10 bars (excluding current)
	var avgVolume float64
	for i := len(volumes) - 10; i < len(volumes)-1; i++ {
		avgVolume += volumes[i]
	}
	avgVolume /= 9

	return currentVolume >= avgVolume*s.volumeMultiplier
}

// calculateVolatilityAdjustedPositionSize calculates position size adjusted for volatility
func (s *SupportResistanceStrategy) calculateVolatilityAdjustedPositionSize(symbol string, cash, price, allocation float64) float64 {
	volatility := s.volatility[symbol]

	// Reduce position size in high volatility environments
	volatilityAdjustment := 1.0
	if volatility > 0.03 { // 3% daily volatility
		volatilityAdjustment = 0.7 // Reduce to 70% of normal size
	} else if volatility > 0.02 { // 2% daily volatility
		volatilityAdjustment = 0.85 // Reduce to 85% of normal size
	}

	adjustedAllocation := allocation * volatilityAdjustment
	targetValue := cash * adjustedAllocation
	quantity := targetValue / price

	// Round down to nearest whole number (can't buy fractional shares)
	return float64(int(quantity))
}

// Helper functions for reading environment variables

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
