package feed

import (
	"fmt"
	"sort"
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/logging"
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
	"github.com/rs/zerolog"
)

// HistoricalFeed provides historical market data for backtesting
type HistoricalFeed struct {
	provider  HistoricalDataProvider
	symbols   []string
	timeframe string
	startDate time.Time
	endDate   time.Time
	logger    zerolog.Logger

	// Internal state
	dataPoints  []strategy.DataPoint
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
		logger:     logging.GetLogger("historical-feed"),
		dataPoints: make([]strategy.DataPoint, 0),
		currentIdx: 0,
	}
}

// Initialize loads all historical data and groups it by timestamp
func (hf *HistoricalFeed) Initialize() error {
	if hf.initialized {
		return nil
	}

	hf.logger.Debug().Msg("Initializing historical feed data")

	// Load data for all symbols
	allBars := make(map[string][]strategy.BarData)
	for _, symbol := range hf.symbols {
		hf.logger.Debug().Str("symbol", symbol).Msg("Loading data for symbol")

		bars, err := hf.provider.GetBars(symbol, hf.timeframe, hf.startDate, hf.endDate)
		if err != nil {
			return fmt.Errorf("failed to load data for symbol %s: %w", symbol, err)
		}

		allBars[symbol] = bars
		hf.logger.Debug().Int("bars_loaded", len(bars)).Str("symbol", symbol).Msg("Data loaded")
	}

	// Create a map to group bars by timestamp
	timestampMap := make(map[time.Time]map[string]strategy.BarData)

	// Group bars by timestamp
	for symbol, bars := range allBars {
		for _, bar := range bars {
			if timestampMap[bar.Timestamp] == nil {
				timestampMap[bar.Timestamp] = make(map[string]strategy.BarData)
			}
			timestampMap[bar.Timestamp][symbol] = bar
		}
	}

	// Convert to sorted slice of DataPoints
	timestamps := make([]time.Time, 0, len(timestampMap))
	for timestamp := range timestampMap {
		timestamps = append(timestamps, timestamp)
	}

	// Sort timestamps
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

	// Create DataPoints in chronological order
	for _, timestamp := range timestamps {
		// Only create datapoint if we have data for all symbols at this timestamp
		symbolBars := timestampMap[timestamp]
		if len(symbolBars) == len(hf.symbols) {
			hf.dataPoints = append(hf.dataPoints, strategy.DataPoint{
				Timestamp: timestamp,
				Bars:      symbolBars,
			})
		} else {
			// Log missing data for debugging
			missingSymbols := make([]string, 0)
			for _, symbol := range hf.symbols {
				if _, exists := symbolBars[symbol]; !exists {
					missingSymbols = append(missingSymbols, symbol)
				}
			}
			hf.logger.Debug().
				Time("timestamp", timestamp).
				Strs("missing_symbols", missingSymbols).
				Msg("Skipping datapoint with incomplete data")
		}
	}

	hf.logger.Info().
		Int("total_datapoints", len(hf.dataPoints)).
		Int("symbols", len(hf.symbols)).
		Msg("Historical feed initialized")

	hf.initialized = true
	return nil
}

// GetNextDataPoint returns the next chronological datapoint with bars for all symbols
func (hf *HistoricalFeed) GetNextDataPoint() (*strategy.DataPoint, error) {
	if !hf.initialized {
		if err := hf.Initialize(); err != nil {
			return nil, err
		}
	}

	if hf.currentIdx >= len(hf.dataPoints) {
		return nil, nil // No more data
	}

	dataPoint := hf.dataPoints[hf.currentIdx]
	hf.currentIdx++

	hf.logger.Debug().
		Time("timestamp", dataPoint.Timestamp).
		Int("symbols_count", len(dataPoint.Bars)).
		Msg("Providing next datapoint")

	return &dataPoint, nil
}

// HasMoreData returns true if there's more data available
func (hf *HistoricalFeed) HasMoreData() bool {
	if !hf.initialized {
		return true // Assume there's data until we try to initialize
	}

	return hf.currentIdx < len(hf.dataPoints)
}

// Reset resets the feed to the beginning
func (hf *HistoricalFeed) Reset() error {
	hf.logger.Info().Msg("Resetting historical feed")
	hf.currentIdx = 0
	return nil
}

// Close closes the data feed (no-op for historical feed)
func (hf *HistoricalFeed) Close() error {
	hf.logger.Info().Msg("Closing historical feed")
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

// GetTotalDataPoints returns the total number of datapoints loaded
func (hf *HistoricalFeed) GetTotalDataPoints() int {
	return len(hf.dataPoints)
}

// GetProgress returns the current progress as a percentage
func (hf *HistoricalFeed) GetProgress() float64 {
	if len(hf.dataPoints) == 0 {
		return 0
	}

	return float64(hf.currentIdx) / float64(len(hf.dataPoints)) * 100
}

// GetCurrentTimestamp returns the timestamp of the current datapoint
func (hf *HistoricalFeed) GetCurrentTimestamp() *time.Time {
	if hf.currentIdx == 0 || hf.currentIdx > len(hf.dataPoints) {
		return nil
	}

	timestamp := hf.dataPoints[hf.currentIdx-1].Timestamp
	return &timestamp
}

// GetDateRange returns the actual date range of the loaded data
func (hf *HistoricalFeed) GetDateRange() (time.Time, time.Time) {
	if len(hf.dataPoints) == 0 {
		return time.Time{}, time.Time{}
	}

	return hf.dataPoints[0].Timestamp, hf.dataPoints[len(hf.dataPoints)-1].Timestamp
}
