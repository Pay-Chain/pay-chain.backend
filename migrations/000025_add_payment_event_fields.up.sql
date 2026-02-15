ALTER TABLE payment_events ADD COLUMN IF NOT EXISTS chain_id UUID;
ALTER TABLE payment_events ADD COLUMN IF NOT EXISTS block_number BIGINT;
