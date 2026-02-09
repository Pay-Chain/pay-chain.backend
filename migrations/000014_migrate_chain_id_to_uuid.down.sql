-- 000014_migrate_chain_id_to_uuid.down.sql
-- This is complex to revert perfectly because of UUID mapping, 
-- but we'll try to restore integer IDs from network_id.

-- 1. Add old integer columns
ALTER TABLE chains ADD COLUMN old_id INTEGER;
ALTER TABLE chain_rpcs ADD COLUMN old_chain_id INTEGER;
ALTER TABLE supported_tokens ADD COLUMN old_chain_id INTEGER;
ALTER TABLE wallets ADD COLUMN old_chain_id INTEGER;
ALTER TABLE bridge_configs ADD COLUMN old_source_chain_id INTEGER;
ALTER TABLE bridge_configs ADD COLUMN old_dest_chain_id INTEGER;
ALTER TABLE fee_configs ADD COLUMN old_chain_id INTEGER;

-- 2. Populate old columns from network_id
-- This assumes network_id was previously the integer id.
UPDATE chains SET old_id = CAST(network_id AS INTEGER);

UPDATE chain_rpcs cr SET old_chain_id = c.old_id FROM chains c WHERE cr.chain_id = c.id;
UPDATE supported_tokens st SET old_chain_id = c.old_id FROM chains c WHERE st.chain_id = c.id;
UPDATE wallets w SET old_chain_id = c.old_id FROM chains c WHERE w.chain_id = c.id;
UPDATE bridge_configs bc SET old_source_chain_id = c.old_id FROM chains c WHERE bc.source_chain_id = c.id;
UPDATE bridge_configs bc SET old_source_chain_id = c.old_id FROM chains c WHERE bc.dest_chain_id = c.id;
UPDATE fee_configs fc SET old_chain_id = c.old_id FROM chains c WHERE fc.chain_id = c.id;

-- 3. Drop constraints
ALTER TABLE chain_rpcs DROP CONSTRAINT IF EXISTS chain_rpcs_chain_id_fkey;
ALTER TABLE supported_tokens DROP CONSTRAINT IF EXISTS supported_tokens_chain_id_fkey;
ALTER TABLE wallets DROP CONSTRAINT IF EXISTS wallets_chain_id_fkey;
ALTER TABLE bridge_configs DROP CONSTRAINT IF EXISTS bridge_configs_source_chain_id_fkey;
ALTER TABLE bridge_configs DROP CONSTRAINT IF EXISTS bridge_configs_dest_chain_id_fkey;
ALTER TABLE fee_configs DROP CONSTRAINT IF EXISTS fee_configs_chain_id_fkey;
ALTER TABLE supported_tokens DROP CONSTRAINT IF EXISTS supported_tokens_chain_id_token_id_key;

ALTER TABLE chains DROP CONSTRAINT IF EXISTS chains_pkey CASCADE;
ALTER TABLE chains DROP CONSTRAINT IF EXISTS chains_network_id_unique;

-- 4. Drop UUID columns
ALTER TABLE chains DROP COLUMN id;
ALTER TABLE chain_rpcs DROP COLUMN chain_id;
ALTER TABLE supported_tokens DROP COLUMN chain_id;
ALTER TABLE wallets DROP COLUMN chain_id;
ALTER TABLE bridge_configs DROP COLUMN source_chain_id;
ALTER TABLE bridge_configs DROP COLUMN dest_chain_id;
ALTER TABLE fee_configs DROP COLUMN chain_id;

-- 5. Rename old columns back
ALTER TABLE chains RENAME COLUMN old_id TO id;
ALTER TABLE chain_rpcs RENAME COLUMN old_chain_id TO chain_id;
ALTER TABLE supported_tokens RENAME COLUMN old_chain_id TO chain_id;
ALTER TABLE wallets RENAME COLUMN old_chain_id TO chain_id;
ALTER TABLE bridge_configs RENAME COLUMN old_source_chain_id TO source_chain_id;
ALTER TABLE bridge_configs RENAME COLUMN old_dest_chain_id TO dest_chain_id;
ALTER TABLE fee_configs RENAME COLUMN old_chain_id TO chain_id;

-- 6. Restore constraints
ALTER TABLE chains ADD PRIMARY KEY (id);
ALTER TABLE chain_rpcs ADD CONSTRAINT chain_rpcs_chain_id_fkey FOREIGN KEY (chain_id) REFERENCES chains(id) ON DELETE CASCADE;
ALTER TABLE supported_tokens ADD CONSTRAINT supported_tokens_chain_id_fkey FOREIGN KEY (chain_id) REFERENCES chains(id);
ALTER TABLE wallets ADD CONSTRAINT wallets_chain_id_fkey FOREIGN KEY (chain_id) REFERENCES chains(id);
ALTER TABLE bridge_configs ADD CONSTRAINT bridge_configs_source_chain_id_fkey FOREIGN KEY (source_chain_id) REFERENCES chains(id);
ALTER TABLE bridge_configs ADD CONSTRAINT bridge_configs_dest_chain_id_fkey FOREIGN KEY (dest_chain_id) REFERENCES chains(id);
ALTER TABLE fee_configs ADD CONSTRAINT fee_configs_chain_id_fkey FOREIGN KEY (chain_id) REFERENCES chains(id);
ALTER TABLE supported_tokens ADD CONSTRAINT supported_tokens_chain_id_token_id_key UNIQUE(chain_id, token_id);

-- 7. Drop network_id
ALTER TABLE chains DROP COLUMN network_id;
