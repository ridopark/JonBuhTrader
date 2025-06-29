package main

import (
	"fmt"
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/backtester"
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
	"github.com/ridopark/JonBuhTrader/pkg/strategy/examples"
)

// MockDataFeed implements a simple data feed for testing
type MockDataFeed struct {
	data      []strategy.DataPoint
	idx       int
	symbols   []string
	timeframe string
}

func (m *MockDataFeed) Initialize() error {
	m.idx = 0
	return nil
}

func (m *MockDataFeed) GetNextDataPoint() (*strategy.DataPoint, error) {
	if m.idx >= len(m.data) {
		return nil, fmt.Errorf("no more data")
	}
	dp := m.data[m.idx]
	m.idx++
	return &dp, nil
}

func (m *MockDataFeed) HasMoreData() bool {
	return m.idx < len(m.data)
}

func (m *MockDataFeed) Reset() error {
	m.idx = 0
	return nil
}

func (m *MockDataFeed) Close() error {
	return nil
}

func (m *MockDataFeed) GetSymbols() []string {
	return m.symbols
}

func (m *MockDataFeed) GetTimeframe() string {
	return m.timeframe
}

// TestMACrossoverCapitalAllocation creates a test to verify capital allocation fix
func main() {
	fmt.Println("Testing MA Crossover Capital Allocation...")

	// Test symbols
	symbols := []string{"AAPL", "MSFT", "GOOGL"}

	// Create MA Crossover strategy
	maStrategy := examples.NewMovingAverageCrossoverStrategy(5, 20)
	maStrategy.SetSymbols(symbols)

	// Create test data that will generate crossover signals for all symbols
	// We'll create data where short MA crosses above long MA for all symbols simultaneously
	testData := []strategy.DataPoint{
		// Initial data - no crossover yet (short MA below long MA)
		{time.Now().Add(-24 * time.Hour), map[string]strategy.BarData{
			"AAPL":  {Symbol: "AAPL", Timestamp: time.Now().Add(-24 * time.Hour), Open: 150.0, High: 151.5, Low: 148.5, Close: 150.0, Volume: 1000, Timeframe: "1h"},
			"MSFT":  {Symbol: "MSFT", Timestamp: time.Now().Add(-24 * time.Hour), Open: 300.0, High: 303.0, Low: 297.0, Close: 300.0, Volume: 1000, Timeframe: "1h"},
			"GOOGL": {Symbol: "GOOGL", Timestamp: time.Now().Add(-24 * time.Hour), Open: 2500.0, High: 2525.0, Low: 2475.0, Close: 2500.0, Volume: 1000, Timeframe: "1h"},
		}},
		{time.Now().Add(-23 * time.Hour), map[string]strategy.BarData{
			"AAPL":  {Symbol: "AAPL", Timestamp: time.Now().Add(-23 * time.Hour), Open: 149.0, High: 150.5, Low: 147.5, Close: 149.0, Volume: 1000, Timeframe: "1h"},
			"MSFT":  {Symbol: "MSFT", Timestamp: time.Now().Add(-23 * time.Hour), Open: 299.0, High: 302.0, Low: 296.0, Close: 299.0, Volume: 1000, Timeframe: "1h"},
			"GOOGL": {Symbol: "GOOGL", Timestamp: time.Now().Add(-23 * time.Hour), Open: 2490.0, High: 2515.0, Low: 2465.0, Close: 2490.0, Volume: 1000, Timeframe: "1h"},
		}},
		// Add more data points to build up moving averages
		{time.Now().Add(-22 * time.Hour), map[string]strategy.BarData{
			"AAPL":  {Symbol: "AAPL", Timestamp: time.Now().Add(-22 * time.Hour), Open: 148.0, High: 149.5, Low: 146.5, Close: 148.0, Volume: 1000, Timeframe: "1h"},
			"MSFT":  {Symbol: "MSFT", Timestamp: time.Now().Add(-22 * time.Hour), Open: 298.0, High: 301.0, Low: 295.0, Close: 298.0, Volume: 1000, Timeframe: "1h"},
			"GOOGL": {Symbol: "GOOGL", Timestamp: time.Now().Add(-22 * time.Hour), Open: 2480.0, High: 2505.0, Low: 2455.0, Close: 2480.0, Volume: 1000, Timeframe: "1h"},
		}},
	}

	// Add more data points...
	for i := 21; i >= 1; i-- {
		timestamp := time.Now().Add(time.Duration(-i) * time.Hour)
		// Decreasing prices for the first part
		aaplPrice := 150.0 - float64(24-i)*2.0
		msftPrice := 300.0 - float64(24-i)*4.0
		googlPrice := 2500.0 - float64(24-i)*20.0

		// Then sudden price jumps for the last few hours to trigger crossovers
		if i <= 5 {
			aaplPrice = 150.0 + float64(5-i)*8.0 // Sharp upward movement
			msftPrice = 300.0 + float64(5-i)*16.0
			googlPrice = 2500.0 + float64(5-i)*100.0
		}

		testData = append(testData, strategy.DataPoint{
			Timestamp: timestamp,
			Bars: map[string]strategy.BarData{
				"AAPL":  {Symbol: "AAPL", Timestamp: timestamp, Open: aaplPrice, High: aaplPrice * 1.01, Low: aaplPrice * 0.99, Close: aaplPrice, Volume: 1000, Timeframe: "1h"},
				"MSFT":  {Symbol: "MSFT", Timestamp: timestamp, Open: msftPrice, High: msftPrice * 1.01, Low: msftPrice * 0.99, Close: msftPrice, Volume: 1000, Timeframe: "1h"},
				"GOOGL": {Symbol: "GOOGL", Timestamp: timestamp, Open: googlPrice, High: googlPrice * 1.01, Low: googlPrice * 0.99, Close: googlPrice, Volume: 1000, Timeframe: "1h"},
			},
		})
	}

	// Create mock data feed
	dataFeed := &MockDataFeed{
		data:      testData,
		symbols:   symbols,
		timeframe: "1h",
	}

	// Create engine with $10,000 initial capital
	initialCapital := 10000.0
	engine := backtester.NewEngine(maStrategy, dataFeed, initialCapital)

	fmt.Printf("Initial capital: $%.2f\n", initialCapital)
	fmt.Printf("Strategy: MA Crossover (5/20) with symbols: %v\n\n", symbols)

	// Run the backtest
	err := engine.Run()
	if err != nil {
		fmt.Printf("Backtest failed: %v\n", err)
		return
	}

	// Get results
	results := engine.GetResults()

	fmt.Printf("Backtest completed!\n")
	fmt.Printf("Final capital: $%.2f\n", results.FinalCapital)

	totalTradeValue := 0.0
	buyTradeValue := 0.0
	fmt.Printf("\nTrades executed:\n")
	for _, trade := range results.Trades {
		tradeValue := trade.Quantity * trade.Price
		totalTradeValue += tradeValue
		if trade.Side == "BUY" {
			buyTradeValue += tradeValue
		}
		fmt.Printf("  %s %s: %.2f shares @ $%.2f = $%.2f\n",
			trade.Side, trade.Symbol, trade.Quantity, trade.Price, tradeValue)
	}

	fmt.Printf("\nTotal BUY trade value: $%.2f\n", buyTradeValue)
	fmt.Printf("Total trade value: $%.2f\n", totalTradeValue)

	if buyTradeValue > initialCapital {
		fmt.Printf("❌ ERROR: Total BUY trade value ($%.2f) exceeds initial capital ($%.2f)!\n", buyTradeValue, initialCapital)
	} else {
		fmt.Printf("✅ Capital allocation looks correct - no over-allocation detected.\n")
	}

	fmt.Println("\nTest completed!")
}
