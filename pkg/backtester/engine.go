package backtester

import (
	"fmt"

	"github.com/ridopark/JonBuhTrader/pkg/feed"
	"github.com/ridopark/JonBuhTrader/pkg/logging"
	"github.com/ridopark/JonBuhTrader/pkg/strategy"
	"github.com/rs/zerolog"
)

// Engine coordinates the backtest execution
type Engine struct {
	strategy  strategy.Strategy
	feed      feed.DataFeed
	broker    *Broker
	portfolio *Portfolio
	results   *Results
	ctx       *StrategyContext
	logger    zerolog.Logger
}

// NewEngine creates a new backtesting engine with default configuration
func NewEngine(s strategy.Strategy, f feed.DataFeed, initialCapital float64) *Engine {
	return NewEngineWithConfig(s, f, initialCapital, "percentage", 0.001, 0.001, 0.003)
}

// NewEngineWithConfig creates a new backtesting engine with custom commission and slippage configuration
func NewEngineWithConfig(s strategy.Strategy, f feed.DataFeed, initialCapital float64, commissionType string, commissionRate, slippage, maxSlippage float64) *Engine {
	var commissionTypeEnum CommissionType
	switch commissionType {
	case "fixed":
		commissionTypeEnum = CommissionTypeFixed
	case "percentage":
		commissionTypeEnum = CommissionTypePercentage
	default:
		commissionTypeEnum = CommissionTypePercentage
	}

	commissionConfig := NewCommissionConfig(commissionTypeEnum, commissionRate)
	portfolio := NewPortfolio(initialCapital, commissionConfig)
	broker := NewBroker(commissionConfig, slippage, maxSlippage)
	results := &Results{
		StrategyName:   s.GetName(),
		InitialCapital: initialCapital,
		Trades:         make([]strategy.TradeEvent, 0),
		EquityCurve:    make([]EquityPoint, 0),
	}

	engine := &Engine{
		strategy:  s,
		feed:      f,
		broker:    broker,
		portfolio: portfolio,
		results:   results,
		logger:    logging.GetLogger("backtester"),
	}

	// Create context after engine is initialized
	engine.ctx = NewStrategyContext(engine)

	return engine
}

// Run executes the backtest
func (e *Engine) Run() error {
	e.logger.Info().Msg("Starting backtest execution")

	// Initialize strategy
	if err := e.strategy.Initialize(e.ctx); err != nil {
		return fmt.Errorf("failed to initialize strategy: %w", err)
	}

	// Initialize data feed
	if err := e.feed.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize data feed: %w", err)
	}
	defer e.feed.Close()

	if !e.feed.HasMoreData() {
		return fmt.Errorf("no data available for the specified date range and symbols")
	}

	// Track orders from strategy
	// var pendingOrders []strategy.Order

	// Process market data
	dataPointCount := 0
	for e.feed.HasMoreData() {
		dataPoint, err := e.feed.GetNextDataPoint()
		if err != nil {
			return fmt.Errorf("error reading market data: %w", err)
		}

		if dataPoint == nil {
			e.logger.Debug().Msg("Received nil datapoint, breaking")
			break
		}

		dataPointCount++

		// Update price history for technical indicators
		e.ctx.UpdatePriceHistory(*dataPoint)

		// Get orders from strategy for this bar
		orders, err := e.strategy.OnDataPoint(e.ctx, *dataPoint)
		if err != nil {
			e.logger.Error().Err(err).Msg("Strategy error on bar")
			continue
		}

		// Execute orders through broker
		for _, order := range orders {
			bar := dataPoint.Bars[order.Symbol]
			trade, err := e.broker.ExecuteOrder(order, bar)
			if err != nil {
				e.logger.Error().Err(err).Msg("Order execution failed")
				continue
			}

			// Apply trade to portfolio
			e.portfolio.ExecuteTrade(*trade, bar.Close)

			// Notify strategy of trade
			if err := e.strategy.OnTrade(e.ctx, *trade); err != nil {
				e.logger.Error().Err(err).Msg("Strategy error on trade")
			}

			// Record trade in results
			e.results.Trades = append(e.results.Trades, *trade)
		}

		// Update portfolio value with current market prices
		e.portfolio.UpdateMarketValues(dataPoint.Bars)

		// Record equity point
		e.results.EquityCurve = append(e.results.EquityCurve, EquityPoint{
			Timestamp: dataPoint.Timestamp,
			Value:     e.portfolio.GetTotalValue(),
		})
	}

	e.logger.Info().Int("bars_processed", dataPointCount).Msg("Backtest completed")

	if dataPointCount > 0 {
		e.CloseAllPostionsAtEnd()
	}

	// Cleanup strategy
	if err := e.strategy.Cleanup(e.ctx); err != nil {
		e.logger.Error().Err(err).Msg("Strategy cleanup error")
	}
	// Finalize results
	if len(e.results.EquityCurve) > 0 {
		e.results.EndDate = e.results.EquityCurve[len(e.results.EquityCurve)-1].Timestamp
		e.results.StartDate = e.results.EquityCurve[0].Timestamp
	}

	e.results.FinalCapital = e.portfolio.GetTotalValue()
	e.results.TotalReturn = (e.results.FinalCapital - e.results.InitialCapital) / e.results.InitialCapital * 100
	e.results.TotalPL = e.results.FinalCapital - e.results.InitialCapital
	e.results.Portfolio = e.portfolio.ToStrategyPortfolio()

	// Calculate performance metrics
	e.results.CalculateMetrics()

	e.logger.Info().Msg("Backtest execution completed")
	return nil
}

func (e *Engine) CloseAllPostionsAtEnd() {
	e.logger.Info().Msg("Liquidating all positions at end of backtest")

	positions := e.portfolio.GetPositions()
	if len(positions) == 0 {
		e.logger.Info().Msg("No positions to liquidate")
		return
	}

	liquidationCount := 0
	totalLiquidationValue := 0.0

	for symbol, position := range positions {
		if position.Quantity == 0 {
			continue // Skip positions with zero quantity
		}

		// Get the last known price for this symbol from the portfolio's market values
		// We'll use the position's current market value to derive the price
		lastPrice := 0.0
		if position.Quantity != 0 {
			lastPrice = position.MarketValue / position.Quantity
		}

		if lastPrice <= 0 {
			e.logger.Error().Str("symbol", symbol).Msg("Cannot liquidate position: invalid price")
			continue
		}

		// Determine the order side based on current position
		var orderSide strategy.OrderSide
		quantity := position.Quantity
		if quantity > 0 {
			orderSide = strategy.OrderSideSell // Close long position
		} else {
			orderSide = strategy.OrderSideBuy // Close short position
			quantity = -quantity              // Make quantity positive for the order
		}

		// Create liquidation order
		liquidationOrder := strategy.Order{
			Symbol:   symbol,
			Side:     orderSide,
			Quantity: quantity,
			Type:     strategy.OrderTypeMarket,
			Reason:   "end_of_backtest_liquidation",
		}

		// Create a synthetic bar for liquidation at the last known price
		liquidationBar := strategy.BarData{
			Symbol:    symbol,
			Timestamp: e.results.EquityCurve[len(e.results.EquityCurve)-1].Timestamp,
			Open:      lastPrice,
			High:      lastPrice,
			Low:       lastPrice,
			Close:     lastPrice,
			Volume:    0, // Synthetic bar has no volume
		}

		// Execute the liquidation order
		trade, err := e.broker.ExecuteOrder(liquidationOrder, liquidationBar)
		if err != nil {
			e.logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to execute liquidation order")
			continue
		}

		// Apply trade to portfolio
		e.portfolio.ExecuteTrade(*trade, lastPrice)

		// Record the liquidation trade in results
		e.results.Trades = append(e.results.Trades, *trade)

		liquidationValue := trade.Quantity * trade.Price
		totalLiquidationValue += liquidationValue
		liquidationCount++

		e.logger.Info().
			Str("symbol", symbol).
			Str("side", string(orderSide)).
			Float64("quantity", trade.Quantity).
			Float64("price", trade.Price).
			Float64("value", liquidationValue).
			Float64("commission", trade.Commission).
			Msg("Position liquidated")
	}

	if liquidationCount > 0 {
		// Record final equity point after all liquidations
		finalTimestamp := e.results.EquityCurve[len(e.results.EquityCurve)-1].Timestamp
		e.results.EquityCurve = append(e.results.EquityCurve, EquityPoint{
			Timestamp: finalTimestamp,
			Value:     e.portfolio.GetTotalValue(),
		})

		e.logger.Info().
			Int("positions_liquidated", liquidationCount).
			Float64("total_liquidation_value", totalLiquidationValue).
			Float64("final_cash", e.portfolio.GetCash()).
			Float64("final_portfolio_value", e.portfolio.GetTotalValue()).
			Msg("All positions liquidated")
	}
}

// GetResults returns the backtest results
func (e *Engine) GetResults() *Results {
	return e.results
}
