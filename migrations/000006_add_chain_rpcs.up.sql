-- 000006_add_chain_rpcs.up.sql
-- Create table for multiple RPC URLs per chain

CREATE TABLE IF NOT EXISTS chain_rpcs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_id INTEGER NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    priority INTEGER DEFAULT 0, -- Higher number = higher priority
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chain_rpcs_chain_id ON chain_rpcs(chain_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_chain_rpcs_active ON chain_rpcs(is_active) WHERE deleted_at IS NULL;

-- Migrate existing RPC URLs from chains table to chain_rpcs
INSERT INTO chain_rpcs (chain_id, url, priority, is_active)
SELECT id, rpc_url, 10, true
FROM chains
WHERE rpc_url IS NOT NULL AND rpc_url != '';

-- Optionally, we can drop the rpc_url column from chains later, 
-- but for now we can keep it as a 'primary' cached value or just ignore it.
-- Let's NOT drop it immediately to avoid breaking legacy queries if any raw SQL is used elsewhere.
-- However, the code will update to use the relationship.
