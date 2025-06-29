-- Add more historical data for 2025-01-02 to support Support & Resistance strategy testing
-- This extends the existing data from 11:09 AM to 4:00 PM (market close)

-- First, let's see what we have
SELECT 'Current AAPL data:' as info, COUNT(*) as count, MIN(timestamp), MAX(timestamp) 
FROM ohlcv_data WHERE symbol = 'AAPL' AND DATE(timestamp) = '2025-01-02';

-- Generate extended AAPL data from 11:10 AM to 4:00 PM (290 more bars)
-- We'll create realistic price movements with support around 150-151 and resistance around 155-156

DO $$
DECLARE
    start_time TIMESTAMP WITH TIME ZONE := '2025-01-02 11:10:00+00';
    end_time TIMESTAMP WITH TIME ZONE := '2025-01-02 21:00:00+00'; -- 4:00 PM EST = 21:00 UTC
    current_time TIMESTAMP WITH TIME ZONE;
    base_price FLOAT := 152.0; -- Starting from last known price
    price_open FLOAT;
    price_high FLOAT;
    price_low FLOAT;
    price_close FLOAT;
    volume_val FLOAT;
    trend_direction INT := 1; -- 1 for up, -1 for down
    bars_in_trend INT := 0;
    support_level FLOAT := 150.5;
    resistance_level FLOAT := 155.5;
    counter INT := 0;
BEGIN
    current_time := start_time;
    price_close := base_price;
    
    WHILE current_time <= end_time LOOP
        counter := counter + 1;
        bars_in_trend := bars_in_trend + 1;
        
        -- Change trend occasionally
        IF bars_in_trend > (5 + (RANDOM() * 15)::INT) THEN
            trend_direction := trend_direction * -1;
            bars_in_trend := 0;
        END IF;
        
        -- Create price action with support/resistance respect
        price_open := price_close;
        
        -- Basic trend movement
        IF trend_direction = 1 THEN
            price_close := price_open + (RANDOM() * 0.8 + 0.1); -- Up 0.1 to 0.9
        ELSE
            price_close := price_open - (RANDOM() * 0.8 + 0.1); -- Down 0.1 to 0.9
        END IF;
        
        -- Respect support level (bounce)
        IF price_close <= support_level THEN
            price_close := support_level + (RANDOM() * 0.5 + 0.1);
            trend_direction := 1; -- Force upward after support touch
            bars_in_trend := 0;
        END IF;
        
        -- Respect resistance level (rejection)
        IF price_close >= resistance_level THEN
            -- Sometimes break through (20% chance), sometimes reject
            IF RANDOM() < 0.2 THEN
                price_close := resistance_level + (RANDOM() * 0.5 + 0.1); -- Breakout
                support_level := resistance_level; -- Previous resistance becomes support
                resistance_level := resistance_level + 2.0; -- New resistance level
            ELSE
                price_close := resistance_level - (RANDOM() * 0.5 + 0.1); -- Rejection
                trend_direction := -1; -- Force downward after resistance rejection
                bars_in_trend := 0;
            END IF;
        END IF;
        
        -- Calculate high and low
        price_high := GREATEST(price_open, price_close) + (RANDOM() * 0.3);
        price_low := LEAST(price_open, price_close) - (RANDOM() * 0.3);
        
        -- Ensure levels are logical
        price_high := GREATEST(price_high, price_open, price_close);
        price_low := LEAST(price_low, price_open, price_close);
        
        -- Generate realistic volume (higher during breakouts/breakdowns)
        volume_val := 1000000 + (RANDOM() * 2000000);
        
        -- Higher volume near support/resistance levels
        IF ABS(price_close - support_level) < 0.5 OR ABS(price_close - resistance_level) < 0.5 THEN
            volume_val := volume_val * (1.5 + RANDOM() * 1.0); -- 1.5x to 2.5x volume
        END IF;
        
        -- Insert the bar
        INSERT INTO ohlcv_data (symbol, timestamp, open, high, low, close, volume, timeframe)
        VALUES ('AAPL', current_time, price_open, price_high, price_low, price_close, volume_val, '1m');
        
        -- Move to next minute
        current_time := current_time + INTERVAL '1 minute';
    END LOOP;
    
    RAISE NOTICE 'Added % bars for AAPL', counter;
END $$;

-- Generate extended TSLA data with different support/resistance levels
DO $$
DECLARE
    start_time TIMESTAMP WITH TIME ZONE := '2025-01-02 11:10:00+00';
    end_time TIMESTAMP WITH TIME ZONE := '2025-01-02 21:00:00+00'; -- 4:00 PM EST = 21:00 UTC
    current_time TIMESTAMP WITH TIME ZONE;
    base_price FLOAT := 401.0; -- Starting from last known price
    price_open FLOAT;
    price_high FLOAT;
    price_low FLOAT;
    price_close FLOAT;
    volume_val FLOAT;
    trend_direction INT := 1; -- 1 for up, -1 for down
    bars_in_trend INT := 0;
    support_level FLOAT := 395.0;
    resistance_level FLOAT := 410.0;
    counter INT := 0;
BEGIN
    current_time := start_time;
    price_close := base_price;
    
    WHILE current_time <= end_time LOOP
        counter := counter + 1;
        bars_in_trend := bars_in_trend + 1;
        
        -- Change trend occasionally
        IF bars_in_trend > (8 + (RANDOM() * 20)::INT) THEN
            trend_direction := trend_direction * -1;
            bars_in_trend := 0;
        END IF;
        
        -- Create price action with support/resistance respect
        price_open := price_close;
        
        -- Basic trend movement (TSLA more volatile)
        IF trend_direction = 1 THEN
            price_close := price_open + (RANDOM() * 2.0 + 0.2); -- Up 0.2 to 2.2
        ELSE
            price_close := price_open - (RANDOM() * 2.0 + 0.2); -- Down 0.2 to 2.2
        END IF;
        
        -- Respect support level (bounce)
        IF price_close <= support_level THEN
            price_close := support_level + (RANDOM() * 1.0 + 0.3);
            trend_direction := 1; -- Force upward after support touch
            bars_in_trend := 0;
        END IF;
        
        -- Respect resistance level (rejection)
        IF price_close >= resistance_level THEN
            -- Sometimes break through (25% chance), sometimes reject
            IF RANDOM() < 0.25 THEN
                price_close := resistance_level + (RANDOM() * 2.0 + 0.5); -- Breakout
                support_level := resistance_level - 2.0; -- Adjust support
                resistance_level := resistance_level + 8.0; -- New resistance level
            ELSE
                price_close := resistance_level - (RANDOM() * 1.5 + 0.3); -- Rejection
                trend_direction := -1; -- Force downward after resistance rejection
                bars_in_trend := 0;
            END IF;
        END IF;
        
        -- Calculate high and low
        price_high := GREATEST(price_open, price_close) + (RANDOM() * 1.0);
        price_low := LEAST(price_open, price_close) - (RANDOM() * 1.0);
        
        -- Ensure levels are logical
        price_high := GREATEST(price_high, price_open, price_close);
        price_low := LEAST(price_low, price_open, price_close);
        
        -- Generate realistic volume
        volume_val := 800000 + (RANDOM() * 1500000);
        
        -- Higher volume near support/resistance levels
        IF ABS(price_close - support_level) < 1.0 OR ABS(price_close - resistance_level) < 1.0 THEN
            volume_val := volume_val * (1.3 + RANDOM() * 0.8); -- 1.3x to 2.1x volume
        END IF;
        
        -- Insert the bar
        INSERT INTO ohlcv_data (symbol, timestamp, open, high, low, close, volume, timeframe)
        VALUES ('TSLA', current_time, price_open, price_high, price_low, price_close, volume_val, '1m');
        
        -- Move to next minute
        current_time := current_time + INTERVAL '1 minute';
    END LOOP;
    
    RAISE NOTICE 'Added % bars for TSLA', counter;
END $$;

-- Show final counts
SELECT 'Final AAPL data:' as info, COUNT(*) as count, MIN(timestamp), MAX(timestamp) 
FROM ohlcv_data WHERE symbol = 'AAPL' AND DATE(timestamp) = '2025-01-02';

SELECT 'Final TSLA data:' as info, COUNT(*) as count, MIN(timestamp), MAX(timestamp) 
FROM ohlcv_data WHERE symbol = 'TSLA' AND DATE(timestamp) = '2025-01-02';
