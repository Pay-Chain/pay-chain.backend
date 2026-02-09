-- 000014_migrate_chain_id_to_uuid.up.sql

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Add network_id column to chains
ALTER TABLE chains ADD COLUMN IF NOT EXISTS network_id VARCHAR(50);

-- 2. Populate network_id with current integer id
UPDATE chains SET network_id = CAST(id AS VARCHAR) WHERE network_id IS NULL;

-- 3. Add new UUID columns for IDs and FKs
ALTER TABLE chains ADD COLUMN new_id UUID DEFAULT uuid_generate_v4();
ALTER TABLE chain_rpcs ADD COLUMN new_chain_id UUID;
ALTER TABLE supported_tokens ADD COLUMN new_chain_id UUID;
ALTER TABLE wallets ADD COLUMN new_chain_id UUID;
ALTER TABLE bridge_configs ADD COLUMN new_source_chain_id UUID;
ALTER TABLE bridge_configs ADD COLUMN new_dest_chain_id UUID;
ALTER TABLE fee_configs ADD COLUMN new_chain_id UUID;

-- 4. Map old integer IDs to new UUIDs
UPDATE chains SET new_id = uuid_generate_v4(); -- Ensure unique UUIDs

-- Update FKs in referencing tables
UPDATE chain_rpcs cr SET new_chain_id = c.new_id FROM chains c WHERE cr.chain_id = c.id;
UPDATE supported_tokens st SET new_chain_id = c.new_id FROM chains c WHERE st.chain_id = c.id;
UPDATE wallets w SET new_chain_id = c.new_id FROM chains c WHERE w.chain_id = c.id;
UPDATE bridge_configs bc SET new_source_chain_id = c.new_id FROM chains c WHERE bc.source_chain_id = c.id;
UPDATE bridge_configs bc SET new_dest_chain_id = c.new_id FROM chains c WHERE bc.dest_chain_id = c.id;
UPDATE fee_configs fc SET new_chain_id = c.new_id FROM chains c WHERE fc.chain_id = c.id;

-- 5. Drop old constraints and columns
-- Note: We need to drop constraints first. We'll identify and drop them.
ALTER TABLE chain_rpcs DROP CONSTRAINT IF EXISTS chain_rpcs_chain_id_fkey;
ALTER TABLE supported_tokens DROP CONSTRAINT IF EXISTS supported_tokens_chain_id_fkey;
ALTER TABLE wallets DROP CONSTRAINT IF EXISTS wallets_chain_id_fkey;
ALTER TABLE bridge_configs DROP CONSTRAINT IF EXISTS bridge_configs_source_chain_id_fkey;
ALTER TABLE bridge_configs DROP CONSTRAINT IF EXISTS bridge_configs_dest_chain_id_fkey;
ALTER TABLE fee_configs DROP CONSTRAINT IF EXISTS fee_configs_chain_id_fkey;

-- Drop PK from chains
ALTER TABLE chains DROP CONSTRAINT IF EXISTS chains_pkey CASCADE;

-- Drop old columns
ALTER TABLE chains DROP COLUMN id;
ALTER TABLE chain_rpcs DROP COLUMN chain_id;
ALTER TABLE supported_tokens DROP COLUMN chain_id;
ALTER TABLE wallets DROP COLUMN chain_id;
ALTER TABLE bridge_configs DROP COLUMN source_chain_id;
ALTER TABLE bridge_configs DROP COLUMN dest_chain_id;
ALTER TABLE fee_configs DROP COLUMN chain_id;

-- 6. Rename new columns to old names
ALTER TABLE chains RENAME COLUMN new_id TO id;
ALTER TABLE chain_rpcs RENAME COLUMN new_chain_id TO chain_id;
ALTER TABLE supported_tokens RENAME COLUMN new_chain_id TO chain_id;
ALTER TABLE wallets RENAME COLUMN new_chain_id TO chain_id;
ALTER TABLE bridge_configs RENAME COLUMN new_source_chain_id TO source_chain_id;
ALTER TABLE bridge_configs RENAME COLUMN new_dest_chain_id TO dest_chain_id;
ALTER TABLE fee_configs RENAME COLUMN new_chain_id TO chain_id;

-- 7. Restore constraints
ALTER TABLE chains ADD PRIMARY KEY (id);
ALTER TABLE chains ALTER COLUMN network_id SET NOT NULL;
ALTER TABLE chains ADD CONSTRAINT chains_network_id_unique UNIQUE (network_id);

ALTER TABLE chain_rpcs ADD CONSTRAINT chain_rpcs_chain_id_fkey FOREIGN KEY (chain_id) REFERENCES chains(id) ON DELETE CASCADE;
ALTER TABLE supported_tokens ADD CONSTRAINT supported_tokens_chain_id_fkey FOREIGN KEY (chain_id) REFERENCES chains(id);
ALTER TABLE wallets ADD CONSTRAINT wallets_chain_id_fkey FOREIGN KEY (chain_id) REFERENCES chains(id);
ALTER TABLE bridge_configs ADD CONSTRAINT bridge_configs_source_chain_id_fkey FOREIGN KEY (source_chain_id) REFERENCES chains(id);
ALTER TABLE bridge_configs ADD CONSTRAINT bridge_configs_dest_chain_id_fkey FOREIGN KEY (dest_chain_id) REFERENCES chains(id);
ALTER TABLE fee_configs ADD CONSTRAINT fee_configs_chain_id_fkey FOREIGN KEY (chain_id) REFERENCES chains(id);

-- Add Unique constraint back to supported_tokens
-- Note: Check if there's a unique constraint name in 000001
-- It was UNIQUE(chain_id, token_id)
ALTER TABLE supported_tokens ADD CONSTRAINT supported_tokens_chain_id_token_id_key UNIQUE(chain_id, token_id);

-- Ensure correct types and defaults for the new columns
ALTER TABLE chain_rpcs ALTER COLUMN chain_id SET NOT NULL;
ALTER TABLE supported_tokens ALTER COLUMN chain_id SET NOT NULL;
ALTER TABLE wallets ALTER COLUMN chain_id SET NOT NULL;
ALTER TABLE bridge_configs ALTER COLUMN source_chain_id SET NOT NULL;
ALTER TABLE bridge_configs ALTER COLUMN dest_chain_id SET NOT NULL;
