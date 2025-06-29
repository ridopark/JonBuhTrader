package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/ridopark/JonBuhTrader/internal/data"
	"github.com/ridopark/JonBuhTrader/pkg/backtester"
	"github.com/ridopark/JonBuhTrader/pkg/feed"
	"github.com/ridopark/JonBuhTrader/pkg/logging"
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
	"github.com/ridopark/JonBuhTrader/pkg/strategy/examples"
)

func main() {
	// Load environment variables from .env file
	envErr := godotenv.Load()

	// Command line flags
	var (
		symbolsFlag    = flag.String("symbols", "AAPL", "Symbols to backtest (comma-separated, e.g., AAPL,TSLA)")
		strategyFlag   = flag.String("strategy", "buy_and_hold", "Strategy to use")
		startDate      = flag.String("start", "2024-01-01", "Start date (YYYY-MM-DD)")
		endDate        = flag.String("end", "2024-12-31", "End date (YYYY-MM-DD)")
		initialCapital = flag.Float64("capital", 10000.0, "Initial capital")
		timeframe      = flag.String("timeframe", "1m", "Timeframe (1m, 5m, 15m, 1h, 1d)")
	)
	flag.Parse()

	// Get logging configuration from environment variables
	logLevel := getEnv("LOG_LEVEL", "info")
	logPretty := getEnvBool("LOG_PRETTY", true)
	logToFile := getEnvBool("LOG_TO_FILE", true)
	logDir := getEnv("LOG_DIR", "logs")
	logFileName := getEnv("LOG_FILE", "backtester.log")

	// Initialize logging
	logConfig := logging.DefaultConfig()
	logConfig.Level = logging.LogLevel(logLevel)
	logConfig.Pretty = logPretty
	logConfig.EnableFile = logToFile
	logConfig.LogDir = logDir
	logConfig.LogFileName = logFileName
	logging.Initialize(logConfig)

	logger := logging.GetLogger("main")

	// Log environment loading status
	if envErr != nil {
		logger.Warn().Err(envErr).Msg("Could not load .env file, using system environment variables")
	} else {
		logger.Debug().Msg("Successfully loaded .env file")
	}

	logger.Info().Msg("JonBuhTrader Backtester")
	logger.Info().Msg("=======================")

	// Parse dates
	start, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		logger.Fatal().Err(err).Str("start_date", *startDate).Msg("Invalid start date")
	}

	// For end date, add 24 hours to include the entire day
	end, err := time.Parse("2006-01-02", *endDate)
	if err != nil {
		logger.Fatal().Err(err).Str("end_date", *endDate).Msg("Invalid end date")
	}
	end = end.Add(24 * time.Hour) // Add one day to include all data for the end date

	// Parse symbols from comma-delimited string
	symbolsInput := strings.TrimSpace(*symbolsFlag)
	symbols := strings.Split(symbolsInput, ",")
	for i, symbol := range symbols {
		symbols[i] = strings.TrimSpace(symbol)
	}

	logger.Debug().Strs("symbols", symbols).Msg("Parsed symbols from input")

	// Get database configuration from environment variables
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5432")
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "trading_password_2025")
	dbName := getEnv("POSTGRES_DB", "trading_data")

	logger.Debug().
		Str("db_host", dbHost).
		Str("db_port", dbPort).
		Str("db_user", dbUser).
		Str("db_name", dbName).
		Msg("Database configuration loaded from environment")

	// Create database connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Create data provider
	logger.Info().Msg("Connecting to database...")
	provider, err := data.NewTimescaleDBProvider(connStr)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create data provider")
	}
	defer provider.Close()

	// Create data feed
	dataFeed := feed.NewHistoricalFeed(provider, symbols, *timeframe, start, end)

	// Create strategy
	var strategyInstance strategy.Strategy

	// We can override this based on the flag if we had more strategies
	switch *strategyFlag {
	case "buy_and_hold":
		strategyInstance = examples.NewBuyAndHoldStrategy(symbols, *initialCapital)
	case "ma_crossover":
		ma := examples.NewMovingAverageCrossoverStrategy(5, 20) // 5-period and 20-period MA
		ma.SetSymbols(symbols)
		strategyInstance = ma
	case "multi_indicator":
		multi := examples.NewMultiIndicatorStrategy()
		multi.SetSymbols(symbols)
		strategyInstance = multi
	default:
		logger.Fatal().Str("strategy", *strategyFlag).Msg("Unknown strategy. Available strategies: buy_and_hold, ma_crossover, multi_indicator")
	}

	// Get trading configuration from environment variables
	commissionType := getEnv("COMMISSION_TYPE", "percentage")
	commissionRate := getEnvFloat("COMMISSION_RATE", 0.001)
	slippageRate := getEnvFloat("SLIPPAGE_RATE", 0.001)
	maxSlippage := getEnvFloat("MAX_SLIPPAGE", 0.003)

	// Create and run backtester
	logger.Info().
		Strs("symbols", symbols).
		Str("start_date", *startDate).
		Str("end_date", *endDate).
		Str("strategy", *strategyFlag).
		Float64("initial_capital", *initialCapital).
		Str("commission_type", commissionType).
		Float64("commission_rate", commissionRate).
		Float64("slippage_rate", slippageRate).
		Float64("max_slippage", maxSlippage).
		Msg("Running backtest")

	engine := backtester.NewEngineWithConfig(strategyInstance, dataFeed, *initialCapital, commissionType, commissionRate, slippageRate, maxSlippage)

	err = engine.Run()
	if err != nil {
		logger.Fatal().Err(err).Msg("Backtest failed")
	}

	// Get results
	results := engine.GetResults()

	// Calculate detailed metrics
	results.CalculateMetrics()

	// Print results
	logger.Info().Msg("\n" + results.Summary())

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

// Helper function to get boolean environment variable with default
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// Helper function to get float environment variable with default
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}
