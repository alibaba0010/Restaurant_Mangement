-- Rollback: restore total_amount to original subtotal, then drop columns
UPDATE orders SET total_amount = subtotal WHERE subtotal > 0;

ALTER TABLE orders DROP COLUMN IF EXISTS service_charge;
ALTER TABLE orders DROP COLUMN IF EXISTS subtotal;
