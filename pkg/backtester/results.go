package backtester

import (
	"fmt"
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// OpenPosition represents an open position entry
type OpenPosition struct {
	Quantity   float64   // Quantity of shares/contracts
	EntryPrice float64   // Entry price per share
	EntryTime  time.Time // When the position was opened
	Commission float64   // Commission paid on entry
}

// PositionTracker tracks buy/sell pairs to calculate actual P&L using FIFO matching
type PositionTracker struct {
	Symbol       string         // Symbol being tracked
	OpenTrades   []OpenPosition // Stack of open positions (FIFO)
	TotalPL      float64        // Total P&L (realized + unrealized)
	RealizedPL   float64        // Realized P&L from closed positions
	UnrealizedPL float64        // Unrealized P&L from open positions
}

// ProcessTrade processes a trade and returns realized P&L from any closed positions
func (pt *PositionTracker) ProcessTrade(trade strategy.TradeEvent) []float64 {
	realizedPLs := make([]float64, 0)

	if trade.Side == strategy.OrderSideBuy {
		// Opening or adding to long position
		openPos := OpenPosition{
			Quantity:   trade.Quantity,
			EntryPrice: trade.Price,
			EntryTime:  trade.Timestamp,
			Commission: trade.Commission,
		}
		pt.OpenTrades = append(pt.OpenTrades, openPos)

	} else { // SELL
		// Closing long positions using FIFO
		remainingToSell := trade.Quantity
		exitPrice := trade.Price
		exitCommission := trade.Commission

		for len(pt.OpenTrades) > 0 && remainingToSell > 0 {
			openPos := &pt.OpenTrades[0]

			if openPos.Quantity <= remainingToSell {
				// Close entire open position
				quantityClosed := openPos.Quantity

				// Calculate P&L for this closed position
				grossPL := (exitPrice - openPos.EntryPrice) * quantityClosed
				totalCommission := openPos.Commission + (exitCommission * quantityClosed / trade.Quantity)
				netPL := grossPL - totalCommission

				realizedPLs = append(realizedPLs, netPL)
				pt.RealizedPL += netPL

				// Remove this position from open trades
				pt.OpenTrades = pt.OpenTrades[1:]
				remainingToSell -= quantityClosed

			} else {
				// Partially close open position
				quantityClosed := remainingToSell

				// Calculate P&L for the closed portion
				grossPL := (exitPrice - openPos.EntryPrice) * quantityClosed
				totalCommission := openPos.Commission*(quantityClosed/openPos.Quantity) +
					(exitCommission * quantityClosed / trade.Quantity)
				netPL := grossPL - totalCommission

				realizedPLs = append(realizedPLs, netPL)
				pt.RealizedPL += netPL

				// Reduce the open position quantity and commission proportionally
				openPos.Quantity -= quantityClosed
				openPos.Commission -= openPos.Commission * (quantityClosed / (openPos.Quantity + quantityClosed))
				remainingToSell = 0
			}
		}

		// If we still have quantity to sell but no open positions, it means we're going short
		// For simplicity, we'll treat short positions as negative open positions
		if remainingToSell > 0 {
			shortPos := OpenPosition{
				Quantity:   -remainingToSell, // Negative for short
				EntryPrice: exitPrice,
				EntryTime:  trade.Timestamp,
				Commission: exitCommission * remainingToSell / trade.Quantity,
			}
			pt.OpenTrades = append(pt.OpenTrades, shortPos)
		}
	}

	return realizedPLs
}

// GetCurrentPosition returns the net position (positive = long, negative = short)
func (pt *PositionTracker) GetCurrentPosition() float64 {
	totalPosition := 0.0
	for _, pos := range pt.OpenTrades {
		totalPosition += pos.Quantity
	}
	return totalPosition
}

// CalculateUnrealizedPL calculates unrealized P&L based on current market price
func (pt *PositionTracker) CalculateUnrealizedPL(currentPrice float64) float64 {
	unrealizedPL := 0.0

	for _, pos := range pt.OpenTrades {
		if pos.Quantity > 0 {
			// Long position
			unrealizedPL += (currentPrice - pos.EntryPrice) * pos.Quantity
		} else {
			// Short position
			unrealizedPL += (pos.EntryPrice - currentPrice) * (-pos.Quantity)
		}
	}

	pt.UnrealizedPL = unrealizedPL
	return unrealizedPL
}

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

	var totalPL, totalWins, totalLosses float64
	var winningTrades, losingTrades int
	var largestWin, largestLoss float64

	// Track positions per symbol to calculate actual P&L from entry/exit pairs
	positions := make(map[string]*PositionTracker)
	tradeResults := make([]float64, 0)

	// Process trades chronologically to calculate actual P&L from entry/exit pairs
	for _, trade := range r.Trades {
		symbol := trade.Symbol

		// Initialize position tracker for symbol if not exists
		if _, exists := positions[symbol]; !exists {
			positions[symbol] = &PositionTracker{
				Symbol:       symbol,
				OpenTrades:   make([]OpenPosition, 0),
				TotalPL:      0,
				RealizedPL:   0,
				UnrealizedPL: 0,
			}
		}

		pos := positions[symbol]
		realizedPLs := pos.ProcessTrade(trade)

		// Record each realized P&L from closed positions
		for _, pl := range realizedPLs {
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
	}

	// Set total trades to the number of completed round-trip trades (not individual buy/sell orders)
	r.Metrics.TotalTrades = len(tradeResults)
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
Final Cash: $%.2f
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

All Trades:
===========`,
		r.StrategyName,
		r.StartDate.Format("2006-01-02"),
		r.EndDate.Format("2006-01-02"),
		r.InitialCapital,
		r.FinalCapital,
		r.Portfolio.Cash,
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

	// Add detailed trade listing
	if len(r.Trades) > 0 {
		summary += "\n"
		summary += fmt.Sprintf("%-4s %-16s %-8s %-6s %-10s %-10s %-12s %-10s %-8s %-8s %-8s %-10s %-20s\n",
			"#", "Time", "Symbol", "Side", "Quantity", "Price", "Value", "Commission", "SecFee", "FinraTaf", "Slippage", "P&L", "Reason")
		summary += fmt.Sprintf("%-4s %-16s %-8s %-6s %-10s %-10s %-12s %-10s %-8s %-8s %-8s %-10s %-20s\n",
			"---", "----------------", "--------", "------", "----------", "----------", "------------", "----------", "--------", "--------", "--------", "----------", "--------------------")

		// Track positions to calculate P&L per trade
		positionTracker := make(map[string]*PositionTracker)

		for i, trade := range r.Trades {
			tradeValue := trade.Quantity * trade.Price
			timeStr := trade.Timestamp.Format("2006-01-02 15:04")

			// Calculate P&L for this trade
			symbol := trade.Symbol
			if _, exists := positionTracker[symbol]; !exists {
				positionTracker[symbol] = &PositionTracker{
					Symbol:     symbol,
					OpenTrades: make([]OpenPosition, 0),
				}
			}

			pos := positionTracker[symbol]
			realizedPLs := pos.ProcessTrade(trade)

			// Sum up realized P&L for this trade
			tradePL := 0.0
			for _, pl := range realizedPLs {
				tradePL += pl
			}

			// If no realized P&L, show "Open" for open positions
			plStr := "Open"
			if tradePL != 0 {
				plStr = fmt.Sprintf("%.2f", tradePL)
			}

			summary += fmt.Sprintf("%-4d %-16s %-8s %-6s %10.2f %10.2f %12.2f %10.2f %8.2f %8.2f %8.2f %10s %-20s\n",
				i+1,
				timeStr,
				trade.Symbol,
				string(trade.Side),
				trade.Quantity,
				trade.Price,
				tradeValue,
				trade.Commission,
				trade.SecFee,
				trade.FinraTaf,
				trade.Slippage,
				plStr,
				trade.Reason,
			)
		}

		// Add summary totals
		totalValue := 0.0
		totalCommission := 0.0
		totalSecFee := 0.0
		totalFinraTaf := 0.0
		totalSlippage := 0.0
		totalRealizedPL := 0.0

		// Calculate totals and track positions for P&L calculation
		totalPositionTracker := make(map[string]*PositionTracker)

		for _, trade := range r.Trades {
			totalValue += trade.Quantity * trade.Price
			totalCommission += trade.Commission
			totalSecFee += trade.SecFee
			totalFinraTaf += trade.FinraTaf
			totalSlippage += trade.Slippage

			// Calculate realized P&L for totals
			symbol := trade.Symbol
			if _, exists := totalPositionTracker[symbol]; !exists {
				totalPositionTracker[symbol] = &PositionTracker{
					Symbol:     symbol,
					OpenTrades: make([]OpenPosition, 0),
				}
			}

			pos := totalPositionTracker[symbol]
			realizedPLs := pos.ProcessTrade(trade)

			// Sum up all realized P&L
			for _, pl := range realizedPLs {
				totalRealizedPL += pl
			}
		}

		summary += fmt.Sprintf("%-4s %-16s %-8s %-6s %-10s %-10s %-12s %-10s %-8s %-8s %-8s %-10s %-20s\n",
			"---", "----------------", "--------", "------", "----------", "----------", "------------", "----------", "--------", "--------", "--------", "----------", "--------------------")
		summary += fmt.Sprintf("%-4s %-16s %-8s %-6s %-10s %-10s %12.2f %10.2f %8.2f %8.2f %8.2f %10.2f %-20s\n",
			"", "", "TOTAL", "", "", "", totalValue, totalCommission, totalSecFee, totalFinraTaf, totalSlippage, totalRealizedPL, "")
	} else {
		summary += "\nNo trades executed.\n"
	}

	return summary
}
