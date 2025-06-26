-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- Create OHLCV data table
CREATE TABLE IF NOT EXISTS ohlcv_data (
    id BIGSERIAL,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10) NOT NULL, -- e.g., '1m', '5m', '15m', '1h', '1d'
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

-- Convert the table to a hypertable (TimescaleDB time series table)
SELECT create_hypertable('ohlcv_data', 'timestamp', 
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_ohlcv_symbol_timeframe ON ohlcv_data (symbol, timeframe, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ohlcv_timestamp ON ohlcv_data (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ohlcv_symbol ON ohlcv_data (symbol);

-- Create unique constraint to prevent duplicate data
CREATE UNIQUE INDEX IF NOT EXISTS idx_ohlcv_unique 
ON ohlcv_data (symbol, timeframe, timestamp);

-- Create table for storing trading symbols metadata
CREATE TABLE IF NOT EXISTS symbols (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100),
    exchange VARCHAR(50),
    asset_type VARCHAR(20), -- 'stock', 'crypto', 'forex', etc.
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert some common symbols
INSERT INTO symbols (symbol, name, exchange, asset_type) VALUES 
    ('AAPL', 'Apple Inc.', 'NASDAQ', 'stock'),
    ('GOOGL', 'Alphabet Inc.', 'NASDAQ', 'stock'),
    ('TSLA', 'Tesla Inc.', 'NASDAQ', 'stock'),
    ('SPY', 'SPDR S&P 500 ETF', 'NYSE', 'etf'),
    ('BTCUSD', 'Bitcoin USD', 'CRYPTO', 'crypto'),
    ('ETHUSD', 'Ethereum USD', 'CRYPTO', 'crypto')
ON CONFLICT (symbol) DO NOTHING;

-- Create table for trade executions
CREATE TABLE IF NOT EXISTS trades (
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

-- Convert trades table to hypertable
SELECT create_hypertable('trades', 'timestamp', 
    chunk_time_interval => INTERVAL '1 week',
    if_not_exists => TRUE);

-- Create indexes for trades table
CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades (symbol, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_trades_timestamp ON trades (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_trades_order_id ON trades (order_id);

-- Create continuous aggregates for common time periods (optional but useful for performance)
-- 1-hour aggregates from minute data
CREATE MATERIALIZED VIEW IF NOT EXISTS ohlcv_1h
WITH (timescaledb.continuous) AS
SELECT 
    symbol,
    time_bucket('1 hour', timestamp) as bucket,
    first(open, timestamp) as open,
    max(high) as high,
    min(low) as low,
    last(close, timestamp) as close,
    sum(volume) as volume
FROM ohlcv_data 
WHERE timeframe = '1m'
GROUP BY symbol, bucket;

-- Daily aggregates from minute data
CREATE MATERIALIZED VIEW IF NOT EXISTS ohlcv_1d
WITH (timescaledb.continuous) AS
SELECT 
    symbol,
    time_bucket('1 day', timestamp) as bucket,
    first(open, timestamp) as open,
    max(high) as high,
    min(low) as low,
    last(close, timestamp) as close,
    sum(volume) as volume
FROM ohlcv_data 
WHERE timeframe = '1m'
GROUP BY symbol, bucket;

-- Set up refresh policies for continuous aggregates
SELECT add_continuous_aggregate_policy('ohlcv_1h',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

SELECT add_continuous_aggregate_policy('ohlcv_1d',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day');

-- Create function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at columns
CREATE TRIGGER update_ohlcv_updated_at BEFORE UPDATE ON ohlcv_data
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_symbols_updated_at BEFORE UPDATE ON symbols
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create data retention policy (optional - keeps data for 2 years)
-- SELECT add_retention_policy('ohlcv_data', INTERVAL '2 years');
-- SELECT add_retention_policy('trades', INTERVAL '2 years');

PRINT 'TimescaleDB OHLCV database initialized successfully!';
