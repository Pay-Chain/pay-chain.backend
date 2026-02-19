-- 000031_fix_chain_rpcs_fk.up.sql

-- 1. Drop the incorrect foreign key constraint (pointing to chains_old)
ALTER TABLE chain_rpcs DROP CONSTRAINT IF EXISTS chain_rpcs_chain_id_fkey;

-- 2. Add the correct foreign key constraint (pointing to chains)
ALTER TABLE chain_rpcs ADD CONSTRAINT chain_rpcs_chain_id_fkey 
    FOREIGN KEY (chain_id) 
    REFERENCES chains(id) 
    ON DELETE CASCADE;
