-- Remove updated_at from payment_webhook_logs (2026-03-15)
ALTER TABLE payment_webhook_logs DROP COLUMN IF EXISTS updated_at;
