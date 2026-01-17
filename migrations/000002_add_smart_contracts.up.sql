-- 000002_add_smart_contracts.up.sql
-- Smart Contracts table for tracking deployed contracts

CREATE TABLE smart_contracts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    chain_id VARCHAR(50) NOT NULL,  -- CAIP-2 format: namespace:chainId (e.g., eip155:84532)
    contract_address VARCHAR(66) NOT NULL,
    abi JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    UNIQUE(chain_id, contract_address)
);

-- Indexes
CREATE INDEX idx_smart_contracts_chain_id ON smart_contracts(chain_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_smart_contracts_address ON smart_contracts(contract_address) WHERE deleted_at IS NULL;
CREATE INDEX idx_smart_contracts_deleted_at ON smart_contracts(deleted_at) WHERE deleted_at IS NULL;

-- Comments
COMMENT ON TABLE smart_contracts IS 'Stores deployed smart contract information including ABI';
COMMENT ON COLUMN smart_contracts.chain_id IS 'CAIP-2 formatted chain ID (e.g., eip155:84532 for Base Sepolia)';
COMMENT ON COLUMN smart_contracts.abi IS 'Full contract ABI in JSON format';
