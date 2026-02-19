-- 000031_fix_chain_rpcs_fk.down.sql

-- 1. Drop the correct foreign key constraint
ALTER TABLE chain_rpcs DROP CONSTRAINT IF EXISTS chain_rpcs_chain_id_fkey;

-- 2. Restore the incorrect foreign key constraint (pointing to chains_old)
-- Note: This assumes chains_old still exists. If not, this might fail, 
-- but this is a revert of a fix, implying we go back to the broken state.
-- If chains_old is dropped, we can't revert meaningfuly. 
-- We'll verify existence to avoid erroring out, but logically it points to chains_old.

DO $$
BEGIN
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'chains_old') THEN
        ALTER TABLE chain_rpcs ADD CONSTRAINT chain_rpcs_chain_id_fkey 
            FOREIGN KEY (chain_id) 
            REFERENCES chains_old(id) 
            ON DELETE CASCADE;
    END IF;
END $$;
