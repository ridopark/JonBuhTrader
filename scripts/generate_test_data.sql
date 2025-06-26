-- Clear existing AAPL data 
DELETE FROM ohlcv_data WHERE symbol = 'AAPL';

-- Insert test data using generate_series
INSERT INTO ohlcv_data (symbol, timestamp, open, high, low, close, volume, timeframe)
SELECT 
    'AAPL' as symbol,
    '2025-01-02 09:30:00'::timestamp + (n || ' minutes')::interval as timestamp,
    150.0 + (n * 0.05) + (sin(n * 0.1) * 1.5) + ((random() - 0.5) * 0.8) as open,
    150.0 + (n * 0.05) + (sin(n * 0.1) * 1.5) + ((random() - 0.5) * 0.8) + (random() * 0.3) as high,
    150.0 + (n * 0.05) + (sin(n * 0.1) * 1.5) + ((random() - 0.5) * 0.8) - (random() * 0.3) as low,
    150.0 + (n * 0.05) + (sin(n * 0.1) * 1.5) + ((random() - 0.5) * 0.8) + ((random() - 0.5) * 0.5) as close,
    1000000 + (random() * 200000)::integer as volume,
    '1m' as timeframe
FROM generate_series(0, 99) as n;
