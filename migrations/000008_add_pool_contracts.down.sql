-- 000008_add_pool_contracts.down.sql

ALTER TABLE smart_contracts
DROP COLUMN hook_address,
DROP COLUMN fee_tier,
DROP COLUMN token1_address,
DROP COLUMN token0_address;
