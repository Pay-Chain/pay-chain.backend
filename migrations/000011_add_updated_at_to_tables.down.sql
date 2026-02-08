-- Migration: 000011_add_updated_at_to_tables (Down)
-- Description: Remove updated_at column from several tables

ALTER TABLE chains DROP COLUMN IF EXISTS updated_at;
ALTER TABLE tokens DROP COLUMN IF EXISTS updated_at;
ALTER TABLE supported_tokens DROP COLUMN IF EXISTS updated_at;
ALTER TABLE wallets DROP COLUMN IF EXISTS updated_at;
