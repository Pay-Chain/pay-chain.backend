-- Remove deleted_at column from smart_contracts table
DROP INDEX IF EXISTS idx_smart_contracts_deleted_at;
ALTER TABLE smart_contracts DROP COLUMN IF EXISTS deleted_at;
