package data

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/ridopark/JonBuhTrader/pkg/feed"
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// TimescaleDBProvider provides historical data from TimescaleDB
type TimescaleDBProvider struct {
	db *sql.DB
}

// NewTimescaleDBProvider creates a new TimescaleDB data provider
func NewTimescaleDBProvider(connectionString string) (*TimescaleDBProvider, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &TimescaleDBProvider{
		db: db,
	}, nil
}

// GetBars retrieves historical OHLCV data for the given parameters
func (p *TimescaleDBProvider) GetBars(symbol string, timeframe string, start time.Time, end time.Time) ([]strategy.BarData, error) {
	query := `
		SELECT symbol, timestamp, open, high, low, close, volume, timeframe
		FROM ohlcv_data 
		WHERE symbol = $1 AND timeframe = $2 AND timestamp >= $3 AND timestamp <= $4
		ORDER BY timestamp ASC
	`

	rows, err := p.db.Query(query, symbol, timeframe, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query ohlcv_data: %w", err)
	}
	defer rows.Close()

	var bars []strategy.BarData
	for rows.Next() {
		var bar strategy.BarData
		err := rows.Scan(
			&bar.Symbol,
			&bar.Timestamp,
			&bar.Open,
			&bar.High,
			&bar.Low,
			&bar.Close,
			&bar.Volume,
			&bar.Timeframe,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		bars = append(bars, bar)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return bars, nil
}

// GetLastBar gets the most recent bar for a symbol
func (p *TimescaleDBProvider) GetLastBar(symbol string, timeframe string) (*strategy.BarData, error) {
	query := `
		SELECT symbol, timestamp, open, high, low, close, volume, timeframe
		FROM ohlcv_data 
		WHERE symbol = $1 AND timeframe = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	row := p.db.QueryRow(query, symbol, timeframe)

	var bar strategy.BarData
	err := row.Scan(
		&bar.Symbol,
		&bar.Timestamp,
		&bar.Open,
		&bar.High,
		&bar.Low,
		&bar.Close,
		&bar.Volume,
		&bar.Timeframe,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no data found for symbol %s timeframe %s", symbol, timeframe)
		}
		return nil, fmt.Errorf("failed to get last bar: %w", err)
	}

	return &bar, nil
}

// GetBarsLimit gets the last N bars for a symbol
func (p *TimescaleDBProvider) GetBarsLimit(symbol string, timeframe string, limit int) ([]strategy.BarData, error) {
	query := `
		SELECT symbol, timestamp, open, high, low, close, volume, timeframe
		FROM ohlcv_data 
		WHERE symbol = $1 AND timeframe = $2
		ORDER BY timestamp DESC
		LIMIT $3
	`

	rows, err := p.db.Query(query, symbol, timeframe, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query ohlcv_data: %w", err)
	}
	defer rows.Close()

	var bars []strategy.BarData
	for rows.Next() {
		var bar strategy.BarData
		err := rows.Scan(
			&bar.Symbol,
			&bar.Timestamp,
			&bar.Open,
			&bar.High,
			&bar.Low,
			&bar.Close,
			&bar.Volume,
			&bar.Timeframe,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		bars = append(bars, bar)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Reverse the slice to get chronological order (oldest first)
	for i, j := 0, len(bars)-1; i < j; i, j = i+1, j-1 {
		bars[i], bars[j] = bars[j], bars[i]
	}

	return bars, nil
}

// Close closes the database connection
func (p *TimescaleDBProvider) Close() error {
	return p.db.Close()
}

// Verify that TimescaleDBProvider implements the HistoricalDataProvider interface
var _ feed.HistoricalDataProvider = (*TimescaleDBProvider)(nil)
