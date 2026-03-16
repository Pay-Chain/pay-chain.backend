ALTER TABLE chains ADD COLUMN IF NOT EXISTS ccip_chain_selector VARCHAR(255) DEFAULT '';
ALTER TABLE chains ADD COLUMN IF NOT EXISTS stargate_eid INTEGER DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_chains_ccip_selector ON chains(ccip_chain_selector);
CREATE INDEX IF NOT EXISTS idx_chains_stargate_eid ON chains(stargate_eid);
