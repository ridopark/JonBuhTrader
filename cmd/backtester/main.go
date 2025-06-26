package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ridopark/JonBuhTrader/internal/data"
	"github.com/ridopark/JonBuhTrader/pkg/backtester"
	"github.com/ridopark/JonBuhTrader/pkg/feed"
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
	"github.com/ridopark/JonBuhTrader/pkg/strategy/examples"
)

func main() {
	// Command line flags
	var (
		symbol         = flag.String("symbol", "AAPL", "Symbol to backtest")
		strategyFlag   = flag.String("strategy", "buy_and_hold", "Strategy to use")
		startDate      = flag.String("start", "2024-01-01", "Start date (YYYY-MM-DD)")
		endDate        = flag.String("end", "2024-12-31", "End date (YYYY-MM-DD)")
		initialCapital = flag.Float64("capital", 10000.0, "Initial capital")
		timeframe      = flag.String("timeframe", "1m", "Timeframe (1m, 5m, 15m, 1h, 1d)")
		dbHost         = flag.String("db-host", "localhost", "Database host")
		dbPort         = flag.String("db-port", "5432", "Database port")
		dbUser         = flag.String("db-user", "postgres", "Database user")
		dbPassword     = flag.String("db-password", "trading_password_2025", "Database password")
		dbName         = flag.String("db-name", "trading_data", "Database name")
	)
	flag.Parse()

	fmt.Println("JonBuhTrader Backtester")
	fmt.Println("=======================")

	// Parse dates
	start, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		log.Fatalf("Invalid start date: %v", err)
	}

	// For end date, add 24 hours to include the entire day
	end, err := time.Parse("2006-01-02", *endDate)
	if err != nil {
		log.Fatalf("Invalid end date: %v", err)
	}
	end = end.Add(24 * time.Hour) // Add one day to include all data for the end date

	// Create database connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		*dbHost, *dbPort, *dbUser, *dbPassword, *dbName)

	// Create data provider
	fmt.Println("Connecting to database...")
	provider, err := data.NewTimescaleDBProvider(connStr)
	if err != nil {
		log.Fatalf("Failed to create data provider: %v", err)
	}
	defer provider.Close()

	// Create data feed
	symbols := []string{*symbol}
	dataFeed := feed.NewHistoricalFeed(provider, symbols, *timeframe, start, end)

	// Create strategy
	var strategyInstance strategy.Strategy

	// We can override this based on the flag if we had more strategies
	switch *strategyFlag {
	case "buy_and_hold":
		strategyInstance = examples.NewBuyAndHoldStrategy()
	case "ma_crossover":
		strategyInstance = examples.NewMovingAverageCrossoverStrategy(5, 20) // 5-period and 20-period MA
	default:
		log.Fatalf("Unknown strategy: %s. Available strategies: buy_and_hold, ma_crossover", *strategyFlag)
	}

	// Create and run backtester
	fmt.Printf("Running backtest for %s from %s to %s...\n", *symbol, *startDate, *endDate)
	engine := backtester.NewEngine(strategyInstance, dataFeed, *initialCapital)

	err = engine.Run()
	if err != nil {
		log.Fatalf("Backtest failed: %v", err)
	}

	// Get results
	results := engine.GetResults()

	// Calculate detailed metrics
	results.CalculateMetrics()

	// Print results
	fmt.Println("\n" + results.Summary())

	// Optionally save results to file
	// TODO: Add JSON export functionality
}

// Helper function to get environment variable with default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
