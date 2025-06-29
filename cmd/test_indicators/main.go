package main

import (
	"fmt"
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/backtester"
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// TestIndicators creates a simple test to verify our indicator implementations
func main() {
	fmt.Println("Testing Technical Indicators...")

	// Create a mock engine (minimal setup)
	engine := &backtester.Engine{}
	ctx := backtester.NewStrategyContext(engine)

	// Test data - simple price series with highs and lows
	testSymbol := "TEST"
	testData := []struct {
		high, low, close float64
	}{
		{44.5, 43.5, 44.0},
		{44.75, 44.0, 44.25},
		{45.0, 44.25, 44.5},
		{44.5, 43.5, 43.75},
		{45.0, 44.0, 44.5},
		{45.25, 44.5, 44.75},
		{47.5, 46.5, 47.0},
		{47.75, 47.0, 47.25},
		{48.0, 47.25, 47.5},
		{48.25, 47.5, 47.75},
		{47.75, 47.0, 47.25},
		{48.25, 47.5, 47.75},
		{47.25, 46.5, 46.75},
		{46.75, 46.0, 46.25},
		{46.75, 46.0, 46.25},
	}

	// Simulate feeding data to the context
	baseTime := time.Now()
	for i, data := range testData {
		dataPoint := strategy.DataPoint{
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
			Bars: map[string]strategy.BarData{
				testSymbol: {
					Symbol:    testSymbol,
					Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
					Open:      data.close,
					High:      data.high,
					Low:       data.low,
					Close:     data.close,
					Volume:    1000,
					Timeframe: "1h",
				},
			},
		}
		ctx.UpdatePriceHistory(dataPoint)
	}

	testPrices := make([]float64, len(testData))
	for i, data := range testData {
		testPrices[i] = data.close
	}

	fmt.Printf("Test data: %v\n\n", testPrices)

	// Test SMA
	fmt.Println("=== SMA Tests ===")
	for _, period := range []int{5, 10, 14} {
		sma, err := ctx.SMA(testSymbol, period)
		if err != nil {
			fmt.Printf("SMA(%d): Error - %v\n", period, err)
		} else {
			fmt.Printf("SMA(%d): %.4f\n", period, sma)
		}
	}

	// Test EMA
	fmt.Println("\n=== EMA Tests ===")
	for _, period := range []int{5, 10, 14} {
		ema, err := ctx.EMA(testSymbol, period)
		if err != nil {
			fmt.Printf("EMA(%d): Error - %v\n", period, err)
		} else {
			fmt.Printf("EMA(%d): %.4f\n", period, ema)
		}
	}

	// Test RSI
	fmt.Println("\n=== RSI Tests ===")
	for _, period := range []int{14} {
		rsi, err := ctx.RSI(testSymbol, period)
		if err != nil {
			fmt.Printf("RSI(%d): Error - %v\n", period, err)
		} else {
			fmt.Printf("RSI(%d): %.4f\n", period, rsi)
		}
	}

	// Test MACD
	fmt.Println("\n=== MACD Tests ===")
	macd, signal, histogram, err := ctx.MACD(testSymbol, 12, 26, 9)
	if err != nil {
		fmt.Printf("MACD: Error - %v\n", err)
	} else {
		fmt.Printf("MACD: %.4f, Signal: %.4f, Histogram: %.4f\n", macd, signal, histogram)
	}

	// Test ADX
	fmt.Println("\n=== ADX Tests ===")
	for _, period := range []int{14} {
		adx, err := ctx.ADX(testSymbol, period)
		if err != nil {
			fmt.Printf("ADX(%d): Error - %v\n", period, err)
		} else {
			fmt.Printf("ADX(%d): %.4f\n", period, adx)
		}
	}

	// Test SuperTrend
	fmt.Println("\n=== SuperTrend Tests ===")
	superTrend, err := ctx.SuperTrend(testSymbol, 10, 3.0)
	if err != nil {
		fmt.Printf("SuperTrend: Error - %v\n", err)
	} else {
		fmt.Printf("SuperTrend(10, 3.0): %.4f\n", superTrend)
	}

	// Test Parabolic SAR
	fmt.Println("\n=== Parabolic SAR Tests ===")
	sar, err := ctx.ParbolicSAR(testSymbol, 0.02, 0.2)
	if err != nil {
		fmt.Printf("Parabolic SAR: Error - %v\n", err)
	} else {
		fmt.Printf("Parabolic SAR(0.02, 0.2): %.4f\n", sar)
	}

	// Test with insufficient data
	fmt.Println("\n=== Error Handling Tests ===")
	_, err1 := ctx.SMA("NONEXISTENT", 10)
	if err1 != nil {
		fmt.Printf("Expected error for non-existent symbol: %v\n", err1)
	}

	_, err2 := ctx.SMA(testSymbol, 100)
	if err2 != nil {
		fmt.Printf("Expected error for insufficient data: %v\n", err2)
	}

	fmt.Println("\nIndicator tests completed!")
}
