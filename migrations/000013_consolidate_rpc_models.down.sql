-- 000013_consolidate_rpc_models.down.sql

-- Remove health tracking columns from chain_rpcs table
ALTER TABLE chain_rpcs 
DROP COLUMN IF EXISTS last_error_at,
DROP COLUMN IF EXISTS error_count;

-- Recreate rpc_endpoints table (although it was unused)
CREATE TABLE IF NOT EXISTS rpc_endpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    priority INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    last_error_at TIMESTAMP,
    error_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
