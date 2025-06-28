package strategy

import (
	"time"
)

// BarData represents OHLCV data for a single time period
type BarData struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Timeframe string
}

// DataPoint represents market data for all symbols at a specific timestamp
type DataPoint struct {
	Timestamp time.Time
	Bars      map[string]BarData // symbol -> bar data
}

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeStop   OrderType = "STOP"
)

// Order represents a trading order
type Order struct {
	ID        string
	Symbol    string
	Side      OrderSide
	Type      OrderType
	Quantity  float64
	Price     float64 // For limit orders
	StopPrice float64 // For stop orders
	Timestamp time.Time
	Strategy  string
}

// TradeEvent represents a completed trade
type TradeEvent struct {
	ID         string
	OrderID    string
	Symbol     string
	Side       OrderSide
	Quantity   float64
	Price      float64
	Timestamp  time.Time
	Commission float64
	SecFee     float64 // SEC Transaction Fee
	FinraTaf   float64 // FINRA Trading Activity Fee
	Slippage   float64 // Slippage cost
	Strategy   string
}

// Position represents a current position in a symbol
type Position struct {
	Symbol       string
	Quantity     float64
	AvgPrice     float64
	MarketValue  float64
	UnrealizedPL float64
	RealizedPL   float64
}

// Portfolio represents the current portfolio state
type Portfolio struct {
	Cash       float64
	TotalValue float64
	Positions  map[string]*Position
	TotalPL    float64
	DayPL      float64
	Trades     []TradeEvent
}

// Context provides strategy access to market data and portfolio state
type Context interface {
	// Portfolio access
	GetPortfolio() *Portfolio
	GetPosition(symbol string) *Position
	GetCash() float64

	// Historical data access
	GetBars(symbol string, timeframe string, limit int) ([]BarData, error)
	GetLastBar(symbol string, timeframe string) (*BarData, error)

	// Technical indicators (to be implemented)
	SMA(symbol string, period int) (float64, error)
	EMA(symbol string, period int) (float64, error)
	RSI(symbol string, period int) (float64, error)

	// Logging
	Log(level string, message string, fields map[string]interface{})
}

// Strategy defines the interface that all trading strategies must implement
type Strategy interface {
	// Initialize is called once before the strategy starts
	Initialize(ctx Context) error

	// OnDatapoint is called for each new datapoint of market data
	// Returns a slice of orders to be executed
	OnDataPoint(ctx Context, datapoint DataPoint) ([]Order, error)

	// OnTrade is called when a trade is executed
	OnTrade(ctx Context, trade TradeEvent) error

	// Cleanup is called when the strategy is shutting down
	Cleanup(ctx Context) error

	// GetName returns the strategy name
	GetName() string

	// GetParameters returns the strategy parameters
	GetParameters() map[string]interface{}
}

// StrategyConfig holds configuration for a strategy
type StrategyConfig struct {
	Name       string                 `yaml:"name"`
	Parameters map[string]interface{} `yaml:"parameters"`
	Symbols    []string               `yaml:"symbols"`
	Timeframe  string                 `yaml:"timeframe"`
}
