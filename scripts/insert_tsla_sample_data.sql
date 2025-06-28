-- Insert sample OHLCV data for TSLA
-- Date range: 2025-01-02 09:30:00+00 to 2025-01-02 11:09:00+00
-- Timeframe: 1-minute bars
-- This represents intraday trading data for TSLA

-- First, ensure TSLA is in the symbols table
INSERT INTO symbols (symbol, name, exchange, asset_type) 
VALUES ('TSLA', 'Tesla Inc', 'NASDAQ', 'stock') 
ON CONFLICT (symbol) DO NOTHING;

-- Generate 1-minute OHLCV data for TSLA
-- Starting price around $400, with realistic intraday movements
WITH tsla_data AS (
  SELECT 
    generate_series(
      '2025-01-02 09:30:00+00'::timestamptz,
      '2025-01-02 11:09:00+00'::timestamptz,
      '1 minute'::interval
    ) AS timestamp
),
price_movements AS (
  SELECT 
    timestamp,
    -- Generate a base price that trends slightly upward with random variations
    400.00 + 
    (EXTRACT(EPOCH FROM timestamp - '2025-01-02 09:30:00+00'::timestamptz) / 3600.0) * 2.5 + -- Slight upward trend
    (random() - 0.5) * 8.0 + -- Random variation of ±$4
    sin(EXTRACT(EPOCH FROM timestamp) / 300.0) * 1.5 AS base_price, -- 5-minute cycle variation
    random() AS rand1,
    random() AS rand2,
    random() AS rand3,
    random() AS rand4
  FROM tsla_data
),
ohlcv_calculated AS (
  SELECT 
    timestamp,
    base_price,
    -- Calculate OHLC based on base price with realistic spreads
    (base_price + (rand1 - 0.5) * 0.5)::numeric(20,2) AS open,
    (base_price + rand2 * 1.2)::numeric(20,2) AS high,
    (base_price - rand3 * 1.2)::numeric(20,2) AS low,
    (base_price + (rand4 - 0.5) * 0.5)::numeric(20,2) AS close,
    -- Generate realistic volume (higher volume during market open)
    (
      CASE 
        WHEN timestamp <= '2025-01-02 10:00:00+00' THEN 50000 + random() * 30000 -- Higher volume first 30 min
        WHEN timestamp <= '2025-01-02 10:30:00+00' THEN 30000 + random() * 20000 -- Medium volume next 30 min
        ELSE 15000 + random() * 15000 -- Lower volume after that
      END
    )::numeric(20,0) AS volume
  FROM price_movements
),
ohlcv_final AS (
  SELECT 
    'TSLA' AS symbol,
    '1m' AS timeframe,
    timestamp,
    open,
    GREATEST(open, close, high) AS high, -- Ensure high is actually the highest
    LEAST(open, close, low) AS low,     -- Ensure low is actually the lowest
    close,
    volume
  FROM ohlcv_calculated
)
INSERT INTO ohlcv_data (symbol, timeframe, timestamp, open, high, low, close, volume)
SELECT symbol, timeframe, timestamp, open, high, low, close, volume
FROM ohlcv_final
ON CONFLICT (symbol, timeframe, timestamp) DO UPDATE SET
  open = EXCLUDED.open,
  high = EXCLUDED.high,
  low = EXCLUDED.low,
  close = EXCLUDED.close,
  volume = EXCLUDED.volume,
  updated_at = now();

-- Display summary of inserted data
SELECT 
  symbol,
  timeframe,
  COUNT(*) AS total_bars,
  MIN(timestamp) AS start_time,
  MAX(timestamp) AS end_time,
  MIN(low) AS min_price,
  MAX(high) AS max_price,
  AVG(volume) AS avg_volume
FROM ohlcv_data 
WHERE symbol = 'TSLA' 
  AND timeframe = '1m'
  AND timestamp BETWEEN '2025-01-02 09:30:00+00' AND '2025-01-02 11:09:00+00'
GROUP BY symbol, timeframe;
