ALTER TABLE partner_payment_sessions
ADD COLUMN IF NOT EXISTS payment_request_id UUID;

CREATE INDEX IF NOT EXISTS idx_partner_payment_sessions_payment_request_id
ON partner_payment_sessions(payment_request_id);
