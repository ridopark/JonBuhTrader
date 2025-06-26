package feed

import (
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// DataFeed defines the interface for providing market data
type DataFeed interface {
	// Initialize sets up the data feed
	Initialize() error

	// GetNextBar returns the next bar of data, or nil if no more data
	GetNextBar() (*strategy.BarData, error)

	// HasMoreData returns true if there's more data available
	HasMoreData() bool

	// Reset resets the feed to the beginning
	Reset() error

	// Close closes the data feed
	Close() error

	// GetSymbols returns the symbols available in this feed
	GetSymbols() []string

	// GetTimeframe returns the timeframe of the data
	GetTimeframe() string
}

// HistoricalDataProvider defines the interface for historical data sources
type HistoricalDataProvider interface {
	// GetBars retrieves historical OHLCV data for the given parameters
	GetBars(symbol string, timeframe string, start time.Time, end time.Time) ([]strategy.BarData, error)

	// GetLastBar gets the most recent bar for a symbol
	GetLastBar(symbol string, timeframe string) (*strategy.BarData, error)

	// GetBarsLimit gets the last N bars for a symbol
	GetBarsLimit(symbol string, timeframe string, limit int) ([]strategy.BarData, error)
}
