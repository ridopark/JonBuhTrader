package feed

import (
	"fmt"
	"sort"
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// HistoricalFeed provides historical market data for backtesting
type HistoricalFeed struct {
	provider  HistoricalDataProvider
	symbols   []string
	timeframe string
	startDate time.Time
	endDate   time.Time

	// Internal state
	allBars     []strategy.BarData
	currentIdx  int
	initialized bool
}

// NewHistoricalFeed creates a new historical data feed
func NewHistoricalFeed(provider HistoricalDataProvider, symbols []string, timeframe string, start, end time.Time) *HistoricalFeed {
	return &HistoricalFeed{
		provider:   provider,
		symbols:    symbols,
		timeframe:  timeframe,
		startDate:  start,
		endDate:    end,
		allBars:    make([]strategy.BarData, 0),
		currentIdx: 0,
	}
}

// Initialize loads all historical data and sorts it by timestamp
func (hf *HistoricalFeed) Initialize() error {
	if hf.initialized {
		return nil
	}

	// Load data for all symbols
	for _, symbol := range hf.symbols {
		bars, err := hf.provider.GetBars(symbol, hf.timeframe, hf.startDate, hf.endDate)
		if err != nil {
			return fmt.Errorf("failed to load data for symbol %s: %w", symbol, err)
		}

		hf.allBars = append(hf.allBars, bars...)
	}

	// Sort all bars by timestamp to ensure chronological order
	sort.Slice(hf.allBars, func(i, j int) bool {
		return hf.allBars[i].Timestamp.Before(hf.allBars[j].Timestamp)
	})

	hf.initialized = true
	return nil
}

// GetNextBar returns the next chronological bar from any symbol
func (hf *HistoricalFeed) GetNextBar() (*strategy.BarData, error) {
	if !hf.initialized {
		if err := hf.Initialize(); err != nil {
			return nil, err
		}
	}

	if hf.currentIdx >= len(hf.allBars) {
		return nil, nil // No more data
	}

	bar := hf.allBars[hf.currentIdx]
	hf.currentIdx++

	return &bar, nil
}

// HasMoreData returns true if there's more data available
func (hf *HistoricalFeed) HasMoreData() bool {
	if !hf.initialized {
		return true // Assume there's data until we try to initialize
	}

	return hf.currentIdx < len(hf.allBars)
}

// Reset resets the feed to the beginning
func (hf *HistoricalFeed) Reset() error {
	hf.currentIdx = 0
	return nil
}

// Close closes the data feed (no-op for historical feed)
func (hf *HistoricalFeed) Close() error {
	return nil
}

// GetSymbols returns the symbols in this feed
func (hf *HistoricalFeed) GetSymbols() []string {
	return hf.symbols
}

// GetTimeframe returns the timeframe of the data
func (hf *HistoricalFeed) GetTimeframe() string {
	return hf.timeframe
}

// GetTotalBars returns the total number of bars loaded
func (hf *HistoricalFeed) GetTotalBars() int {
	return len(hf.allBars)
}

// GetProgress returns the current progress as a percentage
func (hf *HistoricalFeed) GetProgress() float64 {
	if len(hf.allBars) == 0 {
		return 0
	}

	return float64(hf.currentIdx) / float64(len(hf.allBars)) * 100
}

// GetCurrentTimestamp returns the timestamp of the current bar
func (hf *HistoricalFeed) GetCurrentTimestamp() *time.Time {
	if hf.currentIdx == 0 || hf.currentIdx > len(hf.allBars) {
		return nil
	}

	timestamp := hf.allBars[hf.currentIdx-1].Timestamp
	return &timestamp
}

// GetDateRange returns the actual date range of the loaded data
func (hf *HistoricalFeed) GetDateRange() (time.Time, time.Time) {
	if len(hf.allBars) == 0 {
		return time.Time{}, time.Time{}
	}

	return hf.allBars[0].Timestamp, hf.allBars[len(hf.allBars)-1].Timestamp
}
