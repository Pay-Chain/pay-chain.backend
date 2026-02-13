DROP INDEX IF EXISTS idx_chain_address;
DROP INDEX IF EXISTS idx_smart_contracts_chain_type_active;
DROP INDEX IF EXISTS idx_smart_contracts_chain_id;
DROP INDEX IF EXISTS idx_smart_contracts_deleted_at;

ALTER TABLE smart_contracts
    DROP COLUMN IF EXISTS version,
    DROP COLUMN IF EXISTS deployer_address,
    DROP COLUMN IF EXISTS token0_address,
    DROP COLUMN IF EXISTS token1_address,
    DROP COLUMN IF EXISTS fee_tier,
    DROP COLUMN IF EXISTS hook_address,
    DROP COLUMN IF EXISTS start_block,
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS destination_map,
    DROP COLUMN IF EXISTS deleted_at;
