-- Reconcile smart_contracts schema with current backend model/repository usage.
-- Safe to run on top of older minimal schema from 000018.

ALTER TABLE smart_contracts
    ADD COLUMN IF NOT EXISTS version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    ADD COLUMN IF NOT EXISTS deployer_address VARCHAR(255),
    ADD COLUMN IF NOT EXISTS token0_address VARCHAR(255),
    ADD COLUMN IF NOT EXISTS token1_address VARCHAR(255),
    ADD COLUMN IF NOT EXISTS fee_tier INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS hook_address VARCHAR(255),
    ADD COLUMN IF NOT EXISTS start_block BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS destination_map TEXT[] NOT NULL DEFAULT '{}'::text[],
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_smart_contracts_deleted_at ON smart_contracts(deleted_at);
CREATE INDEX IF NOT EXISTS idx_smart_contracts_chain_id ON smart_contracts(chain_id);
CREATE INDEX IF NOT EXISTS idx_smart_contracts_chain_type_active ON smart_contracts(chain_id, type, is_active);
CREATE UNIQUE INDEX IF NOT EXISTS idx_chain_address ON smart_contracts(chain_id, address);
