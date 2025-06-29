package examples

import (
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// MACrossoverSignalImpl implements strategy.TradingSignal for MA Crossover signals
type MACrossoverSignalImpl struct {
	Symbol     string
	Bar        strategy.BarData
	SignalType string // "bullish_crossover"
	Price      float64
	ShortMA    float64
	LongMA     float64
	Confidence float64
	Priority   float64
}

func (s MACrossoverSignalImpl) GetSymbol() string {
	return s.Symbol
}

func (s MACrossoverSignalImpl) GetPrice() float64 {
	return s.Price
}

func (s MACrossoverSignalImpl) GetConfidence() float64 {
	return s.Confidence
}

func (s MACrossoverSignalImpl) GetSignalType() string {
	return s.SignalType
}

func (s MACrossoverSignalImpl) GetBarData() strategy.BarData {
	return s.Bar
}

func (s MACrossoverSignalImpl) GetPriority() float64 {
	return s.Priority
}

// MultiIndicatorSignalImpl implements strategy.TradingSignal for Multi-Indicator signals
type MultiIndicatorSignalImpl struct {
	Symbol     string
	Bar        strategy.BarData
	SignalType string // "buy" or "sell"
	Price      float64
	RSI        float64
	SMA        float64
	EMA        float64
	MACD       float64
	MACDSignal float64
	MACDHisto  float64
	Confidence float64
	Priority   float64
}

func (s MultiIndicatorSignalImpl) GetSymbol() string {
	return s.Symbol
}

func (s MultiIndicatorSignalImpl) GetPrice() float64 {
	return s.Price
}

func (s MultiIndicatorSignalImpl) GetConfidence() float64 {
	return s.Confidence
}

func (s MultiIndicatorSignalImpl) GetSignalType() string {
	return s.SignalType
}

func (s MultiIndicatorSignalImpl) GetBarData() strategy.BarData {
	return s.Bar
}

func (s MultiIndicatorSignalImpl) GetPriority() float64 {
	return s.Priority
}

// SupportResistanceSignalImpl implements strategy.TradingSignal for Support/Resistance signals
type SupportResistanceSignalImpl struct {
	Symbol     string
	Bar        strategy.BarData
	Level      SupportResistanceLevel
	SignalType string // "support_bounce" or "resistance_breakout"
	Price      float64
	Confidence float64
	Priority   float64
}

func (s SupportResistanceSignalImpl) GetSymbol() string {
	return s.Symbol
}

func (s SupportResistanceSignalImpl) GetPrice() float64 {
	return s.Price
}

func (s SupportResistanceSignalImpl) GetConfidence() float64 {
	return s.Confidence
}

func (s SupportResistanceSignalImpl) GetSignalType() string {
	return s.SignalType
}

func (s SupportResistanceSignalImpl) GetBarData() strategy.BarData {
	return s.Bar
}

func (s SupportResistanceSignalImpl) GetPriority() float64 {
	return s.Priority
}
