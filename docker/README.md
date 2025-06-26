# JonBuhTrader PostgreSQL TimescaleDB Setup

This directory contains the Docker configuration for a PostgreSQL TimescaleDB database optimized for storing OHLCV (Open, High, Low, Close, Volume) financial time series data.

## 🚀 Quick Start

### Prerequisites
- Docker and Docker Compose installed
- Environment variables configured in your main `.env` file

### Starting the Database

```bash
# From the project root
cd docker
docker-compose up -d

# Or using the management script
./db-manager.sh start
```

### Accessing the Database

**TimescaleDB (PostgreSQL)**
- Host: `localhost`
- Port: `5432` (configurable via `POSTGRES_PORT`)
- Database: `trading_data` (configurable via `POSTGRES_DB`)
- Username: `postgres` (configurable via `POSTGRES_USER`)
- Password: Set in `.env` file (`POSTGRES_PASSWORD`)

**PgAdmin (Web Interface)**
- URL: http://localhost:8080 (configurable via `PGADMIN_PORT`)
- Email: Set in `.env` file (`PGADMIN_EMAIL`)
- Password: Set in `.env` file (`PGADMIN_PASSWORD`)

## 📊 Database Schema

### Tables

#### `ohlcv_data` (Hypertable)
Stores time series OHLCV data with automatic partitioning by time.

```sql
CREATE TABLE ohlcv_data (
    id BIGSERIAL,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10) NOT NULL, -- '1m', '5m', '15m', '1h', '1d', etc.
    timestamp TIMESTAMPTZ NOT NULL,
    open DECIMAL(20, 8) NOT NULL,
    high DECIMAL(20, 8) NOT NULL,
    low DECIMAL(20, 8) NOT NULL,
    close DECIMAL(20, 8) NOT NULL,
    volume DECIMAL(20, 8) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, timestamp)
);
```

**Indexes:**
- `idx_ohlcv_symbol_timeframe`: (symbol, timeframe, timestamp DESC)
- `idx_ohlcv_timestamp`: (timestamp DESC)
- `idx_ohlcv_symbol`: (symbol)
- `idx_ohlcv_unique`: UNIQUE (symbol, timeframe, timestamp)

#### `symbols`
Stores metadata about trading symbols.

```sql
CREATE TABLE symbols (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100),
    exchange VARCHAR(50),
    asset_type VARCHAR(20), -- 'stock', 'crypto', 'forex', etc.
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### `trades` (Hypertable)
Stores individual trade executions.

```sql
CREATE TABLE trades (
    id BIGSERIAL,
    symbol VARCHAR(20) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    side VARCHAR(4) NOT NULL, -- 'BUY' or 'SELL'
    quantity DECIMAL(20, 8) NOT NULL,
    price DECIMAL(20, 8) NOT NULL,
    total_value DECIMAL(20, 8) NOT NULL,
    commission DECIMAL(20, 8) DEFAULT 0,
    order_id VARCHAR(50),
    strategy VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, timestamp)
);
```

### Continuous Aggregates (Materialized Views)

#### `ohlcv_1h`
Hourly aggregates automatically computed from minute data.

#### `ohlcv_1d`
Daily aggregates automatically computed from minute data.

## 🛠️ Management Script

The `db-manager.sh` script provides convenient commands for managing the database:

```bash
./db-manager.sh start       # Start database services
./db-manager.sh stop        # Stop database services
./db-manager.sh restart     # Restart database services
./db-manager.sh status      # Show service status
./db-manager.sh logs        # Show logs (optionally specify service)
./db-manager.sh connect     # Connect to database via psql
./db-manager.sh backup      # Create database backup
./db-manager.sh reset       # Reset database (WARNING: deletes all data!)
./db-manager.sh help        # Show help
```

## 📝 Example Usage

### Inserting OHLCV Data

```sql
INSERT INTO ohlcv_data (symbol, timeframe, timestamp, open, high, low, close, volume) 
VALUES 
    ('AAPL', '1m', '2025-06-26 10:00:00+00', 150.00, 151.50, 149.75, 150.25, 1000000),
    ('BTCUSD', '1m', '2025-06-26 10:00:00+00', 65000.00, 65500.00, 64800.00, 65200.00, 50.75);
```

### Querying Recent Data

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

### Recording Trades

```sql
INSERT INTO trades (symbol, timestamp, side, quantity, price, total_value, order_id, strategy)
VALUES ('AAPL', NOW(), 'BUY', 100, 150.25, 15025.00, 'ORD123456', 'momentum');
```

## 🔧 Configuration

All configuration is done through environment variables in your main `.env` file:

```env
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

## 🔒 Security Notes

- No default passwords are set in docker-compose.yml
- All credentials must be explicitly configured in `.env`
- The database is only accessible from localhost by default
- Consider using strong passwords and rotating them regularly

## 🏗️ Architecture

- **TimescaleDB**: PostgreSQL extension optimized for time series data
- **Automatic Partitioning**: Data is automatically partitioned by time (1-day chunks for OHLCV, 1-week for trades)
- **Continuous Aggregates**: Pre-computed aggregations for faster queries
- **Indexing**: Optimized indexes for common query patterns
- **Data Integrity**: Unique constraints prevent duplicate data

## 📚 Useful Resources

- [TimescaleDB Documentation](https://docs.timescale.com/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
