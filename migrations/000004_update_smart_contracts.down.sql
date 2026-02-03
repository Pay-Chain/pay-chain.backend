-- 000004_update_smart_contracts.down.sql

DROP INDEX IF EXISTS idx_smart_contracts_lookup;
DROP INDEX IF EXISTS idx_smart_contracts_active;
DROP INDEX IF EXISTS idx_smart_contracts_type;

ALTER TABLE smart_contracts 
DROP COLUMN IF EXISTS is_active,
DROP COLUMN IF EXISTS metadata,
DROP COLUMN IF EXISTS start_block,
DROP COLUMN IF EXISTS deployer_address,
DROP COLUMN IF EXISTS version,
DROP COLUMN IF EXISTS type;
