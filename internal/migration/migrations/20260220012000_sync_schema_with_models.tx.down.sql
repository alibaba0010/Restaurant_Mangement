-- Sync schema with models (2026-02-20) - Down

-- 8. Drop payment_webhook_logs
DROP TABLE IF EXISTS payment_webhook_logs;

-- 7. Drop payment_refunds
DROP TABLE IF EXISTS payment_refunds;

-- 6. Revert payments updates (cannot easily revert enum values in Postgres < 12, but we can leave them)
ALTER TABLE payments DROP COLUMN IF EXISTS payment_method;
ALTER TABLE payments DROP COLUMN IF EXISTS customer_email;
ALTER TABLE payments DROP COLUMN IF EXISTS customer_phone;
ALTER TABLE payments DROP COLUMN IF EXISTS customer_name;
ALTER TABLE payments DROP COLUMN IF EXISTS refund_amount;
ALTER TABLE payments DROP COLUMN IF EXISTS refund_reason;
ALTER TABLE payments DROP COLUMN IF EXISTS completed_at;
ALTER TABLE payments DROP COLUMN IF EXISTS failed_at;
ALTER TABLE payments ALTER COLUMN amount TYPE NUMERIC(10, 2);

-- 5. Revert order items updates
ALTER TABLE order_items ALTER COLUMN menu_id DROP NOT NULL;

-- 4. Revert orders updates
ALTER TABLE orders DROP COLUMN IF EXISTS order_type;
ALTER TABLE orders DROP COLUMN IF EXISTS currency;
ALTER TABLE orders DROP COLUMN IF EXISTS payment_status;
ALTER TABLE orders DROP COLUMN IF EXISTS payment_method;
ALTER TABLE orders DROP COLUMN IF EXISTS payment_reference;
ALTER TABLE orders DROP COLUMN IF EXISTS paid_at;
ALTER TABLE orders DROP COLUMN IF EXISTS confirmed_at;
ALTER TABLE orders DROP COLUMN IF EXISTS preparing_at;
ALTER TABLE orders DROP COLUMN IF EXISTS ready_at;
ALTER TABLE orders DROP COLUMN IF EXISTS completed_at;
ALTER TABLE orders DROP COLUMN IF EXISTS cancelled_at;
ALTER TABLE orders ALTER COLUMN total_amount TYPE DECIMAL(10, 2);

-- 3. Drop menu_category_joins
DROP TABLE IF EXISTS menu_category_joins;

-- 2. Revert menu updates
ALTER TABLE menu ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES menu_categories(id) ON DELETE SET NULL;
ALTER TABLE menu DROP COLUMN IF EXISTS stock_quantity;
ALTER TABLE menu DROP COLUMN IF EXISTS is_vegetarian;
ALTER TABLE menu DROP COLUMN IF EXISTS is_vegan;
ALTER TABLE menu DROP COLUMN IF EXISTS is_gluten_free;
ALTER TABLE menu DROP COLUMN IF EXISTS allergens;
ALTER TABLE menu DROP COLUMN IF EXISTS deleted_at;

-- 1. Revert restaurants updates
ALTER TABLE restaurants DROP COLUMN IF EXISTS latitude;
ALTER TABLE restaurants DROP COLUMN IF EXISTS longitude;

-- 0. Revert Payment column renames
DO $$ 
BEGIN 
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='payments' AND column_name='reference') THEN 
        ALTER TABLE payments RENAME COLUMN reference TO transaction_ref; 
    END IF; 
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='payments' AND column_name='external_reference') THEN 
        ALTER TABLE payments RENAME COLUMN external_reference TO provider_reference; 
    END IF; 
END $$;
