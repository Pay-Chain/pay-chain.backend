-- 000013_consolidate_rpc_models.up.sql

-- Add health tracking columns to chain_rpcs table
ALTER TABLE chain_rpcs 
ADD COLUMN IF NOT EXISTS last_error_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS error_count INTEGER DEFAULT 0;

-- Drop the unused rpc_endpoints table if it exists
DROP TABLE IF EXISTS rpc_endpoints;
