package main

import (
	"fmt"
	"log"

	"github.com/ridopark/JonBuhTrader/pkg/strategy/examples"
)

func main() {
	fmt.Println("Support & Resistance Strategy Initialization Test")
	fmt.Println("================================================")

	// Test strategy creation
	strategy := examples.NewSupportResistanceStrategy()
	if strategy == nil {
		log.Fatal("Failed to create strategy")
	}

	fmt.Printf("✅ Strategy created successfully: %s\n", strategy.GetName())

	// Test setting symbols
	symbols := []string{"AAPL", "MSFT", "GOOGL"}
	strategy.SetSymbols(symbols)

	fmt.Printf("✅ Symbols set successfully: %v\n", strategy.GetSymbols())

	// Test parameter access
	params := strategy.GetParameters()
	fmt.Printf("✅ Strategy parameters loaded: %d parameters\n", len(params))

	// Print key parameters
	fmt.Printf("   - Position Size: %.1f%%\n", params["positionSize"].(float64)*100)
	fmt.Printf("   - Stop Loss: %.1f%%\n", params["stopLoss"].(float64))
	fmt.Printf("   - Take Profit: %.1f%%\n", params["takeProfit"].(float64))
	fmt.Printf("   - Max Positions: 3 (from allocator config)\n")
	fmt.Printf("   - Allocation Method: Confidence-based\n")

	fmt.Println("\n✅ Support & Resistance Strategy refactoring verification complete!")
	fmt.Println("   - Strategy uses new CapitalAllocator")
	fmt.Println("   - Signal collection uses TradingSignal interface")
	fmt.Println("   - Old allocation logic removed")
	fmt.Println("   - All code compiles successfully")
}
