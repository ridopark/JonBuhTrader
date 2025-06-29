package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
	"github.com/ridopark/JonBuhTrader/pkg/strategy/examples"
)

// mockContext implements strategy.Context for testing
type mockContext struct {
	cash      float64
	positions map[string]*strategy.Position
	logs      []map[string]interface{}
}

func (m *mockContext) GetPortfolio() *strategy.Portfolio {
	totalValue := m.cash
	for _, pos := range m.positions {
		if pos != nil {
			totalValue += pos.MarketValue
		}
	}
	return &strategy.Portfolio{
		Cash:       m.cash,
		TotalValue: totalValue,
		Positions:  m.positions,
	}
}

func (m *mockContext) GetCash() float64 {
	return m.cash
}

func (m *mockContext) GetPosition(symbol string) *strategy.Position {
	return m.positions[symbol]
}

func (m *mockContext) SMA(symbol string, period int) (float64, error) {
	return 0.0, nil // Mock implementation
}

func (m *mockContext) EMA(symbol string, period int) (float64, error) {
	return 0.0, nil // Mock implementation
}

func (m *mockContext) RSI(symbol string, period int) (float64, error) {
	return 50.0, nil // Mock implementation
}

func (m *mockContext) MACD(symbol string, fastPeriod, slowPeriod, signalPeriod int) (float64, float64, float64, error) {
	return 0.0, 0.0, 0.0, nil // Mock implementation
}

func (m *mockContext) ADX(symbol string, period int) (float64, error) {
	return 25.0, nil // Mock implementation
}

func (m *mockContext) SuperTrend(symbol string, period int, multiplier float64) (float64, error) {
	return 0.0, nil // Mock implementation
}

func (m *mockContext) ParbolicSAR(symbol string, step, max float64) (float64, error) {
	return 0.0, nil // Mock implementation
}

func (m *mockContext) Log(level string, message string, data map[string]interface{}) {
	logEntry := map[string]interface{}{
		"level":   level,
		"message": message,
		"time":    time.Now().Format("15:04:05"),
	}
	for k, v := range data {
		logEntry[k] = v
	}
	m.logs = append(m.logs, logEntry)

	// Print important logs
	if level == "info" && (message == "Enhanced support_bounce BUY signal" || message == "Enhanced resistance_breakout BUY signal") {
		fmt.Printf("[%s] %s: %s (symbol: %v, confidence: %.2f, quantity: %.0f, orderValue: %.2f)\n",
			level, logEntry["time"], message, data["symbol"], data["confidence"], data["quantity"], data["orderValue"])
	}
}

func main() {
	fmt.Println("Testing Support & Resistance Strategy with Capital Allocation")
	fmt.Println("===========================================================")

	// Create strategy
	strategy := examples.NewSupportResistanceStrategy()

	// Set multiple symbols to test allocation
	symbols := []string{"AAPL", "MSFT", "GOOGL"}
	strategy.SetSymbols(symbols)

	// Create mock context with initial cash
	ctx := &mockContext{
		cash:      10000.0, // $10,000 starting cash
		positions: make(map[string]*strategy.Position),
	}

	// Initialize strategy
	err := strategy.Initialize(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize strategy: %v", err)
	}

	// Create some test data that should generate support/resistance signals
	// We'll simulate multiple bars to build up support/resistance levels
	fmt.Println("\nBuilding support/resistance levels...")

	// Build some price history first (no signals expected)
	for i := 0; i < 25; i++ {
		dataPoint := strategy.DataPoint{
			Bars: map[string]strategy.BarData{
				"AAPL": {
					Open:   150.0 + float64(i%3),
					High:   152.0 + float64(i%3),
					Low:    149.0 + float64(i%3),
					Close:  151.0 + float64(i%3),
					Volume: 1000000,
				},
				"MSFT": {
					Open:   300.0 + float64(i%4),
					High:   302.0 + float64(i%4),
					Low:    298.0 + float64(i%4),
					Close:  301.0 + float64(i%4),
					Volume: 800000,
				},
				"GOOGL": {
					Open:   2500.0 + float64(i%5)*2,
					High:   2520.0 + float64(i%5)*2,
					Low:    2480.0 + float64(i%5)*2,
					Close:  2510.0 + float64(i%5)*2,
					Volume: 500000,
				},
			},
		}

		orders, err := strategy.OnDataPoint(ctx, dataPoint)
		if err != nil {
			log.Fatalf("Strategy failed: %v", err)
		}

		if len(orders) > 0 {
			fmt.Printf("Bar %d: Generated %d orders\n", i+1, len(orders))
		}
	}

	fmt.Printf("\nBuilt price history. Current cash: $%.2f\n", ctx.cash)

	// Now simulate some conditions that might generate multiple simultaneous signals
	// This simulates a scenario where all three stocks hit support levels
	fmt.Println("\nSimulating multiple simultaneous support bounce signals...")

	testDataPoint := strategy.DataPoint{
		Bars: map[string]strategy.BarData{
			"AAPL": {
				Open:   149.5,
				High:   150.8,
				Low:    149.0,   // Near previous support
				Close:  150.2,   // Bouncing off support
				Volume: 1500000, // High volume
			},
			"MSFT": {
				Open:   298.5,
				High:   301.2,
				Low:    298.0,   // Near previous support
				Close:  300.8,   // Bouncing off support
				Volume: 1200000, // High volume
			},
			"GOOGL": {
				Open:   2480.0,
				High:   2515.0,
				Low:    2480.0, // Near previous support
				Close:  2512.0, // Bouncing off support
				Volume: 750000, // High volume
			},
		},
	}

	orders, err := strategy.OnDataPoint(ctx, testDataPoint)
	if err != nil {
		log.Fatalf("Strategy failed: %v", err)
	}

	fmt.Printf("\nGenerated %d orders from potential signals\n", len(orders))

	// Calculate total order value
	totalOrderValue := 0.0
	for i, order := range orders {
		orderValue := order.Quantity * testDataPoint.Bars[order.Symbol].Close
		totalOrderValue += orderValue
		fmt.Printf("Order %d: %s %.0f shares of %s = $%.2f\n",
			i+1, order.Side, order.Quantity, order.Symbol, orderValue)

		// Simulate order execution by updating cash and positions
		ctx.cash -= orderValue
		ctx.positions[order.Symbol] = &strategy.Position{
			Symbol:   order.Symbol,
			Quantity: order.Quantity,
			AvgPrice: testDataPoint.Bars[order.Symbol].Close,
		}
	}

	fmt.Printf("\nTotal order value: $%.2f\n", totalOrderValue)
	fmt.Printf("Remaining cash: $%.2f\n", ctx.cash)
	fmt.Printf("Cash utilization: %.1f%%\n", (totalOrderValue/10000.0)*100)

	// Verify no over-allocation
	if totalOrderValue > 10000.0 {
		fmt.Printf("❌ OVER-ALLOCATION DETECTED! Total orders ($%.2f) exceed available cash ($10,000)\n", totalOrderValue)
	} else {
		fmt.Printf("✅ No over-allocation. Capital allocation working correctly.\n")
	}

	// Show final positions
	fmt.Println("\nFinal Positions:")
	for symbol, position := range ctx.positions {
		if position != nil && position.Quantity > 0 {
			currentValue := position.Quantity * position.AvgPrice
			fmt.Printf("  %s: %.0f shares @ $%.2f = $%.2f\n",
				symbol, position.Quantity, position.AvgPrice, currentValue)
		}
	}

	// Test another round to see if remaining cash is properly managed
	fmt.Println("\nTesting with another signal after positions are taken...")

	// Try another signal to see how remaining cash is handled
	testDataPoint2 := strategy.DataPoint{
		Bars: map[string]strategy.BarData{
			"TSLA": {
				Open:   800.0,
				High:   805.0,
				Low:    798.0,
				Close:  803.0,
				Volume: 1000000,
			},
		},
	}

	// Add TSLA to symbols for this test
	strategy.SetSymbols(append(symbols, "TSLA"))

	orders2, err := strategy.OnDataPoint(ctx, testDataPoint2)
	if err != nil {
		log.Fatalf("Strategy failed on second test: %v", err)
	}

	fmt.Printf("Second round generated %d orders\n", len(orders2))
	if len(orders2) > 0 {
		for _, order := range orders2 {
			orderValue := order.Quantity * testDataPoint2.Bars[order.Symbol].Close
			fmt.Printf("Would create order: %s %.0f shares of %s = $%.2f\n",
				order.Side, order.Quantity, order.Symbol, orderValue)
		}
	}

	fmt.Println("\n✅ Support & Resistance Strategy allocation test completed successfully!")
}
