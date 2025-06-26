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

// NewEngine creates a new backtesting engine
func NewEngine(s strategy.Strategy, f feed.DataFeed, initialCapital float64) *Engine {
	commission := 0.001 // 0.1% commission
	slippage := 0.001   // 0.1% slippage

	portfolio := NewPortfolio(initialCapital, commission)
	broker := NewBroker(commission, slippage)
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
	barCount := 0
	for e.feed.HasMoreData() {
		bar, err := e.feed.GetNextBar()
		if err != nil {
			return fmt.Errorf("error reading market data: %w", err)
		}

		if bar == nil {
			e.logger.Debug().Msg("Received nil bar, breaking")
			break
		}

		barCount++

		// Get orders from strategy for this bar
		orders, err := e.strategy.OnBar(e.ctx, *bar)
		if err != nil {
			e.logger.Error().Err(err).Msg("Strategy error on bar")
			continue
		}

		// Execute orders through broker
		for _, order := range orders {
			trade, err := e.broker.ExecuteOrder(order, *bar)
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
		e.portfolio.UpdateMarketValues(map[string]float64{
			bar.Symbol: bar.Close,
		})

		// Record equity point
		e.results.EquityCurve = append(e.results.EquityCurve, EquityPoint{
			Timestamp: bar.Timestamp,
			Value:     e.portfolio.GetTotalValue(),
		})
	}

	e.logger.Info().Int("bars_processed", barCount).Msg("Backtest completed")

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

// GetResults returns the backtest results
func (e *Engine) GetResults() *Results {
	return e.results
}
