-- Add deleted_at column to smart_contracts table
ALTER TABLE smart_contracts ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_smart_contracts_deleted_at ON smart_contracts(deleted_at);
