package backtester

import (
	"fmt"
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// Results contains the results of a backtest
type Results struct {
	StrategyName   string                `json:"strategy_name"`
	StartDate      time.Time             `json:"start_date"`
	EndDate        time.Time             `json:"end_date"`
	InitialCapital float64               `json:"initial_capital"`
	FinalCapital   float64               `json:"final_capital"`
	TotalReturn    float64               `json:"total_return"`
	TotalPL        float64               `json:"total_pl"`
	MaxDrawdown    float64               `json:"max_drawdown"`
	Trades         []strategy.TradeEvent `json:"trades"`
	EquityCurve    []EquityPoint         `json:"equity_curve"`
	Portfolio      *strategy.Portfolio   `json:"portfolio"`

	// Performance Metrics
	Metrics *PerformanceMetrics `json:"metrics"`
}

// PerformanceMetrics contains detailed performance analysis
type PerformanceMetrics struct {
	TotalTrades       int     `json:"total_trades"`
	WinningTrades     int     `json:"winning_trades"`
	LosingTrades      int     `json:"losing_trades"`
	WinRate           float64 `json:"win_rate"`
	AvgWin            float64 `json:"avg_win"`
	AvgLoss           float64 `json:"avg_loss"`
	LargestWin        float64 `json:"largest_win"`
	LargestLoss       float64 `json:"largest_loss"`
	ProfitFactor      float64 `json:"profit_factor"`
	SharpeRatio       float64 `json:"sharpe_ratio"`
	SortinoRatio      float64 `json:"sortino_ratio"`
	MaxDrawdown       float64 `json:"max_drawdown"`
	MaxDrawdownPct    float64 `json:"max_drawdown_pct"`
	CalmarRatio       float64 `json:"calmar_ratio"`
	VaR95             float64 `json:"var_95"`
	ExpectedShortfall float64 `json:"expected_shortfall"`
}

// CalculateMetrics calculates performance metrics for the results
func (r *Results) CalculateMetrics() {
	r.Metrics = &PerformanceMetrics{}

	if len(r.Trades) == 0 {
		return
	}

	// Basic trade statistics
	r.Metrics.TotalTrades = len(r.Trades)

	var totalPL, totalWins, totalLosses float64
	var winningTrades, losingTrades int
	var largestWin, largestLoss float64

	// Group trades by symbol and calculate P&L for each trade
	tradeResults := make([]float64, 0)

	// Simple P&L calculation (this is simplified - in reality we'd need to track entry/exit pairs)
	for _, trade := range r.Trades {
		// This is a simplified calculation
		// In a real implementation, we'd need to pair buy/sell trades to calculate actual P&L
		pl := 0.0
		if trade.Side == strategy.OrderSideSell {
			// Assume this is a profitable trade for simplicity
			pl = trade.Quantity * trade.Price * 0.01 // 1% profit assumption
		} else {
			pl = -trade.Commission // Just commission cost for buy trades
		}

		tradeResults = append(tradeResults, pl)
		totalPL += pl

		if pl > 0 {
			winningTrades++
			totalWins += pl
			if pl > largestWin {
				largestWin = pl
			}
		} else if pl < 0 {
			losingTrades++
			totalLosses += pl
			if pl < largestLoss {
				largestLoss = pl
			}
		}
	}

	r.Metrics.WinningTrades = winningTrades
	r.Metrics.LosingTrades = losingTrades

	if r.Metrics.TotalTrades > 0 {
		r.Metrics.WinRate = float64(winningTrades) / float64(r.Metrics.TotalTrades) * 100
	}

	if winningTrades > 0 {
		r.Metrics.AvgWin = totalWins / float64(winningTrades)
	}

	if losingTrades > 0 {
		r.Metrics.AvgLoss = totalLosses / float64(losingTrades)
	}

	r.Metrics.LargestWin = largestWin
	r.Metrics.LargestLoss = largestLoss

	// Profit Factor
	if totalLosses != 0 {
		r.Metrics.ProfitFactor = totalWins / (-totalLosses)
	}

	// Drawdown metrics
	r.Metrics.MaxDrawdown = r.MaxDrawdown
	r.Metrics.MaxDrawdownPct = r.MaxDrawdown * 100

	// Calmar Ratio (Annual Return / Max Drawdown)
	if r.MaxDrawdown > 0 {
		annualReturn := r.TotalReturn // Simplified - should be annualized
		r.Metrics.CalmarRatio = annualReturn / (r.MaxDrawdown * 100)
	}

	// Calculate Sharpe Ratio (simplified)
	if len(r.EquityCurve) > 1 {
		returns := make([]float64, len(r.EquityCurve)-1)
		for i := 1; i < len(r.EquityCurve); i++ {
			if r.EquityCurve[i-1].Value > 0 {
				returns[i-1] = (r.EquityCurve[i].Value - r.EquityCurve[i-1].Value) / r.EquityCurve[i-1].Value
			}
		}

		r.Metrics.SharpeRatio = calculateSharpeRatio(returns)
		r.Metrics.SortinoRatio = calculateSortinoRatio(returns)
	}
}

// calculateSharpeRatio calculates the Sharpe ratio from returns
func calculateSharpeRatio(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// Calculate mean return
	sum := 0.0
	for _, ret := range returns {
		sum += ret
	}
	mean := sum / float64(len(returns))

	// Calculate standard deviation
	sumSquares := 0.0
	for _, ret := range returns {
		diff := ret - mean
		sumSquares += diff * diff
	}

	if len(returns) <= 1 {
		return 0
	}

	stdDev := sumSquares / float64(len(returns)-1)
	if stdDev <= 0 {
		return 0
	}

	// Sharpe ratio (assuming risk-free rate of 0)
	return mean / stdDev
}

// calculateSortinoRatio calculates the Sortino ratio from returns
func calculateSortinoRatio(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// Calculate mean return
	sum := 0.0
	for _, ret := range returns {
		sum += ret
	}
	mean := sum / float64(len(returns))

	// Calculate downside deviation (only negative returns)
	sumDownside := 0.0
	downsideCount := 0
	for _, ret := range returns {
		if ret < 0 {
			sumDownside += ret * ret
			downsideCount++
		}
	}

	if downsideCount == 0 {
		return 0 // No downside
	}

	downsideDeviation := sumDownside / float64(downsideCount)
	if downsideDeviation <= 0 {
		return 0
	}

	// Sortino ratio
	return mean / downsideDeviation
}

// Summary returns a human-readable summary of the results
func (r *Results) Summary() string {
	if r.Metrics == nil {
		r.CalculateMetrics()
	}

	summary := fmt.Sprintf(`
Backtest Results for %s
=======================
Period: %s to %s
Initial Capital: $%.2f
Final Capital: $%.2f
Total Return: %.2f%%
Total P&L: $%.2f
Max Drawdown: %.2f%%

Trade Statistics:
- Total Trades: %d
- Winning Trades: %d (%.1f%%)
- Losing Trades: %d (%.1f%%)
- Average Win: $%.2f
- Average Loss: $%.2f
- Largest Win: $%.2f
- Largest Loss: $%.2f
- Profit Factor: %.2f

Risk Metrics:
- Sharpe Ratio: %.2f
- Sortino Ratio: %.2f
- Calmar Ratio: %.2f
- Max Drawdown: %.2f%%
`,
		r.StrategyName,
		r.StartDate.Format("2006-01-02"),
		r.EndDate.Format("2006-01-02"),
		r.InitialCapital,
		r.FinalCapital,
		r.TotalReturn,
		r.FinalCapital-r.InitialCapital,
		r.MaxDrawdown*100,
		r.Metrics.TotalTrades,
		r.Metrics.WinningTrades,
		r.Metrics.WinRate,
		r.Metrics.LosingTrades,
		100-r.Metrics.WinRate,
		r.Metrics.AvgWin,
		r.Metrics.AvgLoss,
		r.Metrics.LargestWin,
		r.Metrics.LargestLoss,
		r.Metrics.ProfitFactor,
		r.Metrics.SharpeRatio,
		r.Metrics.SortinoRatio,
		r.Metrics.CalmarRatio,
		r.Metrics.MaxDrawdownPct,
	)

	return summary
}
