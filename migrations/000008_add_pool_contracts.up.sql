-- 000008_add_pool_contracts.up.sql

ALTER TABLE smart_contracts
ADD COLUMN token0_address VARCHAR(66),
ADD COLUMN token1_address VARCHAR(66),
ADD COLUMN fee_tier INTEGER,
ADD COLUMN hook_address VARCHAR(66);

CREATE INDEX idx_smart_contracts_token0 ON smart_contracts(token0_address);
CREATE INDEX idx_smart_contracts_token1 ON smart_contracts(token1_address);
