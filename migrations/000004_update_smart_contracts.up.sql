-- 000004_update_smart_contracts.up.sql
-- Add missing columns for full Smart Contract management

ALTER TABLE smart_contracts 
ADD COLUMN type VARCHAR(50) NOT NULL DEFAULT 'GATEWAY',
ADD COLUMN version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
ADD COLUMN deployer_address VARCHAR(66),
ADD COLUMN start_block BIGINT DEFAULT 0,
ADD COLUMN metadata JSONB DEFAULT '{}'::jsonb,
ADD COLUMN is_active BOOLEAN DEFAULT true;

-- Generic ABI support (already in 000002, but ensuring it's not null/valid)
-- ALTER TABLE smart_contracts ALTER COLUMN abi SET DEFAULT '{}'::jsonb;

-- Indexes for efficient lookup by type (e.g., GetGateway, GetRouter)
CREATE INDEX idx_smart_contracts_type ON smart_contracts(type) WHERE deleted_at IS NULL;
CREATE INDEX idx_smart_contracts_active ON smart_contracts(is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_smart_contracts_lookup ON smart_contracts(chain_id, type, is_active) WHERE deleted_at IS NULL;
