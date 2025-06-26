# JonBuhTrader

A comprehensive algorithmic trading system built in Go, featuring backtesting capabilities, real-time trading, and PostgreSQL TimescaleDB for efficient time series data storage.

## 🏗️ Project Structure

```
JonBuhTrader/
├── cmd/                     # Command-line applications
│   ├── trader/              # Main trading application
│   └── backtester/          # Backtesting engine CLI
├── pkg/                     # Public packages (reusable components)
│   ├── backtester/          # Core backtesting engine
│   ├── strategy/            # Trading strategy framework
│   ├── feed/                # Market data feeds
│   ├── broker/              # Broker integrations (Alpaca, etc.)
│   └── reporting/           # Performance analysis and reporting
├── internal/                # Private packages (internal use only)
│   ├── data/                # Data management and storage
│   ├── config/              # Configuration management
│   └── auth/                # Authentication and API key management
├── configs/                 # Configuration files
│   ├── trading/             # Live trading configurations
│   └── backtester/          # Backtesting configurations
├── docker/                  # Docker setup for PostgreSQL TimescaleDB
├── docs/                    # Documentation
├── deployments/             # Deployment configurations
└── .env                     # Environment variables
```

## 🎯 Backtester Architecture

### Core Components

#### **Main Backtester Engine** (`pkg/backtester/`)
- `engine.go` - Main backtesting engine with event loop
- `portfolio.go` - Portfolio management (positions, cash, P&L tracking)
- `broker.go` - Simulated broker for order execution
- `performance.go` - Performance metrics and analytics
- `events.go` - Event system (market data, orders, fills)

#### **Strategy Interface** (`pkg/strategy/`)
- `interface.go` - Generic strategy interface that both backtester and live trading use
- `base.go` - Base strategy implementation with common functionality
- `examples/` - Example strategies (moving average, mean reversion, etc.)

### Data Management Integration

#### **Historical Data Service** (`internal/data/`)
- `historical.go` - Service to fetch OHLCV data from TimescaleDB
- `provider.go` - Interface for different data providers (database, files, APIs)
- `cache.go` - In-memory caching for frequently accessed data

#### **Market Data Feed** (`pkg/feed/`)
- `historical_feed.go` - Replays historical data as market events
- `live_feed.go` - Real-time data feed (for live trading)
- `interface.go` - Common interface for both historical and live feeds

### Configuration & Setup

#### **Backtester Configuration** (`configs/backtester/`)
- `config.yaml` - Backtester settings (date ranges, initial capital, fees)
- `strategies.yaml` - Strategy configurations and parameters
- `symbols.yaml` - Symbol universe for testing

#### **Database Schema Extensions**
- `backtest_runs` table - Store backtest metadata and results
- `backtest_trades` table - Store simulated trades from backtests
- `backtest_performance` table - Store performance metrics

### Strategy Framework

#### **Strategy Base Interface**
```go
type Strategy interface {
    Initialize(ctx Context) error
    OnBar(bar BarData) []Order
    OnTrade(trade TradeEvent) error
    Cleanup() error
}
```

#### **Strategy Context**
- Portfolio state access
- Historical data queries
- Technical indicators
- Risk management rules

### Execution & Results

#### **Command Line Interface** (`cmd/backtester/`)
- `main.go` - CLI for running backtests
- Support for configuration files and command-line overrides
- Parallel execution for multiple strategies/parameters

#### **Results & Reporting** (`pkg/reporting/`)
- `metrics.go` - Calculate Sharpe ratio, max drawdown, win rate, etc.
- `charts.go` - Generate performance charts and plots
- `export.go` - Export results to CSV, JSON, HTML reports

### Integration Points

#### **Shared Components with Live Trading**
- Same strategy interface for both backtesting and live trading
- Shared order management and portfolio tracking
- Common configuration format
- Unified logging and monitoring

#### **Data Pipeline**
```
Historical Data (TimescaleDB) → Market Data Feed → Strategy → Orders → Simulated Broker → Portfolio Updates → Performance Tracking
```

### Advanced Features

#### **Multi-Asset Support**
- Portfolio-level strategies across multiple symbols
- Currency conversion and cross-asset correlations
- Sector/industry rotation strategies

#### **Risk Management**
- Position sizing algorithms
- Stop-loss and take-profit orders
- Portfolio-level risk limits
- Correlation-based risk metrics

#### **Optimization Framework**
- Parameter sweeps and grid searches
- Genetic algorithm optimization
- Walk-forward analysis
- Out-of-sample testing

## 🗄️ Database Setup

### PostgreSQL TimescaleDB

The project uses PostgreSQL with the TimescaleDB extension for efficient time series data storage.

#### Quick Start
```bash
# Start the database
cd docker
docker-compose up -d

# Or use the management script
./db-manager.sh start
```

#### Access Points
- **TimescaleDB**: `localhost:5432`
- **PgAdmin**: http://localhost:8080

#### Database Schema

**`ohlcv_data` (Hypertable)**
- Stores time series OHLCV data with automatic partitioning
- Optimized indexes for time-based queries
- Unique constraints to prevent duplicate data

**`symbols`**
- Trading symbol metadata and classification
- Support for stocks, crypto, forex, ETFs

**`trades` (Hypertable)**
- Individual trade execution records
- Both live and backtested trades

**Continuous Aggregates**
- `ohlcv_1h` - Hourly aggregates from minute data
- `ohlcv_1d` - Daily aggregates from minute data

## 🚀 Usage Examples

### Backtesting

#### Simple Backtest
```bash
./backtester -strategy moving_average -symbol AAPL -start 2024-01-01 -end 2024-12-31
```

#### Configuration-based Backtest
```bash
./backtester -config configs/backtester/ma_strategy.yaml
```

#### Parameter Optimization
```bash
./backtester -strategy moving_average -optimize -param-range "fast_ma:5-20,slow_ma:20-50"
```

### Database Operations

#### Insert OHLCV Data
```sql
INSERT INTO ohlcv_data (symbol, timeframe, timestamp, open, high, low, close, volume) 
VALUES 
    ('AAPL', '1m', '2025-06-26 10:00:00+00', 150.00, 151.50, 149.75, 150.25, 1000000),
    ('BTCUSD', '1m', '2025-06-26 10:00:00+00', 65000.00, 65500.00, 64800.00, 65200.00, 50.75);
```

#### Query Recent Data
```sql
-- Get latest OHLCV data for AAPL
SELECT * FROM ohlcv_data 
WHERE symbol = 'AAPL' 
ORDER BY timestamp DESC 
LIMIT 100;

-- Get hourly aggregates for the last 24 hours
SELECT * FROM ohlcv_1h 
WHERE symbol = 'AAPL' 
  AND bucket >= NOW() - INTERVAL '24 hours'
ORDER BY bucket DESC;
```

## 🔧 Configuration

### Environment Variables

All sensitive configuration is managed through environment variables in `.env`:

```env
# API Keys (Alpaca Paper Trading)
ALPACA_API_KEY=your_alpaca_api_key
ALPACA_SECRET_KEY=your_alpaca_secret_key

# Optional API Keys
ALPHA_VANTAGE_API_KEY=your_alpha_vantage_key
POLYGON_API_KEY=your_polygon_key
YAHOO_API_KEY=not_required_for_yahoo

# PostgreSQL/TimescaleDB Configuration
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=trading_data
POSTGRES_HOST=localhost
POSTGRES_PORT=5432

# PgAdmin Configuration
PGADMIN_EMAIL=admin@trading.com
PGADMIN_PASSWORD=admin_password
PGADMIN_PORT=8080
```

## 🎛️ Database Management

Use the provided management script for common database operations:

```bash
./docker/db-manager.sh start       # Start database services
./docker/db-manager.sh stop        # Stop database services
./docker/db-manager.sh restart     # Restart database services
./docker/db-manager.sh status      # Show service status
./docker/db-manager.sh logs        # Show logs
./docker/db-manager.sh connect     # Connect to database via psql
./docker/db-manager.sh backup      # Create database backup
./docker/db-manager.sh reset       # Reset database (WARNING: deletes all data!)
./docker/db-manager.sh help        # Show help
```

## 🔒 Security Features

- No default passwords in configuration files
- All credentials must be explicitly set in `.env`
- Database access restricted to localhost by default
- API keys stored securely in environment variables

## 🏗️ Architecture Benefits

1. **Unified Codebase**: Same strategy code works for both backtesting and live trading
2. **Database Integration**: Leverages TimescaleDB for high-performance time series data
3. **Scalable**: Handles large datasets and multiple strategies efficiently
4. **Extensible**: Easy to add new strategies, data sources, and metrics
5. **Production Ready**: Built with Go's performance and reliability
6. **Comprehensive**: Includes optimization, reporting, and risk management

## 📚 Getting Started

1. **Clone the repository**
2. **Set up environment variables** in `.env`
3. **Start the database**: `cd docker && docker-compose up -d`
4. **Build the applications**: `go build ./cmd/...`
5. **Run a backtest**: `./backtester -help`

## 🛠️ Development

### Prerequisites
- Go 1.21+
- Docker and Docker Compose
- PostgreSQL client tools (optional, for direct database access)

### Building
```bash
# Build all applications
go build ./cmd/...

# Build specific application
go build ./cmd/backtester
go build ./cmd/trader
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

## 📈 Roadmap

- [ ] Implement core backtesting engine
- [ ] Add strategy framework with common indicators
- [ ] Create example strategies (MA crossover, mean reversion, momentum)
- [ ] Build parameter optimization framework
- [ ] Add performance reporting and visualization
- [ ] Implement live trading integration
- [ ] Add real-time data feeds
- [ ] Create web dashboard for monitoring
- [ ] Add machine learning strategy framework
- [ ] Implement portfolio optimization algorithms

## 📄 License

[Add your license information here]

## 🤝 Contributing

[Add contribution guidelines here]
