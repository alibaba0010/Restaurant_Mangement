-- Sync schema with Go models (2026-02-20)
-- Based on analysis of existing schema screenshots and Go internal models

-- 1. Create many-to-many join table for Menu Categories
CREATE TABLE IF NOT EXISTS menu_category_joins (
    menu_id UUID NOT NULL REFERENCES menu(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES menu_categories(id) ON DELETE CASCADE,
    PRIMARY KEY (menu_id, category_id)
);

-- 2. Migrate existing category data from menu table to the join table
-- This ensures no data is lost during the transition to many-to-many
INSERT INTO menu_category_joins (menu_id, category_id)
SELECT id, category_id FROM menu WHERE category_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- 3. Update Menu Table
ALTER TABLE menu ADD COLUMN IF NOT EXISTS stock_quantity INT NOT NULL DEFAULT 0;
ALTER TABLE menu ADD COLUMN IF NOT EXISTS is_vegetarian BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE menu ADD COLUMN IF NOT EXISTS is_vegan BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE menu ADD COLUMN IF NOT EXISTS is_gluten_free BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE menu ADD COLUMN IF NOT EXISTS allergens JSONB;
ALTER TABLE menu ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Now safe to drop category_id as data has been migrated
ALTER TABLE menu DROP COLUMN IF EXISTS category_id;

-- 4. Update Orders Table (Adding lifecycle tracking and payment details)
ALTER TABLE orders ALTER COLUMN total_amount TYPE DECIMAL(12, 2);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS order_type VARCHAR(50) NOT NULL DEFAULT 'delivery';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS currency VARCHAR(3) NOT NULL DEFAULT 'USD';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_status VARCHAR(50) NOT NULL DEFAULT 'pending';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_method VARCHAR(100);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_reference VARCHAR(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS paid_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS confirmed_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS preparing_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS ready_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMP WITH TIME ZONE;

-- 5. Update Order Items
ALTER TABLE order_items ALTER COLUMN menu_id SET NOT NULL;

-- 6. Update Payments Table
ALTER TABLE payments ALTER COLUMN amount TYPE DECIMAL(12, 2);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS payment_method VARCHAR(100);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS customer_email VARCHAR(255);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS customer_phone VARCHAR(20);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS customer_name VARCHAR(255);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS refund_amount DECIMAL(12, 2);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS refund_reason TEXT;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS failed_at TIMESTAMP WITH TIME ZONE;

-- Update payment_status enum with new states for production
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'processing';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'paid';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'refunded';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'refunding';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'partially_refunded';

-- 7. Create New Payment Tables (Refunds and Webhook Auditing)
CREATE TABLE IF NOT EXISTS payment_refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    provider_refund_id VARCHAR(255),
    amount DECIMAL(12, 2) NOT NULL,
    reason TEXT NOT NULL,
    status payment_status NOT NULL DEFAULT 'pending',
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS payment_webhook_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider payment_provider NOT NULL,
    payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,
    event_type VARCHAR(100) NOT NULL,
    provider_event_id VARCHAR(255) UNIQUE,
    payload JSONB NOT NULL,
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
