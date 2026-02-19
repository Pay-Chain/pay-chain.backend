ALTER TABLE chains ADD COLUMN IF NOT EXISTS ccip_chain_selector VARCHAR(255) DEFAULT '';
ALTER TABLE chains ADD COLUMN IF NOT EXISTS layerzero_eid INTEGER DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_chains_ccip_selector ON chains(ccip_chain_selector);
CREATE INDEX IF NOT EXISTS idx_chains_layerzero_eid ON chains(layerzero_eid);
