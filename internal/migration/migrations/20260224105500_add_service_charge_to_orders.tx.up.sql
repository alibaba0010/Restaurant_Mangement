-- Add service charge columns to orders table
ALTER TABLE orders ADD COLUMN IF NOT EXISTS subtotal DECIMAL(12,2) NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS service_charge DECIMAL(12,2) NOT NULL DEFAULT 0;

-- Backfill existing orders: treat current total_amount as subtotal, recalculate
UPDATE orders SET
    subtotal = total_amount,
    service_charge = CASE
        WHEN total_amount < 100 THEN ROUND(total_amount * 0.10, 2)
        ELSE ROUND(total_amount * 0.05, 2)
    END,
    total_amount = total_amount + CASE
        WHEN total_amount < 100 THEN ROUND(total_amount * 0.10, 2)
        ELSE ROUND(total_amount * 0.05, 2)
    END
WHERE subtotal = 0 AND total_amount > 0;
