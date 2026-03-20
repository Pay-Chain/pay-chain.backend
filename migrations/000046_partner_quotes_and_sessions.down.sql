DROP INDEX IF EXISTS idx_partner_payment_sessions_quote_id;
DROP INDEX IF EXISTS idx_partner_payment_sessions_merchant_created_at;
DROP INDEX IF EXISTS idx_partner_payment_sessions_status_expires_at;
DROP TABLE IF EXISTS partner_payment_sessions;

DROP INDEX IF EXISTS idx_payment_quotes_merchant_created_at;
DROP INDEX IF EXISTS idx_payment_quotes_status_expires_at;
DROP TABLE IF EXISTS payment_quotes;
