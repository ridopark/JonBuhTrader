# Support & Resistance Strategy

## Overview
The Support & Resistance strategy identifies key price levels where the stock has historically bounced (support) or been rejected (resistance). It then trades on:
1. **Bounces off support levels** (buy signals)
2. **Breakouts above resistance levels** (buy signals)

## Key Features

### Level Identification
- **Pivot Detection**: Identifies pivot highs and lows using a 3-bar lookback method
- **Level Consolidation**: Groups similar price levels within tolerance
- **Strength Calculation**: Tracks how many times each level has been tested
- **Dynamic Classification**: Automatically classifies levels as support or resistance based on current price

### Trading Logic
- **Support Bounce**: Buy when price bounces off a strong support level
- **Resistance Breakout**: Buy when price breaks above a strong resistance level with volume confirmation
- **Risk Management**: Automatic stop-loss and take-profit orders
- **Volume Confirmation**: Optional volume filter for breakout confirmation

### Configuration (via .env file)

```bash
# Support & Resistance Strategy Configuration
SR_LOOKBACK_PERIOD=20       # Number of bars to look back for pivot identification
SR_MIN_TOUCHES=2            # Minimum touches for a level to be considered valid
SR_LEVEL_TOLERANCE=0.5      # Tolerance percentage for level identification (0.5%)
SR_BREAKOUT_CONFIRMATION=2  # Number of bars to confirm breakout
SR_POSITION_SIZE=0.95       # Position size as percentage of available cash
SR_STOP_LOSS=2.0           # Stop loss percentage (2.0%)
SR_TAKE_PROFIT=4.0         # Take profit percentage (4.0%)
SR_MIN_LEVEL_STRENGTH=3     # Minimum strength score for a level to be traded
SR_USE_VOLUME_FILTER=true   # Use volume confirmation for breakouts
SR_VOLUME_MULTIPLIER=1.5    # Volume must be X times average for breakout confirmation
```

## Usage

```bash
# Run Support & Resistance strategy
./backtester -strategy support_resistance -symbols AAPL,TSLA -start 2025-01-01 -end 2025-01-31

# Test with different symbols
./backtester -strategy support_resistance -symbols MSFT,GOOGL -start 2025-01-01 -end 2025-01-31
```

## Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `SR_LOOKBACK_PERIOD` | 20 | Bars to analyze for pivot identification |
| `SR_MIN_TOUCHES` | 2 | Minimum times a level must be tested |
| `SR_LEVEL_TOLERANCE` | 0.5% | Price tolerance for level grouping |
| `SR_BREAKOUT_CONFIRMATION` | 2 | Bars to confirm breakout |
| `SR_POSITION_SIZE` | 0.95 | Position size (95% of available cash) |
| `SR_STOP_LOSS` | 2.0% | Stop loss threshold |
| `SR_TAKE_PROFIT` | 4.0% | Take profit threshold |
| `SR_MIN_LEVEL_STRENGTH` | 3 | Minimum strength for trading |
| `SR_USE_VOLUME_FILTER` | true | Require volume confirmation |
| `SR_VOLUME_MULTIPLIER` | 1.5x | Volume multiplier for confirmation |

## Strategy Logic

### Entry Signals
1. **Support Bounce**:
   - Price is within tolerance of a support level
   - Support level has minimum required strength
   - Optional volume confirmation

2. **Resistance Breakout**:
   - Price breaks above resistance level + tolerance
   - Resistance level has minimum required strength
   - Optional volume confirmation (1.5x average volume)

### Exit Signals
1. **Stop Loss**: Position closed if loss exceeds 2% (configurable)
2. **Take Profit**: Position closed if profit exceeds 4% (configurable)
3. **End of Backtest**: All positions liquidated

### Risk Management
- **Whole Share Trading**: All quantities rounded down to whole numbers
- **Position Sizing**: Configurable percentage of available cash
- **Stop Loss/Take Profit**: Automatic risk management
- **Volume Filter**: Optional confirmation to reduce false signals

## Data Requirements
- Requires sufficient historical data for level identification (typically 20+ bars)
- Works with any timeframe (1m, 5m, 15m, 1h, 1d)
- Better performance with longer timeframes for more robust levels

## Notes
- Strategy may not generate trades in short backtests due to data requirements
- Levels are recalculated on each bar to adapt to market conditions
- Volume filter helps reduce false breakout signals
- Best suited for trending or range-bound markets with clear support/resistance levels
