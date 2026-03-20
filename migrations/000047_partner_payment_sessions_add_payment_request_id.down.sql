DROP INDEX IF EXISTS idx_partner_payment_sessions_payment_request_id;

ALTER TABLE partner_payment_sessions
DROP COLUMN IF EXISTS payment_request_id;
