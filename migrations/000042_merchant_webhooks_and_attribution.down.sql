DROP INDEX IF EXISTS idx_payments_merchant_id;
ALTER TABLE payments DROP COLUMN IF EXISTS merchant_id;
DROP TABLE IF EXISTS webhook_logs;
DROP TYPE IF EXISTS webhook_delivery_status;
ALTER TABLE merchants 
DROP COLUMN IF EXISTS callback_url,
DROP COLUMN IF EXISTS webhook_secret,
DROP COLUMN IF EXISTS webhook_is_active,
DROP COLUMN IF EXISTS support_email,
DROP COLUMN IF EXISTS logo_url,
DROP COLUMN IF EXISTS webhook_metadata,
DROP COLUMN IF EXISTS metadata;
