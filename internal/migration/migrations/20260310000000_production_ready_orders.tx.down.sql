-- Rollback production readiness migration
DROP INDEX IF EXISTS idx_orders_payment_reference;
DROP INDEX IF EXISTS idx_orders_restaurant_id_created_at;
DROP INDEX IF EXISTS idx_orders_user_id_created_at;

ALTER TABLE orders DROP COLUMN IF EXISTS cancelled_at;
ALTER TABLE orders DROP COLUMN IF EXISTS completed_at;
ALTER TABLE orders DROP COLUMN IF EXISTS ready_at;
ALTER TABLE orders DROP COLUMN IF EXISTS preparing_at;
ALTER TABLE orders DROP COLUMN IF EXISTS confirmed_at;
ALTER TABLE orders DROP COLUMN IF EXISTS paid_at;
ALTER TABLE orders DROP COLUMN IF EXISTS currency;
ALTER TABLE orders DROP COLUMN IF EXISTS payment_reference;
ALTER TABLE orders DROP COLUMN IF EXISTS payment_method;
ALTER TABLE orders DROP COLUMN IF EXISTS payment_status;
ALTER TABLE orders DROP COLUMN IF EXISTS order_type;
