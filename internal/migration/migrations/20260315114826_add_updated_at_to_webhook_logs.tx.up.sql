-- Add updated_at to payment_webhook_logs (2026-03-15)
ALTER TABLE payment_webhook_logs ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
